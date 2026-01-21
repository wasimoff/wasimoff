package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync/atomic"

	"wasi.team/broker/provider"
	"wasi.team/broker/scheduler"
	"wasi.team/broker/storage"
	wasimoff "wasi.team/proto/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

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
	// assemble task info for internal dispatcher queue
	r := req.Msg
	r.Info = s.prepareTaskInfo(r.GetInfo(), req.Peer())

	// resolve any filenames to storage hashes
	if err := s.Store.Storage.ResolveTaskFiles(r); err != nil {
		return nil, err
	}

	// dispatch
	response := &wasimoff.Task_Wasip1_Response{}
	done := make(chan *provider.AsyncTask, 1)
	r.Info.TraceEvent(wasimoff.Task_TraceEvent_BrokerQueueTask)
	SubmitToQueue(scheduler.TaskQueue, provider.NewAsyncTask(ctx, r, response, done))
	call := <-done
	s.copyTaskInfo(r.Info, &response.Info)

	if call.Error != nil {
		return nil, call.Error
	} else {
		response.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_BrokerTransmitClientResponse)
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
	// TODO: request metadata with optional trace for individual tasks

	// compute all the tasks of a request
	results := dispatchJob(ctx, s.Store, &job, scheduler.TaskQueue)

	if results.Error != nil {
		return nil, errors.New(*results.Error)
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
	// assemble task info for internal dispatcher queue
	r := req.Msg
	r.Info = s.prepareTaskInfo(r.GetInfo(), req.Peer())

	// dispatch
	response := &wasimoff.Task_Pyodide_Response{}
	done := make(chan *provider.AsyncTask, 1)
	r.Info.TraceEvent(wasimoff.Task_TraceEvent_BrokerQueueTask)
	SubmitToQueue(scheduler.TaskQueue, provider.NewAsyncTask(ctx, r, response, done))
	call := <-done
	s.copyTaskInfo(r.Info, &response.Info)

	if call.Error != nil {
		return nil, call.Error
	} else {
		response.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_BrokerTransmitClientResponse)
		return connect.NewResponse(response), nil
	}

}

// try to submit a task to the queue or return an error immediately
func SubmitToQueue(queue chan *provider.AsyncTask, task *provider.AsyncTask) {
	select {
	case queue <- task:
		return // ok
	default:
		task.Error = fmt.Errorf("429: Queue Full")
		task.Done()
		return
	}
}

// -------------------- handlers for task metadata --------------------

func (s *ConnectRpcServer) prepareTaskInfo(info *wasimoff.Task_Metadata, peer connect.Peer) *wasimoff.Task_Metadata {
	if info != nil {
		info.TraceEvent(wasimoff.Task_TraceEvent_BrokerReceivedClientRequest)
	}
	return &wasimoff.Task_Metadata{
		Id:        proto.String(strconv.FormatUint(s.taskSeq.Add(1), 10)),
		Requester: proto.String(peer.Addr),
		Reference: proto.String(info.GetReference()),
		Trace:     info.GetTrace(),
		Provider:  nil,
	}
}

func (s *ConnectRpcServer) copyTaskInfo(info *wasimoff.Task_Metadata, res **wasimoff.Task_Metadata) {
	if *res == nil {
		log.Printf("WARN: Metadata missing on response for Task %q", *info.Id)
		*res = info
	} else {
		r := *res
		r.Id = info.Id
		r.Reference = info.Reference
		r.Requester = info.Requester
	}
	(*res).TraceEvent(wasimoff.Task_TraceEvent_BrokerReceivedProviderResult)
}
