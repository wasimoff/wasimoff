package client

import (
	"context"
	"fmt"
	"log"
	"time"
	"wasi.team/broker/provider"
	"wasi.team/broker/scheduler"
	wasimoff "wasi.team/proto/v1"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Continuously schedule 'tsp-rand-10' tasks internally.
func BenchmodeTspFlood(store *provider.ProviderStore, parallel int) {

	if parallel <= 0 {
		return
	}

	// wait for required binary upload
	bin := "tsp.wasm"
	args := []string{"tsp.wasm", "rand", "10"}
	log.Printf("BENCHMODE: please upload %q binary", bin)
	binary := wasimoff.File{Ref: &bin}
	for {
		if store.Storage.Get(bin) != nil {
			// file uploaded
			log.Printf("BENCHMODE: required binary uploaded, let's go ...")
			err := store.Storage.ResolvePbFile(&binary) // ! <-- this one is important
			if err != nil {
				panic(err)
			}
			break
		}
		time.Sleep(time.Second)
	}

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// receive finished tasks to tick the throughput counter and reinsert ticket
	doneChan := make(chan *provider.AsyncTask, parallel)
	go func() {
		for t := range doneChan {
			if t.Error == nil {
				// store.RateTick()
			}
			tickets <- struct{}{}
		}
	}()

	// loop forever with incrementing index
	for i := 0; ; i++ {
		<-tickets
		scheduler.TaskQueue <- provider.NewAsyncTask(
			context.Background(),
			&wasimoff.Task_Wasip1_Request{
				Info: &wasimoff.Task_Metadata{
					Id: proto.String(fmt.Sprintf("benchmode/%d", i)),
				},
				Params: &wasimoff.Task_Wasip1_Params{
					Binary: &binary,
					Args:   args,
				},
			},
			&wasimoff.Task_Wasip1_Response{},
			doneChan,
		)
	}
}

// Continuously schedule 'pyodide' tasks internally.
func BenchmodePyodideTest(parallel int) {

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// receive finished tasks to tick the throughput counter and reinsert ticket
	doneChan := make(chan *provider.AsyncTask, parallel)
	go func() {
		for t := range doneChan {
			if t.Error != nil {
				fmt.Printf("ERR: %s\n", t.Error)
			} else {
				if t.Response.GetError() != "" {
					fmt.Printf("Pytest ERR: %s\n", t.Response.GetError())
				} else {
					r := t.Response.(*wasimoff.Task_Pyodide_Response)
					fmt.Printf("Pytest: %s\n", prototext.Format(r.GetOk()))
				}
			}
			tickets <- struct{}{}
		}
	}()

	// loop forever with incrementing index
	for i := 0; ; i++ {
		<-tickets
		scheduler.TaskQueue <- provider.NewAsyncTask(
			context.Background(),
			&wasimoff.Task_Pyodide_Request{
				Info: &wasimoff.Task_Metadata{
					Id: proto.String(fmt.Sprintf("pytest/%d", i)),
				},
				Params: &wasimoff.Task_Pyodide_Params{
					Run: &wasimoff.Task_Pyodide_Params_Script{
						Script: "import numpy as np; mat = np.random.rand(5,5); print(mat); mat.mean()",
					},
					Packages: []string{"numpy"},
				},
			},
			&wasimoff.Task_Pyodide_Response{},
			doneChan,
		)
	}
}
