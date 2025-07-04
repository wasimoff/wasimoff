package provider

import (
	"context"
	"fmt"
	"log"
	"time"

	wasimoff "wasi.team/proto/v1"
)

// AsyncTask is an individual parametrized task from an offloading job that
// can be submitted to a Provider's Submit() channel.
type AsyncTask struct {
	Context  context.Context
	Request  wasimoff.Task_Request  // the overall request with metadata, QoS and task parameters
	Response wasimoff.Task_Response // response containing either an error or specific output

	// track a few things for metrics
	CloudOffloaded bool
	TimeStart      time.Time
	TimeScheduled  time.Time

	Error error           // errors encountered internally during scheduling or RPC
	done  chan *AsyncTask // received itself when complete
}

// NewAsyncTask creates a new call struct for a scheduler
func NewAsyncTask(
	ctx context.Context,
	args wasimoff.Task_Request,
	res wasimoff.Task_Response,
	done chan *AsyncTask,
) *AsyncTask {
	if done == nil {
		done = make(chan *AsyncTask, 1)
	}
	if cap(done) == 0 {
		log.Panic("AsyncTask: done channel is unbuffered")
	}
	if ctx == nil {
		log.Panic("AsyncTask: context is nil")
	}
	return &AsyncTask{
		Context:        ctx,
		Request:        args,
		Response:       res,
		CloudOffloaded: false,
		TimeStart:      time.Now(), // TODO: not quite the actual "start"
		Error:          nil,
		done:           done,
	}
}

// Done signals on the channel that this call is complete
func (t *AsyncTask) Done() *AsyncTask {
	// TODO: re-add a select to never block here?
	t.done <- t
	return t
}

// Check some prerequisites before attempting to schedule a task
func (t *AsyncTask) Check() (err error) {
	// done channel must never be nil
	if t.done == nil {
		panic("AsyncTask.done is nil, nobody is listening for this result")
	}
	// the Request and Result must not be nil
	if t.Request == nil || t.Response == nil {
		return fmt.Errorf("AsyncTask.Request and AsyncTask.Result must not be nil")
	}
	// the context is already cancelled
	if t.Context.Err() != nil {
		return t.Context.Err()
	}
	// ok
	return nil
}

// Intercept replaces the done channel with another channel and returns the previous channel
func (t *AsyncTask) Intercept(interceptingChannel chan *AsyncTask) chan *AsyncTask {
	previous := t.done
	t.done = interceptingChannel
	return previous
}

// DoneCapacity returns the capacity of the done channel
func (t *AsyncTask) DoneCapacity() int {
	return cap(t.done)
}
