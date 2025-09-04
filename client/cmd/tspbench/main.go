package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"wasi.team/client"
	wasimoff "wasi.team/proto/v1"

	"github.com/paulbellamy/ratecounter"
	"google.golang.org/protobuf/proto"
)

var (
	broker   string = "http://localhost:4080"
	tspn     int    = 10
	parallel int    = 32
)

func main() {

	flag.IntVar(&tspn, "tsp", tspn, "tsp rand N")
	flag.IntVar(&parallel, "p", parallel, "how many parallel tasks to have in-flight")
	flag.StringVar(&broker, "wasimoff", broker, "URL to the Wasimoff Broker")
	flag.Parse()

	// open a websocket to the broker
	client, err := client.NewWasimoffWebsocketClient(context.Background(), broker)
	if err != nil {
		log.Fatalf("ERR: %s", err)
	}
	defer client.Messenger.Close(nil)

	fmt.Printf("connected to %q\n", broker)
	fmt.Printf("run \"tsp.wasm rand %d\" with %d in parallel\n", tspn, parallel)

	// construct task structure once
	request := &wasimoff.Task_Wasip1_Request{
		Params: &wasimoff.Task_Wasip1_Params{
			Binary: &wasimoff.File{Ref: proto.String("tsp.wasm")},
			Args:   []string{"tsp.wasm", "rand", fmt.Sprintf("%d", tspn)},
		},
	}

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// count completed requests
	counter := atomic.Uint64{}
	ratecounter := ratecounter.NewRateCounter(5 * time.Second)

	// escape sequence to clear line
	clr := "\033[2K\r"

	for {
		<-tickets
		go func() {

			response, err := client.RunWasip1(context.Background(), request)
			if err != nil || response.GetError() != "" {
				// error
				if err != nil {
					fmt.Println()
					log.Fatalf("[%d] ERR: %s", counter.Load(), err)
				} else {
					fmt.Printf(clr+"[%d] FAIL: %s", counter.Load(), response.GetError())
				}
				time.Sleep(time.Second)
			} else {
				// ok
				ratecounter.Incr(1)
				counter.Add(1)
			}
			tickets <- struct{}{}
			fmt.Printf(clr+"requests: %12d (%d req/s)", counter.Load(), ratecounter.Rate()/5)

		}()
	}

}
