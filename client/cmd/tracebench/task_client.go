package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"wasi.team/broker/net/transport"
	"wasi.team/client"
	wasimoffv1 "wasi.team/proto/v1"
)

type TraceBenchTasker interface {
	Run(queue chan *transport.PendingCall, tasklen time.Duration, name string)
}

type ArgonTasker struct {
	ctx      context.Context
	client   *client.WasimoffWebsocketClient
	sequence atomic.Uint64
	wg       *sync.WaitGroup
}

// hash of the currently committed file in repository at wasi-apps/argonload/argonload.wasm
var argonload = "sha256:a77ee84e1e8b0e9734cc4647b8ee0813c55c697c53a38397cc43e308ec871b8f"

func NewArgonTasker(ctx context.Context, wg *sync.WaitGroup, broker string) TraceBenchTasker {

	argon := &ArgonTasker{ctx: ctx, wg: wg}

	// connect to wasimoff over websocket
	if broker != "" {
		w, err := client.NewWasimoffWebsocketClient(ctx, broker)
		if err != nil {
			log.Fatalf("ERR: can't connect to Wasimoff Broker %q: %s", broker, err)
		}
		argon.client = w
	}

	return argon

}

func (at *ArgonTasker) Run(calls chan *transport.PendingCall, seconds time.Duration, name string) {

	request := &wasimoffv1.Task_Wasip1_Request{
		Info: &wasimoffv1.Task_Metadata{
			// set task reference with incrementing counter and name
			Reference: proto.String(fmt.Sprintf("%d/%s", at.sequence.Add(1), name)),
			// start a trace with current start time
			Trace: &wasimoffv1.Task_Trace{
				Created: proto.Int64(time.Now().UnixNano()),
			},
		},
		Qos: &wasimoffv1.Task_QoS{},
		Params: &wasimoffv1.Task_Wasip1_Params{
			// use a fixed sha256 ref for binary
			Binary: &wasimoffv1.File{Ref: &argonload},
			// set the iteration count to given parameter
			Args: []string{"argonload", "-i", strconv.Itoa(durationToIterations(seconds))},
		},
	}

	if at.client != nil {
		// send the request and return async call to unlock mutex quickly again
		response := &wasimoffv1.Task_Wasip1_Response{}
		at.wg.Add(1)
		at.client.Messenger.SendRequest(at.ctx, request, response, calls)

	} else {
		// print the request for dry-runs without an actual client connection
		buf, err := protojson.Marshal(request)
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
