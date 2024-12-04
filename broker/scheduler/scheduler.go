package scheduler

import (
	"context"
	"log"
	"reflect"
	"wasimoff/broker/provider"
)

// Scheduler is a generic interface which must be fulfilled by a concrete scheduler,
// i.e. the type that selects suitable providers given task information and submits the task.
type Scheduler interface {
	// The Schedule function tries to submit a Task to a suitable Provider's queue and returns the WasmTask struct
	Schedule(ctx context.Context, task *provider.AsyncWasiTask) error
	// Called on task completion to measure overall throughput
	RateTick()
}

// The Dispatcher takes a task queue and a provider selector strategy and then
// decides which task to send to which provider for computation.
func Dispatcher(selector Scheduler, queue chan *provider.AsyncWasiTask) {
	for task := range queue {

		// each task is handled in a separate goroutine
		go func(task *provider.AsyncWasiTask) {
			interceptingChannel := make(chan *provider.AsyncWasiTask, task.DoneCapacity())
			interceptedChannel := task.Intercept(interceptingChannel)

			retries := 10
			var err error
			for i := 0; i < retries; i++ {
				// schedule the task with a provider
				err = selector.Schedule(context.TODO(), task)
				if err == nil && task.Error == nil && task.Response.Error == nil {
					result := <-interceptingChannel
					if result.Error == nil && result.Response.Error == nil {
						break
					}

					log.Printf("Task %v failed no. %d, retrying", task, i)
					result.Response.Error = nil
					result.Error = nil
				} else {
					log.Printf("selector.Schedule %v failed no. %d, retrying", task, i)
					task.Response.Error = nil
					task.Error = nil
				}
			}

			// still erroneous after retries, give up
			if err != nil {
				task.Error = err
			} else {
				// otherwise signal completion to measure throughput
				selector.RateTick()
			}
			interceptedChannel <- task

		}(task)
	}
}

// dynamicSubmit uses `reflect.Select` to dynamically select a Provider to submit a task to.
// This uses the Providers' unbuffered Queue, so that a task can only be submitted to a Provider
// when it currently has free capacity, without needing to busy-loop and recheck capacity yourself.
// Based on StackOverflow answer by Dave C. on https://stackoverflow.com/a/32381409.
func dynamicSubmit(ctx context.Context, call *provider.AsyncWasiTask, providers []*provider.Provider) error {

	// setup select cases
	cases := make([]reflect.SelectCase, len(providers), len(providers)+1)
	for i, p := range providers {
		if p.Submit == nil {
			panic("provider does not have a queue")
		}
		cases[i].Chan = reflect.ValueOf(p.Submit)
		cases[i].Dir = reflect.SelectSend
		cases[i].Send = reflect.ValueOf(call)
	}

	// add context.Done as select case for timeout or cancellation
	if ctx != nil {
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(ctx.Done()),
			Dir:  reflect.SelectRecv,
		})
	}

	// select one of the queues and return the WasmCall struct
	i, _, _ := reflect.Select(cases)
	if i == len(providers) {
		// index out of bounds for providers, so it must be the ctx.Done
		return ctx.Err()
	}
	return nil

}
