// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/v1/messages.proto

package wasimoffv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	http "net/http"
	strings "strings"
	v1 "wasimoff/proto/v1"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// WasimoffName is the fully-qualified name of the Wasimoff service.
	WasimoffName = "wasimoff.v1.Wasimoff"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// WasimoffRunWasip1Procedure is the fully-qualified name of the Wasimoff's RunWasip1 RPC.
	WasimoffRunWasip1Procedure = "/wasimoff.v1.Wasimoff/RunWasip1"
	// WasimoffRunPyodideProcedure is the fully-qualified name of the Wasimoff's RunPyodide RPC.
	WasimoffRunPyodideProcedure = "/wasimoff.v1.Wasimoff/RunPyodide"
	// WasimoffUploadProcedure is the fully-qualified name of the Wasimoff's Upload RPC.
	WasimoffUploadProcedure = "/wasimoff.v1.Wasimoff/Upload"
)

// WasimoffClient is a client for the wasimoff.v1.Wasimoff service.
type WasimoffClient interface {
	// offload a WebAssembly WASI preview 1 task
	RunWasip1(context.Context, *connect.Request[v1.Task_Wasip1_Request]) (*connect.Response[v1.Task_Wasip1_Response], error)
	// offload a Python task in Pyodide
	RunPyodide(context.Context, *connect.Request[v1.Task_Pyodide_Request]) (*connect.Response[v1.Task_Pyodide_Response], error)
	// upload a file to the broker
	Upload(context.Context, *connect.Request[v1.Filesystem_Upload_Request]) (*connect.Response[v1.Filesystem_Upload_Response], error)
}

// NewWasimoffClient constructs a client for the wasimoff.v1.Wasimoff service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewWasimoffClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) WasimoffClient {
	baseURL = strings.TrimRight(baseURL, "/")
	wasimoffMethods := v1.File_proto_v1_messages_proto.Services().ByName("Wasimoff").Methods()
	return &wasimoffClient{
		runWasip1: connect.NewClient[v1.Task_Wasip1_Request, v1.Task_Wasip1_Response](
			httpClient,
			baseURL+WasimoffRunWasip1Procedure,
			connect.WithSchema(wasimoffMethods.ByName("RunWasip1")),
			connect.WithClientOptions(opts...),
		),
		runPyodide: connect.NewClient[v1.Task_Pyodide_Request, v1.Task_Pyodide_Response](
			httpClient,
			baseURL+WasimoffRunPyodideProcedure,
			connect.WithSchema(wasimoffMethods.ByName("RunPyodide")),
			connect.WithClientOptions(opts...),
		),
		upload: connect.NewClient[v1.Filesystem_Upload_Request, v1.Filesystem_Upload_Response](
			httpClient,
			baseURL+WasimoffUploadProcedure,
			connect.WithSchema(wasimoffMethods.ByName("Upload")),
			connect.WithClientOptions(opts...),
		),
	}
}

// wasimoffClient implements WasimoffClient.
type wasimoffClient struct {
	runWasip1  *connect.Client[v1.Task_Wasip1_Request, v1.Task_Wasip1_Response]
	runPyodide *connect.Client[v1.Task_Pyodide_Request, v1.Task_Pyodide_Response]
	upload     *connect.Client[v1.Filesystem_Upload_Request, v1.Filesystem_Upload_Response]
}

// RunWasip1 calls wasimoff.v1.Wasimoff.RunWasip1.
func (c *wasimoffClient) RunWasip1(ctx context.Context, req *connect.Request[v1.Task_Wasip1_Request]) (*connect.Response[v1.Task_Wasip1_Response], error) {
	return c.runWasip1.CallUnary(ctx, req)
}

// RunPyodide calls wasimoff.v1.Wasimoff.RunPyodide.
func (c *wasimoffClient) RunPyodide(ctx context.Context, req *connect.Request[v1.Task_Pyodide_Request]) (*connect.Response[v1.Task_Pyodide_Response], error) {
	return c.runPyodide.CallUnary(ctx, req)
}

// Upload calls wasimoff.v1.Wasimoff.Upload.
func (c *wasimoffClient) Upload(ctx context.Context, req *connect.Request[v1.Filesystem_Upload_Request]) (*connect.Response[v1.Filesystem_Upload_Response], error) {
	return c.upload.CallUnary(ctx, req)
}

// WasimoffHandler is an implementation of the wasimoff.v1.Wasimoff service.
type WasimoffHandler interface {
	// offload a WebAssembly WASI preview 1 task
	RunWasip1(context.Context, *connect.Request[v1.Task_Wasip1_Request]) (*connect.Response[v1.Task_Wasip1_Response], error)
	// offload a Python task in Pyodide
	RunPyodide(context.Context, *connect.Request[v1.Task_Pyodide_Request]) (*connect.Response[v1.Task_Pyodide_Response], error)
	// upload a file to the broker
	Upload(context.Context, *connect.Request[v1.Filesystem_Upload_Request]) (*connect.Response[v1.Filesystem_Upload_Response], error)
}

// NewWasimoffHandler builds an HTTP handler from the service implementation. It returns the path on
// which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewWasimoffHandler(svc WasimoffHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	wasimoffMethods := v1.File_proto_v1_messages_proto.Services().ByName("Wasimoff").Methods()
	wasimoffRunWasip1Handler := connect.NewUnaryHandler(
		WasimoffRunWasip1Procedure,
		svc.RunWasip1,
		connect.WithSchema(wasimoffMethods.ByName("RunWasip1")),
		connect.WithHandlerOptions(opts...),
	)
	wasimoffRunPyodideHandler := connect.NewUnaryHandler(
		WasimoffRunPyodideProcedure,
		svc.RunPyodide,
		connect.WithSchema(wasimoffMethods.ByName("RunPyodide")),
		connect.WithHandlerOptions(opts...),
	)
	wasimoffUploadHandler := connect.NewUnaryHandler(
		WasimoffUploadProcedure,
		svc.Upload,
		connect.WithSchema(wasimoffMethods.ByName("Upload")),
		connect.WithHandlerOptions(opts...),
	)
	return "/wasimoff.v1.Wasimoff/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case WasimoffRunWasip1Procedure:
			wasimoffRunWasip1Handler.ServeHTTP(w, r)
		case WasimoffRunPyodideProcedure:
			wasimoffRunPyodideHandler.ServeHTTP(w, r)
		case WasimoffUploadProcedure:
			wasimoffUploadHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedWasimoffHandler returns CodeUnimplemented from all methods.
type UnimplementedWasimoffHandler struct{}

func (UnimplementedWasimoffHandler) RunWasip1(context.Context, *connect.Request[v1.Task_Wasip1_Request]) (*connect.Response[v1.Task_Wasip1_Response], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("wasimoff.v1.Wasimoff.RunWasip1 is not implemented"))
}

func (UnimplementedWasimoffHandler) RunPyodide(context.Context, *connect.Request[v1.Task_Pyodide_Request]) (*connect.Response[v1.Task_Pyodide_Response], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("wasimoff.v1.Wasimoff.RunPyodide is not implemented"))
}

func (UnimplementedWasimoffHandler) Upload(context.Context, *connect.Request[v1.Filesystem_Upload_Request]) (*connect.Response[v1.Filesystem_Upload_Response], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("wasimoff.v1.Wasimoff.Upload is not implemented"))
}
