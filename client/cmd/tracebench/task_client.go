package main

import (
	"context"
	"strconv"
	"sync"

	"wasi.team/broker/net/transport"
	"wasi.team/client"
	wasimoffv1 "wasi.team/proto/v1"
)

type TraceBenchClient struct {
	client.WasimoffWebsocketClient
	ctx context.Context
}

func Connect(ctx context.Context, broker string) *TraceBenchClient {

	// connect to wasimoff over websocket
	w, err := client.NewWasimoffWebsocketClient(ctx, broker)
	if err != nil {
		panic(err)
	}

	return &TraceBenchClient{
		WasimoffWebsocketClient: *w,
		ctx:                     ctx,
	}

}

type ArgonTasker struct {
	client     *TraceBenchClient
	request    *wasimoffv1.Task_Wasip1_Request
	request_mu sync.Mutex
}

// hash of the currently committed file in repository at wasi-apps/argonload/argonload.wasm
var argonload = "sha256:a77ee84e1e8b0e9734cc4647b8ee0813c55c697c53a38397cc43e308ec871b8f"

func NewArgonTasker(tb *TraceBenchClient) *ArgonTasker {

	req := &wasimoffv1.Task_Wasip1_Request{
		Info: &wasimoffv1.Task_Metadata{},
		Qos:  &wasimoffv1.Task_QoS{},
		Params: &wasimoffv1.Task_Wasip1_Params{
			Binary: &wasimoffv1.File{Ref: &argonload},
			Args:   []string{"argonload", "-i", "10"},
			Envs:   []string{},
		},
	}

	return &ArgonTasker{
		client:     tb,
		request:    req,
		request_mu: sync.Mutex{},
	}

}

func (at *ArgonTasker) Run(calls chan *transport.PendingCall, iterations int) *transport.PendingCall {

	at.request_mu.Lock()
	defer at.request_mu.Unlock()

	// set the iteration count to given parameter
	at.request.Params.Args[2] = strconv.Itoa(iterations)

	// send the request and return async call to unlock mutex quickly again
	response := &wasimoffv1.Task_Wasip1_Response{}
	return at.client.Messenger.SendRequest(at.client.ctx, at.request, response, calls)

}
