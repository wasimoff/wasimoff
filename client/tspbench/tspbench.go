package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"wasi.team/broker/net/transport"
	wasimoff "wasi.team/proto/v1"

	"github.com/paulbellamy/ratecounter"
	"google.golang.org/protobuf/proto"
)

var broker string = "http://localhost:4080"
var tspn int = 10
var parallel int = 32

func main() {

	flag.IntVar(&tspn, "tsp", tspn, "tsp rand N")
	flag.IntVar(&parallel, "p", parallel, "how many parallel tasks to have in-flight")
	flag.StringVar(&broker, "wasimoff", broker, "URL to the Wasimoff Broker")
	flag.Parse()

	// open a websocket to the broker
	messenger, err := transport.DialWasimoff(context.Background(), broker)
	if err != nil {
		log.Fatalf("ERR: %s", err)
	}
	defer messenger.Close(nil)

	// construct task structure once
	task := &wasimoff.Task_Wasip1_Params{
		Binary: &wasimoff.File{Ref: proto.String("tsp.wasm")},
		Args:   []string{"tsp.wasm", "rand", fmt.Sprintf("%d", tspn)},
	}
	request := &wasimoff.Task_Wasip1_Request{Params: task}

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// count completed requests
	counter := atomic.Uint64{}
	ratecounter := ratecounter.NewRateCounter(5 * time.Second)

	for {
		<-tickets
		go func() {

			response := &wasimoff.Task_Wasip1_Response{}
			err := messenger.RequestSync(context.Background(), request, response)
			if err != nil {
				log.Println(err)
				time.Sleep(time.Second)
			} else {
				ratecounter.Incr(1)
				counter.Add(1)
			}
			tickets <- struct{}{}
			fmt.Printf("\rrequests: %12d (%d req/s)", counter.Load(), ratecounter.Rate()/5)

		}()
	}

}
