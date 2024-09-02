package provider

import (
	"context"
	"fmt"
	"sync"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"

	"github.com/marusama/semaphore/v2"
)

// Provider is a single connection initiated by a computing provider
type Provider struct {
	messenger *transport.Messenger // messenger connection to provider

	// cancellable lifetime context to signal closure upwards
	lifetime  transport.Lifetime
	closeOnce sync.Once

	// unbuffered channel to submit tasks; can be `nil` if nobody's listening
	Submit chan *PendingWasiCall

	// resizeable semaphore to limit number of concurrent tasks
	limiter semaphore.Semaphore

	// information about the provider, to be accessed with Get()
	info map[ProviderInfoKey]string

	// list of files known on this provider
	files []string
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
		files:     make([]string, 10),
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

// PendingWasiCall represents an asynchronous WebAssembly exec call
type PendingWasiCall struct {
	Request *pb.ExecuteWasiArgs     // arguments to the call
	Result  *pb.ExecuteWasiResponse // response from the Provider
	Error   error                   // error encountered during the call
	Done    chan *PendingWasiCall   // receives itself when request completes
}

// NewPendingWasiCall creates a new call struct for the Submit chan
func NewPendingWasiCall(args *pb.ExecuteWasiArgs, res *pb.ExecuteWasiResponse) *PendingWasiCall {
	// TODO: very duplicate with a *Task and at the same time an RPC *Call ...
	return &PendingWasiCall{
		Request: args,
		Result:  res,
		Done:    make(chan *PendingWasiCall, 1),
	}
}

// done signals on the channel that this call is complete
func (call *PendingWasiCall) done() *PendingWasiCall {
	select {
	case call.Done <- call: // ok
	default: // never block here
	}
	return call
}

// Accept tasks on an unbuffered channel to submit to the Provider. Channels can
// be used in a DynamicSubmit, so calls from many different sources can be efficiently
// distributed to multiple Providers.
func (p *Provider) acceptTasks() (err error) {

	// initialize the channel
	if p.Submit == nil {
		p.Submit = make(chan *PendingWasiCall) // unbuffered by design
	}

	// close Provider if the loop exits
	defer p.Close(err)

	for {

		// acquire a semaphore before accepting a task
		//? off-by-one because we acquire and hold a semaphore before we even get a task
		if err = p.limiter.Acquire(p.lifetime.Context, 1); err != nil {
			// nobody to notify and nothing to free, just quit
			return err
		}

		select {

		// Provider is closing, quit the loop
		case <-p.lifetime.Closing():
			return p.Err()

		// receive call details from channel
		case call := <-p.Submit:
			// the Done channel MUST NEVER be nil
			if call.Done == nil {
				panic("call.Done is nil, nobody is listening for this result")
			}
			// the Request and Result most not be nil
			if call.Request == nil || call.Result == nil {
				call.Error = fmt.Errorf("call.Request and call.Result must not be nil")
				call.done()
				p.limiter.Release(1)
				continue
			}

			// run the Request in a goroutine asynchronously
			// TODO: avoid gofunc by using a second listener on a `chan *PendingCall`
			go func() {
				call.Error = p.run(call.Request, call.Result)
				call.done()
				p.limiter.Release(1)
			}()

		}

	}
}
