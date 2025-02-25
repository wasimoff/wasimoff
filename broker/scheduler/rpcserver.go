package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"wasimoff/broker/provider"
	"wasimoff/broker/storage"
	wasimoff "wasimoff/proto/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

// reuseable task queue for HTTP handler and websocket
var TaskQueue = make(chan *provider.AsyncTask, 2048)

// This ConnectRPC server implements the Wasimoff service from messages.proto, to
// be used with the Connect handler, which automatically creates correct routes and
// gRPC endpoints per request. Since each request is handled in a separate goroutine,
// these functions can be synchronous and blocking but any shared resources must be
// threadsafe, of course.

type ConnectRpcServer struct {
	Store   *provider.ProviderStore
	taskSeq atomic.Uint64
	jobSeq  atomic.Uint64
}

func (s *ConnectRpcServer) Upload(
	ctx context.Context,
	req *connect.Request[wasimoff.Filesystem_Upload_Request],
) (
	*connect.Response[wasimoff.Filesystem_Upload_Response],
	error,
) {
	if req.Msg.Upload == nil {
		return nil, fmt.Errorf("file cannot be nil")
	}
	u := req.Msg.GetUpload()

	// check the content-type of the request: accept zip or wasm
	ft, err := storage.CheckMediaType(u.GetMedia())
	if err != nil {
		return nil, fmt.Errorf("unsupported filetype")
	}

	// can have a friendly lookup-name as query parameter
	name := u.GetRef()

	// insert file in storage
	file, err := s.Store.Storage.Insert(name, ft, u.Blob)
	if err != nil {
		return nil, fmt.Errorf("inserting in storage failed: %w", err)
	}

	// return the content address to client
	return connect.NewResponse(&wasimoff.Filesystem_Upload_Response{
		Ref: proto.String(file.Ref()),
	}), nil
}

func (s *ConnectRpcServer) RunWasip1(
	ctx context.Context,
	req *connect.Request[wasimoff.Task_Wasip1_Request],
) (
	*connect.Response[wasimoff.Task_Wasip1_Response],
	error,
) {
	r := req.Msg

	// resolve any filenames to storage hashes
	if err := s.Store.Storage.ResolveTaskFiles(r); err != nil {
		return nil, err
	}

	// assemble task info for internal dispatcher queue
	r.Info = &wasimoff.Task_Metadata{
		Id:        proto.String(fmt.Sprintf("connect/%d/wasip1", s.taskSeq.Add(1))),
		Requester: proto.String(req.Peer().Addr),
	}

	// dispatch
	response := &wasimoff.Task_Wasip1_Response{}
	done := make(chan *provider.AsyncTask, 1)
	TaskQueue <- provider.NewAsyncTask(ctx, r, response, done)
	call := <-done

	if call.Error != nil {
		return nil, call.Error
	} else {
		return connect.NewResponse(response), nil
	}

}

func (s *ConnectRpcServer) RunWasip1Job(
	ctx context.Context,
	req *connect.Request[wasimoff.Task_Wasip1_JobRequest],
) (
	*connect.Response[wasimoff.Task_Wasip1_JobResponse],
	error,
) {

	job := OffloadingJob{JobSpec: req.Msg}
	// amend the job with information about client
	job.JobID = fmt.Sprintf("%05d", s.jobSeq.Add(1))
	job.ClientAddr = req.Peer().Addr

	// compute all the tasks of a request
	results := dispatchJob(ctx, s.Store, &job, TaskQueue)

	if results.Error != nil {
		return nil, fmt.Errorf(*results.Error)
	} else {
		return connect.NewResponse(results), nil
	}

}

func (s *ConnectRpcServer) RunPyodide(
	ctx context.Context,
	req *connect.Request[wasimoff.Task_Pyodide_Request],
) (
	*connect.Response[wasimoff.Task_Pyodide_Response],
	error,
) {
	r := req.Msg

	// assemble task info for internal dispatcher queue
	r.Info = &wasimoff.Task_Metadata{
		Id:        proto.String(fmt.Sprintf("connect/%d/pyodide", s.taskSeq.Add(1))),
		Requester: proto.String(req.Peer().Addr),
	}

	// dispatch
	response := &wasimoff.Task_Pyodide_Response{}
	done := make(chan *provider.AsyncTask, 1)
	TaskQueue <- provider.NewAsyncTask(ctx, r, response, done)
	call := <-done

	if call.Error != nil {
		return nil, call.Error
	} else {
		return connect.NewResponse(response), nil
	}

}
