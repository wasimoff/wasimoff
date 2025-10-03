package main

import "sync"

// Starter is a synchronization point for multiple Tickers that each have some
// setup but need to start at almost the exaact same time instant.
// In the parent, Add(1) for each ticker, then Wait() and Broadcast(time.Now()).
// In the tickers, perform your setup, then start := WaitForValue().
type Starter[T any] struct {
	setup  sync.WaitGroup
	signal chan struct{}
	value  T
}

func NewStarter[T any]() *Starter[T] {
	return &Starter[T]{signal: make(chan struct{})}
}

func (s *Starter[T]) Add(delta int) {
	s.setup.Add(delta)
}

func (s *Starter[T]) Wait() {
	s.setup.Wait()
}

func (s *Starter[T]) Broadcast(value T) {
	s.setup.Wait()
	s.value = value
	close(s.signal)
}

func (s *Starter[T]) WaitForValue() T {
	s.setup.Done()
	<-s.signal
	return s.value
}
