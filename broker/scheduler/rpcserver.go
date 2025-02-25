package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"wasimoff/broker/provider"
	"wasimoff/broker/storage"
	wasimoff "wasimoff/proto/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// This ConnectRPC server implements the Wasimoff service from messages.proto, to
// be used with the Connect handler, which automatically creates correct routes and
// gRPC endpoints per request. Since each request is handled in a separate goroutine,
// these functions can be synchronous and blocking but any shared resources must be
// threadsafe, of course.

type ConnectRpcServer struct {
	Store *provider.ProviderStore
	ctr   atomic.Uint64
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
		Id:        proto.String(fmt.Sprintf("connect/%d/wasip1", s.ctr.Add(1))),
		Requester: proto.String(req.Peer().Addr),
	}
	log.Println("Run Wasip1", prototext.Format(r.Info))

	// dispatch
	response := &wasimoff.Task_Wasip1_Response{}
	done := make(chan *provider.AsyncTask, 1)
	task := provider.NewAsyncTask(ctx, r, response, done)
	taskQueue <- task
	call := <-done

	if call.Error != nil {
		return nil, call.Error
	} else {
		return connect.NewResponse(response), nil
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
		Id:        proto.String(fmt.Sprintf("connect/%d/pyodide", s.ctr.Add(1))),
		Requester: proto.String(req.Peer().Addr),
	}

	// dispatch
	response := &wasimoff.Task_Pyodide_Response{}
	done := make(chan *provider.AsyncTask, 1)
	taskQueue <- provider.NewAsyncTask(ctx, r, response, done)
	call := <-done

	if call.Error != nil {
		return nil, call.Error
	} else {
		return connect.NewResponse(response), nil
	}

}
