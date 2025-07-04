package scheduler

import (
	"context"
	"fmt"

	"wasi.team/broker/provider"
)

// The AnyFreeSelector is probably the simplest implementation of a ProviderSelector,
// which uses any free Provider without concerning itself with *any* task requrirements.
type AnyFreeSelector struct {
	store *provider.ProviderStore
}

// Create a new AnyFreeSelector given an existing ProviderStore.
func NewAnyFreeSelector(store *provider.ProviderStore) *AnyFreeSelector {
	return &AnyFreeSelector{store}
}

func (s *AnyFreeSelector) selectCandidates(_ *provider.AsyncTask) (candidates []*provider.Provider, err error) {

	// if the list is empty, return nil
	if s.store.Size() == 0 {
		err = fmt.Errorf("provider store is empty")
		return
	}

	// return all the providers ...
	return s.store.Values(), nil
}

func (s *AnyFreeSelector) Schedule(ctx context.Context, task *provider.AsyncTask) (call *provider.AsyncTask, err error) {

	providers, err := s.selectCandidates(task)
	if err != nil {
		return nil, err
	}

	err = dynamicSubmit(ctx, task, providers, nil)
	return

}
