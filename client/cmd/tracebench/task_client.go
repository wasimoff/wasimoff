package main

import (
	"context"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"wasi.team/broker/net/transport"
	"wasi.team/client"
	wasimoffv1 "wasi.team/proto/v1"
)

type ArgonTasker struct {
	ctx        context.Context
	client     *client.WasimoffWebsocketClient
	request    *wasimoffv1.Task_Wasip1_Request
	request_mu sync.Mutex
}

// hash of the currently committed file in repository at wasi-apps/argonload/argonload.wasm
var argonload = "sha256:a77ee84e1e8b0e9734cc4647b8ee0813c55c697c53a38397cc43e308ec871b8f"

func NewArgonTasker(ctx context.Context, broker string) *ArgonTasker {

	argon := &ArgonTasker{
		ctx:        ctx,
		request_mu: sync.Mutex{},
	}

	// connect to wasimoff over websocket
	if broker != "" {
		w, err := client.NewWasimoffWebsocketClient(ctx, broker)
		if err != nil {
			log.Fatalf("ERR: can't connect to Wasimoff Broker %q: %s", broker, err)
		}
		argon.client = w
	}

	// TODO: check if mutex + message reuse is the best idea here
	argon.request = &wasimoffv1.Task_Wasip1_Request{
		Info: &wasimoffv1.Task_Metadata{
			Reference: proto.String("argontasker"),
			Trace:     &wasimoffv1.Task_Trace{},
		},
		Qos: &wasimoffv1.Task_QoS{},
		Params: &wasimoffv1.Task_Wasip1_Params{
			Binary: &wasimoffv1.File{Ref: &argonload},
			Args:   []string{"argonload", "-i", "10"},
			Envs:   []string{},
		},
	}

	return argon

}

func (at *ArgonTasker) Run(calls chan *transport.PendingCall, seconds time.Duration) {

	at.request_mu.Lock()
	defer at.request_mu.Unlock()

	// set the iteration count to given parameter
	iter := durationToIterations(seconds)
	at.request.Params.Args[2] = strconv.Itoa(iter)

	// set current start time in trace
	at.request.Info.Trace.Created = proto.Int64(time.Now().UnixNano())

	if at.client != nil {
		// send the request and return async call to unlock mutex quickly again
		response := &wasimoffv1.Task_Wasip1_Response{}
		at.client.Messenger.SendRequest(at.ctx, at.request, response, calls)

	} else {
		// print the request for dry-runs without an actual client connection
		buf, err := protojson.Marshal(at.request)
		if err != nil {
			panic(err)
		}
		log.Printf("DRYRUN: %s", buf)
	}

}

// For argonload/wasm running on an Intel i5-1345U, we get around iter=35 for 1s runtime.
func durationToIterations(d time.Duration) int {
	itertations := 35 * d.Seconds()
	return int(math.Ceil(itertations))
}
