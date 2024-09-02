package scheduler

import (
	"context"
	"log"
	"time"
	"wasimoff/broker/provider"
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

func (s *SimpleMatchSelector) selectCandidates(task *provider.AsyncWasiTask) (candidates []*provider.Provider, err error) {

	// create a list of needed files to check with the providers
	targ := task.Args.Task
	requiredFiles := make([]string, 0, 2)
	if targ.Binary.Ref != nil {
		requiredFiles = append(requiredFiles, *targ.Binary.Ref)
	}
	if targ.Rootfs.Ref != nil {
		requiredFiles = append(requiredFiles, *targ.Rootfs.Ref)
	}

	// find suitable candidates
	candidates = make([]*provider.Provider, 0, s.store.Size())
	s.store.Range(func(addr string, p *provider.Provider) bool {
		// check for files
		for _, file := range requiredFiles {
			if !p.Has(file) {
				// missing requirement, continue
				return true
			}
		}
		// append candidate
		candidates = append(candidates, p)
		return true
	})

	// no candidates found?
	if len(candidates) == 0 {
		log.Printf("Task %s couldn't find a Provider which satisfies: %v", task.Args.Info.TaskID(), requiredFiles)
		// err = fmt.Errorf("no suitable provider found which satisfies all requirements")
	}
	return

}

func (s *SimpleMatchSelector) Schedule(ctx context.Context, task *provider.AsyncWasiTask) (err error) {
	for {

		providers, err := s.selectCandidates(task)
		if err != nil {
			return err
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

func (s *SimpleMatchSelector) TaskDone() {
	s.store.RateTick()
}
