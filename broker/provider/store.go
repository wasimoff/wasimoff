package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"wasi.team/broker/config"
	"wasi.team/broker/storage"
	wasimoff "wasi.team/proto/v1"

	"github.com/marusama/semaphore/v2"
	"github.com/puzpuzpuz/xsync"
	"google.golang.org/api/idtoken"
	"google.golang.org/protobuf/proto"
)

// ProviderStore holds the currently connected providers, safe for concurrent access.
// It also keeps the list of files known to the provider in memory.
type ProviderStore struct {

	// prometheus metric gauges
	metrics *ProviderStoreMetrics

	// Providers are held in a sync.Map safe for concurrent access
	providers *xsync.MapOf[string, *Provider]

	// hold an authenticated client for cloud offloading
	// check with CanCloudOffload()
	CloudSubmit   chan *AsyncTask
	cloudClient   *http.Client
	cloudFunction string

	// Storage holds the uploaded files in memory
	Storage *storage.FileStorage

	// Broadcast is a channel to submit events for all Providers
	Broadcast chan proto.Message

	// ratecounter is used to keep track of throughput [tasks/s]
	ratecounter *RateCounter
}

// NewProviderStore properly initializes the fields in the store
func NewProviderStore(storagepath string, conf *config.Configuration) (*ProviderStore, error) {

	store := ProviderStore{
		providers:   xsync.NewMapOf[*Provider](),
		Broadcast:   make(chan proto.Message, 10),
		ratecounter: NewRateCounter(5 * time.Second),
	}

	// initialize metrics gauges
	store.initializePrometheusMetrics()

	// initialize file storage
	if storagepath == "" || storagepath == ":memory:" || strings.HasPrefix(storagepath, "memory://") {
		store.Storage = storage.NewMemoryFileStorage()
	} else if strings.HasPrefix(storagepath, "boltdb://") {
		store.Storage = storage.NewBoltFileStorage(storagepath[9:])
	} else if strings.HasPrefix(storagepath, "dirfs://") {
		store.Storage = storage.NewDirectoryFileStorage(storagepath[8:])
	} else {
		store.Storage = storage.NewDirectoryFileStorage(storagepath)
	}

	// maybe initialize cloud client
	if conf.CloudCredentials != "" && conf.CloudFunction != "" && conf.CloudConcurrency > 0 {
		client, err := idtoken.NewClient(context.Background(), conf.CloudFunction, idtoken.WithCredentialsFile(conf.CloudCredentials))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GCP cloudclient: %w", err)
		}
		store.cloudClient = client
		store.cloudFunction = conf.CloudFunction
		go store.cloudLoop(conf.CloudConcurrency)
		store.metrics.AvailableWorkers.WithLabelValues("cloud").Set(float64(conf.CloudConcurrency))
	} else {
		store.metrics.AvailableWorkers.WithLabelValues("cloud").Set(0)
	}

	// start broadcast transmitter
	go store.transmitter()

	return &store, nil
}

// ------------- broadcast events to everyone -------------

// transmitter forwards events from the chan to all Providers
func (s *ProviderStore) transmitter() {

	// send current throughput regularly
	go s.throughput(time.Second)

	// broadcast events from channel
	for event := range s.Broadcast {
		s.Range(func(_ string, p *Provider) bool {
			p.messenger.SendEvent(context.Background(), event)
			return true
		})
	}

}

// ------------- cloud offloading runner client -------------

// check if cloudclient is available and the task can be offloaded
func (s *ProviderStore) CanCloudOffload(task *AsyncTask) bool {
	// client available?
	if s.CloudSubmit == nil {
		return false
	}
	// only Wasip1 tasks supported for now
	if _, ok := task.Request.(*wasimoff.Task_Wasip1_Request); ok {
		return true
	}
	// fallback to previous behaviour by default
	return false
}

// cloudRun sends the task to the cloud function using the configured client
func (s *ProviderStore) cloudRun(ctx context.Context, request wasimoff.Task_Request, response wasimoff.Task_Response) error {
	body, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed marshalling request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cloudFunction, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("content-type", "application/proto")
	resp, err := s.cloudClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloud offloading request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading response body: %w", err)
	}
	err = proto.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("failed unmarshalling response: %w", err)
	}
	return nil
}

// throughput-limited listener loop for incoming cloud offloading requests
func (s *ProviderStore) cloudLoop(concurrency int) {

	limiter := semaphore.New(concurrency) // limit simultaneous cloud invocations
	s.CloudSubmit = make(chan *AsyncTask) // unbuffered on purpose

	for {

		// acquire a semaphore before accepting a task
		_ = limiter.Acquire(context.TODO(), 1)

		task, ok := <-s.CloudSubmit
		if !ok {
			log.Println("ERR: cloudSubmit channel closed")
			return
		}

		// prerequisite checks
		if err := task.Check(); err != nil {
			task.Error = err
			task.Done()
			limiter.Release(1)
			continue
		}

		// run the request asynchronously
		go func(limiter semaphore.Semaphore, task *AsyncTask) {
			err := s.cloudRun(task.Context, task.Request, task.Response)
			if err != nil {
				task.Error = fmt.Errorf("cloud offload failed: %w", err)
			}
			task.Done()
			limiter.Release(1)
		}(limiter, task)

	}

}

// -------------- ratecounter in tasks/second --------------

// throughput expects
func (s *ProviderStore) throughput(tick time.Duration) {
	for range time.Tick(tick) {
		tps := s.ratecounter.GetRate()

		select {
		case s.Broadcast <- &wasimoff.Event_Throughput{
			Overall: proto.Float32(float32(tps)),
			// TODO: add individual contribution
		}:
			// ok
		default: // never block
		}

	}
}

// --------------- stub methods for sync.Map ---------------

// Add a Provider to the Map.
func (s *ProviderStore) Add(provider *Provider) {
	s.providers.Store(provider.Get(Address), provider)
	log.Printf("ProviderStore: %d connected", s.Size())
	s.Broadcast <- &wasimoff.Event_ClusterInfo{Providers: proto.Uint32(uint32(s.Size()))}
}

// Remove a Provider from the Map.
func (s *ProviderStore) Remove(provider *Provider) {
	s.providers.Delete(provider.Get(Address))
	log.Printf("ProviderStore: %d connected", s.Size())
	s.Broadcast <- &wasimoff.Event_ClusterInfo{Providers: proto.Uint32(uint32(s.Size()))}
}

// Size is the current size of the Map.
func (s *ProviderStore) Size() int {
	return s.providers.Size()
}

// Load a Provider from the Map by its address.
func (s *ProviderStore) Load(addr string) *Provider {
	p, ok := s.providers.Load(addr)
	if !ok {
		return nil
	}
	return p
}

// Range will iterate over all Providers in the Map and call the given function.
// If the function returns `false`, the iteration will stop. See xsync.Map.Range()
// for more usage notes and (lack of) guarantees.
func (s *ProviderStore) Range(f func(addr string, provider *Provider) bool) {
	s.providers.Range(f)
}

// Keys will simply return the current keys (Provider addresses) of the Map.
func (s *ProviderStore) Keys() []string {
	keys := make([]string, 0, s.Size())
	s.Range(func(addr string, _ *Provider) bool {
		keys = append(keys, addr)
		return true
	})
	return keys
}

// Values will return the current values (Providers) of the Map.
func (s *ProviderStore) Values() []*Provider {
	providers := make([]*Provider, 0, s.Size())
	s.Range(func(_ string, prov *Provider) bool {
		providers = append(providers, prov)
		return true
	})
	return providers
}
