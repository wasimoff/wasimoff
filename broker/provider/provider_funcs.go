package provider

import (
	"context"
	"fmt"
	"sync"

	"wasi.team/broker/storage"
	wasimoff "wasi.team/proto/v1"
)

// ----- execute -----

// run is the internal detail, which executes a task on the Provider without semaphore guards
func (p *Provider) run(ctx context.Context, args wasimoff.Task_Request, result wasimoff.Task_Response) (err error) {
	// addr := p.Get(Address)
	// task := args.GetInfo().GetId()
	// log.Printf(">>> schedule %s >> %s", task, addr)
	if err := p.messenger.RequestSync(ctx, args, result); err != nil {
		// log.Printf("ERROR! <<<<< %s << %s", task, addr)
		return fmt.Errorf("provider.run failed: %w", err)
	}
	// log.Printf("<<< finished %s << %s", task, addr)
	return
}

// Run will run a task on a Provider synchronously, respecting limiter.
func (p *Provider) Run(ctx context.Context, args wasimoff.Task_Request, result wasimoff.Task_Response) error {
	p.limiter.Acquire(ctx, 1)
	defer p.limiter.Release(1)
	return p.run(ctx, args, result)
}

// TryRun will attempt to run a task on the Provider but fails when there is no capacity.
func (p *Provider) TryRun(ctx context.Context, args wasimoff.Task_Request, result wasimoff.Task_Response) error {
	if ok := p.limiter.TryAcquire(1); !ok {
		return fmt.Errorf("no free capacity")
	}
	defer p.limiter.Release(1)
	return p.run(ctx, args, result)
}

// ----- filesystem -----

// ListFiles asks the Provider to list their files in storage
func (p *Provider) ListFiles() error {

	// receive listing into a new struct
	args := wasimoff.Filesystem_Listing_Request{}
	response := wasimoff.Filesystem_Listing_Response{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return fmt.Errorf("provider.ListFiles failed: %w", err)
	}

	// (re)set known files from received list
	p.files = sync.Map{}
	for _, filename := range response.Files {
		p.files.Store(filename, nil)
	}
	return nil
}

// ProbeFile sends a content-address name to check if the Provider *has* a file
func (p *Provider) ProbeFile(addr string) (has bool, err error) {

	// receive response bool into a new struct
	args := wasimoff.Filesystem_Probe_Request{File: &addr}
	response := wasimoff.Filesystem_Probe_Response{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return false, fmt.Errorf("provider.ProbeFile failed: %w", err)
	}

	return response.GetOk(), nil
}

// Upload a file from Storage to this Provider
func (p *Provider) Upload(file *storage.File) (err error) {
	ref := file.Ref()

	// when returning without error, add the file to provider's list
	// (either probe was ok or upload successful)
	defer func() {
		if err == nil {
			p.files.Store(ref, nil)
		}
	}()

	// always probe for file first
	if has, err := p.ProbeFile(ref); err != nil {
		return fmt.Errorf("provider.Upload failed probe before upload: %w", err)
	} else if has {
		return nil // ok, provider has this file already
	}

	// otherwise upload it
	args := wasimoff.Filesystem_Upload_Request{Upload: &wasimoff.File{
		Ref:   &ref,
		Media: &file.Media,
		Blob:  file.Bytes,
	}}
	response := wasimoff.Filesystem_Upload_Response{}
	if err := p.messenger.RequestSync(context.TODO(), &args, &response); err != nil {
		return fmt.Errorf("provider.Upload %q failed: %w", ref, err)
	}
	if response.GetRef() != ref {
		return fmt.Errorf("provider.Upload %q failed: Provider computed a different ref: %s", ref, response.GetRef())
	}
	return
}

// Has returns if this Provider *is known* to have a certain file, without re-probing
func (p *Provider) Has(file string) bool {
	_, ok := p.files.Load(file)
	return ok
}
