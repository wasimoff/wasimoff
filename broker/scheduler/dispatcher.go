package scheduler

import (
	"context"
	"errors"
	"log"
	"reflect"

	"wasi.team/broker/provider"
)

// reuseable task queue for HTTP handler and websocket
var TaskQueue = make(chan *provider.AsyncTask, 2048)

// Scheduler is a generic interface which must be fulfilled by a concrete scheduler,
// i.e. the type that selects suitable providers given task information and submits the task.
type Scheduler interface {
	// The Schedule function tries to submit a Task to a suitable Provider's queue and returns the WasmTask struct
	Schedule(ctx context.Context, task *provider.AsyncTask) error
	// Called on task completion to measure overall throughput
	RateTick()
}

// The Dispatcher takes a task queue and a provider selector strategy and then
// decides which task to send to which provider for computation. Additionally,
// limit the number of concurrently scheduling tasks (this does not mean running,
// in-flight tasks but those that are currently "looking for a slot").
func Dispatcher(selector Scheduler, concurrency int) {

	// use ticketing to limit simultaneous schedules
	tickets := make(chan struct{}, concurrency)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	for task := range TaskQueue {
		<-tickets // get a ticket

		// each task is handled in a separate goroutine
		go func(task *provider.AsyncTask) {
			interceptingChannel := make(chan *provider.AsyncTask, 1)
			interceptedChannel := task.Intercept(interceptingChannel)

			retries := 10
			var err error
			errs := make([]error, 0, 10)
			for i := 0; i < retries; i++ {

				// when retrying, we need to reacquire a ticket
				if i > 0 {
					<-tickets
				}

				// schedule the task with a provider and release a ticket
				err = selector.Schedule(task.Context, task)
				tickets <- struct{}{}

				// oops, scheduling error
				if err != nil {
					log.Printf("RETRY: selector.Schedule %s failed (%d)", task.Request.GetInfo().GetId(), i)
					errs = append(errs, err)
					continue // retry
				}

				result := <-interceptingChannel

				// oops, instantiation error or similar
				if result.Error != nil {
					// don't retry, if the context was cancelled
					if errors.Is(result.Error, context.Canceled) {
						break
					}
					log.Printf("RETRY: task %s failed (%d): %v", task.Request.GetInfo().GetId(), i, result.Error)
					err = result.Error
					errs = append(errs, err)
					continue // retry
				}

				// application errors should not be retried, as they are probably client's fault
				break

			}

			// still erroneous after retries, give up
			if err != nil {
				task.Error = errors.Join(errs...)
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
func dynamicSubmit(ctx context.Context, call *provider.AsyncTask, providers []*provider.Provider) error {

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
