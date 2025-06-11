package scheduler

import (
	"context"
	"log"
	"time"

	"wasi.team/broker/provider"
	wasimoff "wasi.team/proto/v1"
)

// The SimpleMatchSelector is another simple implementation of a ProviderSelector,
// which simply yields the first available provider with the required files in its store.
type SimpleMatchSelector struct {
	store *provider.ProviderStore
}

// Create a new SimpleMatchSelector given an existing ProviderStore.
func NewSimpleMatchSelector(store *provider.ProviderStore) SimpleMatchSelector {
	return SimpleMatchSelector{store}
}

func (s *SimpleMatchSelector) selectCandidates(task *provider.AsyncTask) (candidates []*provider.Provider, err error) {

	// create a list of needed files to check with the providers
	requiredFiles := []string{}
	if r, ok := task.Request.(*wasimoff.Task_Wasip1_Request); ok {
		requiredFiles = r.GetRequiredFiles()
	}

	// find suitable candidates with free slots
	candidates = make([]*provider.Provider, 0, s.store.Size())
	s.store.Range(func(addr string, p *provider.Provider) bool {
		// check for files
		for _, file := range requiredFiles {
			if !p.Has(file) {
				// missing requirement, continue
				return true
			}
		}
		// check for availability
		if p.CurrentTasks() < p.CurrentLimit() || p.Waiting() {
			// append candidates with free capacity for tasks
			candidates = append(candidates, p)
		}
		return true
	})

	// no perfect candidates found? just fallback to the full list
	if len(candidates) == 0 {
		candidates = s.store.Values()
	}
	return

}

func (s *SimpleMatchSelector) Schedule(ctx context.Context, task *provider.AsyncTask) (err error) {
	for {

		providers, err := s.selectCandidates(task)
		if err != nil {
			return err
		}

		// while there are no providers, try to use cloud offloading
		// TODO: cloud oddloading should also use a rate-limited interface with a semaphore
		// TODO: ACCIDENTALLY SERIALIZED!
		if len(providers) == 0 && s.store.CanCloudOffload(task) {
			log.Println("no providers / cloud offloading:", task.Request.GetInfo().Id)
			err := s.store.CloudOffload(task)
			if err != nil {
				log.Printf("WARNING: CloudOffload for task %s failed: %s", *task.Request.GetInfo().Id, err.Error())
			} else {
				log.Println("done:", task.Request.GetInfo().Id)
				return nil
			}
		}

		// wrap parent context in a short timeout
		timeout, cancel := context.WithTimeout(ctx, time.Second)

		// submit the task normally with new context
		err = dynamicSubmit(timeout, task, providers)
		if err != nil && ctx.Err() == nil && timeout.Err() == err {
			// parent context not cancelled and err == our timeout,
			// so reschedule in hopes of picking up changes in provider store
			cancel()
			continue // retry
		}
		cancel()
		return err

	}
}

func (s *SimpleMatchSelector) RateTick() {
	s.store.RateTick()
}
