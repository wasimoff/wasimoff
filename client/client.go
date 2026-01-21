package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"wasi.team/broker/net/transport"
	wasimoff "wasi.team/proto/v1"
	"wasi.team/proto/v1/wasimoffv1connect"

	"github.com/gabriel-vasile/mimetype"
)

// abstract wasimoff client
type WasimoffClient interface {
	Upload(buf []byte, name string) (ref string, err error)
	RunWasip1(ctx context.Context, request *wasimoff.Task_Wasip1_Request) (*wasimoff.Task_Wasip1_Response, error)
	RunPyodide(ctx context.Context, request *wasimoff.Task_Pyodide_Request) (*wasimoff.Task_Pyodide_Response, error)
}

//  ConnectRPC
// ------------------------------------------------------------------------------------

// wasimoff client over http using connectrpc
type WasimoffConnectRpcClient struct {
	http       *http.Client
	origin     *url.URL
	ConnectRPC wasimoffv1connect.TasksClient
}

func NewWasimoffConnectRpcClient(httpClient *http.Client, broker string) *WasimoffConnectRpcClient {

	// hope your url is well formed :)
	origin, err := url.Parse(broker)
	if err != nil {
		panic(fmt.Errorf("failed to parse broker URL: %w", err))
	}

	// create the connectrpc client
	client := wasimoffv1connect.NewTasksClient(httpClient, origin.JoinPath("/api/client").String())

	return &WasimoffConnectRpcClient{
		http:       httpClient,
		origin:     origin,
		ConnectRPC: client,
	}
}

func (c *WasimoffConnectRpcClient) Upload(buf []byte, name string) (ref string, err error) {

	// detect mediatype
	mt := mimetype.Detect(buf)

	// format URL and append name if given
	upload := c.origin.JoinPath("/api/storage/upload")
	if name != "" {
		q := upload.Query()
		q.Del("name")
		q.Add("name", name)
		upload.RawQuery = q.Encode()
	}

	// upload the buffer
	resp, err := c.http.Post(upload.String(), mt.String(), bytes.NewBuffer(buf))
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	// check if status is ok
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed: %s", resp.Status)
	}

	// read returned ref from response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed reading response: %w", err)
	}

	// body usually has trailing newline
	return strings.TrimSpace(string(body)), nil

}

func (c *WasimoffConnectRpcClient) RunWasip1(ctx context.Context, request *wasimoff.Task_Wasip1_Request) (*wasimoff.Task_Wasip1_Response, error) {
	request.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientTransmitRequest)
	resp, err := c.ConnectRPC.RunWasip1(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	resp.Msg.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientReceivedResponse)
	return resp.Msg, nil
}

func (c *WasimoffConnectRpcClient) RunPyodide(ctx context.Context, request *wasimoff.Task_Pyodide_Request) (*wasimoff.Task_Pyodide_Response, error) {
	request.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientTransmitRequest)
	resp, err := c.ConnectRPC.RunPyodide(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	resp.Msg.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientReceivedResponse)
	return resp.Msg, nil
}

//  WebSocket
// ------------------------------------------------------------------------------------

// wasimoff client over websocket
type WasimoffWebsocketClient struct {
	ctx       context.Context
	origin    *url.URL
	socket    *transport.WebSocketTransport
	Messenger *transport.Messenger
}

func NewWasimoffWebsocketClient(ctx context.Context, broker string) (*WasimoffWebsocketClient, error) {

	// construct path to client endpoint
	origin, err := url.Parse(broker)
	if err != nil {
		return nil, fmt.Errorf("parsing broker url: %w", err)
	}
	path := origin.JoinPath("/api/client/ws").String()

	// open a websocket to the broker
	socket, err := transport.DialWebSocketTransport(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("opening websocket: %w", err)
	}

	// wrap it in a messenger for RPC
	msg := transport.NewMessengerInterface(socket)

	return &WasimoffWebsocketClient{ctx, origin, socket, msg}, nil

}

func (c *WasimoffWebsocketClient) Upload(buf []byte, name string) (ref string, err error) {

	// detect mediatype
	mt := mimetype.Detect(buf)

	resp := &wasimoff.Filesystem_Upload_Response{}
	err = c.Messenger.RequestSync(
		c.ctx,
		&wasimoff.Filesystem_Upload_Request{Upload: &wasimoff.File{
			Media: proto.String(mt.String()),
			Ref:   proto.String(name),
			Blob:  buf,
		}},
		resp,
	)
	if err != nil {
		return "", err
	}

	// body usually has trailing newline
	return strings.TrimSpace(resp.GetRef()), nil

}

func (c *WasimoffWebsocketClient) RunWasip1(ctx context.Context, request *wasimoff.Task_Wasip1_Request) (response *wasimoff.Task_Wasip1_Response, err error) {
	request.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientTransmitRequest)
	response = &wasimoff.Task_Wasip1_Response{}
	err = c.Messenger.RequestSync(ctx, request, response)
	response.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientReceivedResponse)
	return
}

func (c *WasimoffWebsocketClient) RunPyodide(ctx context.Context, request *wasimoff.Task_Pyodide_Request) (response *wasimoff.Task_Pyodide_Response, err error) {
	request.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientTransmitRequest)
	response = &wasimoff.Task_Pyodide_Response{}
	err = c.Messenger.RequestSync(ctx, request, response)
	response.GetInfo().TraceEvent(wasimoff.Task_TraceEvent_ClientReceivedResponse)
	return
}

//  example Helpers on interface
// ------------------------------------------------------------------------------------

func Exec(c WasimoffClient, args, envs []string, stdin []byte, rootfsRef string, artifacts []string) (*wasimoff.Task_Wasip1_Output, error) {

	// construct the request from arguments
	request := &wasimoff.Task_Wasip1_Request{
		Params: &wasimoff.Task_Wasip1_Params{
			// wasm binary is first argument
			Binary: &wasimoff.File{Ref: &args[0]},
			Args:   args,
			Envs:   envs,
			Stdin:  stdin,
		},
	}
	if rootfsRef != "" {
		request.Params.Rootfs = &wasimoff.File{
			Ref: &rootfsRef,
		}
	}
	if artifacts != nil {
		request.Params.Artifacts = artifacts
	}

	// send off the request
	response, err := c.RunWasip1(context.TODO(), request)
	if err != nil {
		// general RPC error
		return nil, fmt.Errorf("RunWasip1 error: %w", err)
	}
	if response.GetError() != "" {
		// request is technically fine but failed
		return nil, fmt.Errorf("RunWasip1 request failure: %s", response.GetError())
	}

	// return "OK" response
	return response.GetOk(), nil

}
