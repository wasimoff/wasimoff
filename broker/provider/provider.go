package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"wasi.team/broker/net/transport"
	wasimoff "wasi.team/proto/v1"

	"github.com/marusama/semaphore/v2"
	"google.golang.org/protobuf/proto"
)

// Provider is a single connection initiated by a computing provider
type Provider struct {
	messenger *transport.Messenger // messenger connection to provider

	// cancellable lifetime context to signal closure upwards
	lifetime  transport.Lifetime
	closeOnce sync.Once

	// unbuffered channel to submit tasks; can be `nil` if nobody's listening
	Submit chan *AsyncTask

	// resizeable semaphore to limit number of concurrent tasks
	limiter semaphore.Semaphore
	waiting bool

	// information about the provider, to be accessed with Get()
	info map[ProviderInfoKey]string

	// hashmap of files known on this provider
	files sync.Map // map[string]struct{}
}

type ProviderInfoKey string

const (
	Name      ProviderInfoKey = "name"      // a unique name for identification
	Address   ProviderInfoKey = "address"   // remote address of transport conn
	UserAgent ProviderInfoKey = "useragent" // software and architecture info
)

// Setup a new Provider instance from a given Messenger
func NewProvider(messenger *transport.Messenger) *Provider {
	lifetime := transport.NewLifetime(context.TODO())

	// construct the provider
	provider := &Provider{
		messenger: messenger,
		lifetime:  lifetime,
		Submit:    nil, // must be setup by acceptTasks
		limiter:   semaphore.New(0),
		info:      make(map[ProviderInfoKey]string),
		files:     sync.Map{},
	}

	// set known information
	provider.info[Name] = messenger.Addr()
	provider.info[Address] = messenger.Addr()
	provider.info[UserAgent] = "unknown"

	// start listening on task channel
	go provider.acceptTasks()

	return provider
}

func (p *Provider) Get(key ProviderInfoKey) string {
	return p.info[key]
}

func (p *Provider) Waiting() bool {
	return p.waiting
}

// -------------------- closure -------------------- >>

// Returns the cause of the closure or nil if Provider isn't closed yet.
func (p *Provider) Err() error {
	return p.lifetime.Err()
}

// Returns a channel to listen for lifetime closure.
func (p *Provider) Closing() <-chan struct{} {
	return p.lifetime.Closing()
}

// Close closes the underlying messenger connection to this provider
func (p *Provider) Close(reason error) {
	if reason == nil {
		reason = transport.ErrLifetimeEnded
	}
	p.closeOnce.Do(func() {
		p.messenger.Close(fmt.Errorf("closed from Provider: %w", reason))
		p.lifetime.Cancel(reason)
	})
}

// -------------------- limiter -------------------- >>

// Get the currently running tasks according to the semaphore
func (p *Provider) CurrentTasks() int {
	return p.limiter.GetCount()
}

// Get the currently configured Limit in the task semaphore
func (p *Provider) CurrentLimit() int {
	return p.limiter.GetLimit()
}

// -------------------- task channel -------------------- >>

// Accept tasks on an unbuffered channel to submit to the Provider. Channels can
// be used in a DynamicSubmit, so calls from many different sources can be efficiently
// distributed to multiple Providers.
func (p *Provider) acceptTasks() (err error) {

	// initialize the channel
	if p.Submit == nil {
		// unbuffered on purpose, so senders can use dynamic select to submit
		p.Submit = make(chan *AsyncTask)
	}

	// close Provider if the loop ever exits
	defer p.Close(err)

	for {

		// acquire a semaphore before accepting a task
		//? off-by-one because we acquire and hold a semaphore before we even get a task
		if err = p.limiter.Acquire(p.lifetime.Context, 1); err != nil {
			// nobody to notify and nothing to free, just quit
			return err
		}
		p.waiting = true

		select {

		// Provider is closing, quit the loop
		case <-p.lifetime.Closing():
			return p.Err()

		// receive task details from channel
		case task := <-p.Submit:
			p.waiting = false

			// prerequisite checks
			if err := task.Check(); err != nil {
				task.Error = err
				task.Done()
				p.limiter.Release(1)
				continue
			}

			// run the Request in a goroutine asynchronously
			// TODO: avoid gofunc by using a second listener on a `chan *PendingCall`
			go func() {
				task.Request.GetInfo().Provider = proto.String(p.Get(Name))
				task.Request.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_BrokerTransmitProviderTask)
				task.Error = p.run(task.Context, task.Request, task.Response)
				// send cancellation event if error is due to context
				if errors.Is(task.Error, context.Canceled) {
					// don't really care for result or error here, just that it completed somehow
					_ = p.messenger.RequestSync(p.lifetime.Context, &wasimoff.Task_Cancel{
						Id:     task.Request.GetInfo().Id,
						Reason: proto.String(context.Canceled.Error()),
					}, &wasimoff.Task_Cancel{})
				}
				task.Done()
				p.limiter.Release(1)
			}()

		}

	}
}
