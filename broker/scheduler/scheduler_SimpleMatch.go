package scheduler

import (
	"context"
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

		// ideally, you'd want to use available providers first, but then immediately fall back on
		// cloud offloading, when none are there. with the CloudSubmit chan rewrite, the submission
		// to the cloud can also block though, so this should also be handled inside a select case
		// with timeout ...

		// add cloud offloading queue, if it's suitable for this task
		var cloud chan *provider.AsyncTask = nil
		if s.store.CanCloudOffload(task) {
			cloud = s.store.CloudSubmit
		}

		// wrap parent context in a short timeout, to retry selection regularly
		timeout, cancel := context.WithTimeout(ctx, time.Second)

		// submit the task normally with new context
		err = dynamicSubmit(timeout, task, providers, cloud)
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
