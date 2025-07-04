package scheduler

import (
	"context"
	"errors"
	"log"
	"math"
	"reflect"
	"time"

	"wasi.team/broker/provider"
)

// reuseable task queue for HTTP handler and websocket
var TaskQueue = make(chan *provider.AsyncTask, 2048)

// Scheduler is a generic interface which must be fulfilled by a concrete scheduler,
// i.e. the type that selects suitable providers given task information and submits the task.
type Scheduler interface {
	// The Schedule function tries to submit a Task to a suitable Provider's queue and returns the WasmTask struct
	Schedule(ctx context.Context, task *provider.AsyncTask) error
}

// The Dispatcher takes a task queue and a provider selector strategy and then
// decides which task to send to which provider for computation. Additionally,
// limit the number of concurrently scheduling tasks (this does not mean running,
// in-flight tasks but those that are currently "looking for a slot").
func Dispatcher(store *provider.ProviderStore, selector Scheduler, concurrency int) {

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
			for i := 1; i <= retries; i++ {

				// when retrying, we sleep and need to reacquire a ticket
				// also increment the retry counter in metrics
				if i > 1 {
					store.ObserveRetry(i)
					time.Sleep(exponentialDelay(i))
					<-tickets
				}

				// schedule the task with a provider and release a ticket
				err = selector.Schedule(task.Context, task)
				tickets <- struct{}{}

				// oops, scheduling error
				if err != nil {
					// don't retry, if the context was cancelled
					if errors.Is(err, context.Canceled) {
						break
					}
					log.Printf("RETRY: scheduling %s failed (%d/%d): %s", task.Request.GetInfo().GetId(), i, retries, err)
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
					log.Printf("RETRY: task %s failed (%d/%d): %v", task.Request.GetInfo().GetId(), i, retries, result.Error)
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
			}
			store.ObserveCompleted(task)
			interceptedChannel <- task

		}(task)
	}
}

// exponentialDelay gives a duration between 10ms and 1s for i=1..9
func exponentialDelay(i int) time.Duration {
	// fn(i) = a*e^(i/b) with a,b such that fn(2..10) = 10..1000
	ms := 3.16228 * math.Exp(float64(i)/1.73718)
	return time.Duration(ms * float64(time.Millisecond))

}

// dynamicSubmit uses `reflect.Select` to dynamically select a Provider to submit a task to.
// This uses the Providers' unbuffered Queue, so that a task can only be submitted to a Provider
// when it currently has free capacity, without needing to busy-loop and recheck capacity yourself.
// Attempt to use regular Providers and fall back to add cloud queue if none are free immediately.
// Based on StackOverflow answer by Dave C. on https://stackoverflow.com/a/32381409.
func dynamicSubmit(
	timeout context.Context,
	task *provider.AsyncTask,
	providers []*provider.Provider,
	cloud chan *provider.AsyncTask,
) error {

	// setup select cases
	cases := make([]reflect.SelectCase, len(providers), len(providers)+2)
	for i, p := range providers {
		if p.Submit == nil {
			panic("provider does not have a queue")
		}
		cases[i].Dir = reflect.SelectSend
		cases[i].Chan = reflect.ValueOf(p.Submit)
		cases[i].Send = reflect.ValueOf(task)
	}

	// set scheduling time on return
	defer func() {
		task.TimeScheduled = time.Now()
	}()

	// if there is a cloud offloading capability, attempt queueing only on providers first
	if cloud != nil {
		// first attempt with default case
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectDefault})
		i, _, _ := reflect.Select(cases)
		if i == len(cases)-1 { // last item, i.e. default case
			// no providers immediately free, replace default case with cloud queue
			cases[len(providers)] = reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: reflect.ValueOf(cloud),
				Send: reflect.ValueOf(task),
			}
			// log.Printf("task %s: added cloud offloading queue", *task.Request.GetInfo().Id)
		} else {
			// successfully queued on some provider
			return nil
		}
	}

	// add context.Done as select case for timeout or cancellation
	if timeout != nil {
		cases = append(cases, reflect.SelectCase{
			Chan: reflect.ValueOf(timeout.Done()),
			Dir:  reflect.SelectRecv,
		})
	}

	// select one of the queues
	i, _, _ := reflect.Select(cases)
	if i == len(cases)-1 { // last item, i.e. timeout / ctx.Done
		return timeout.Err()
	}

	if cloud != nil && i == len(cases)-2 {
		log.Printf("task %s: scheduled on cloud offloading", *task.Request.GetInfo().Id)
		task.CloudOffloaded = true
	}
	if i < len(providers) {
		log.Printf("task %s: scheduled on provider %s", *task.Request.GetInfo().Id, providers[i].Get(provider.Address))
	}

	return nil

}
