package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"wasi.team/broker/net/transport"
	"wasi.team/client/tracebench"
	"wasi.team/client/tracebench/csvtrace"
	wasimoffv1 "wasi.team/proto/v1"
)

func main() {

	// parse commandline arguments
	args := cmdline()

	// read input file and apply modifiers
	dataset := csvtrace.ReadDataset(args.Dataset)
	dataset.SelectColumns(args.Columns)
	dataset.ScaleDatasets(args.ScaleRate, args.ScaleTasklen)

	// contexts for app cancellation
	background, shutdown := context.WithCancel(context.Background())
	timeout, cancel := context.WithTimeout(background, time.Duration(args.Timeout)*time.Second)
	defer shutdown()
	defer cancel()
	go func(ctx context.Context) {
		<-ctx.Done()
		log.Println("timeout reached, exiting ...")
	}(timeout)

	// create the request generator
	argon := NewArgonTasker(background, args.Broker)

	output := NewProtoDelimEncoder(args.Tracefile)
	defer output.Close()

	responses := make(chan *transport.PendingCall, 2048)

	threads := sync.WaitGroup{}
	starter := tracebench.NewStarter[time.Time]()

	// spawn ticker threads
	for _, col := range args.Columns {
		threads.Add(1)
		starter.Add(1)

		// create a ticker channel for this dataset column
		ticker := make(chan tracebench.TaskTick, 10)
		go dataset.InterpolatedRateTicker(timeout, col, starter, ticker)

		// another thread to send requests on ticks
		go func(ar *ArgonTasker) {
			defer threads.Done()
			for tick := range ticker {
				if diff := time.Since(tick.Scheduled); diff > 10*time.Millisecond {
					fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, tick.Sequence, diff)
				}
				fmt.Printf("[ %3s ] tick %8d / %10s --> %v\n", col, tick.Sequence, tick.Elapsed, tick.Tasklen)
				ar.Run(responses, float64(tick.Tasklen))
			}
		}(argon)
	}

	// a single function to handle responses
	go func() {
		for c := range responses {
			if c.Error != nil {
				fmt.Fprintf(os.Stderr, "ERR: %s\n", c.Error)
			} else {
				r, ok := c.Response.(*wasimoffv1.Task_Wasip1_Response)
				if !ok {
					panic("can't cast the response to *wasimoffv1.Task_Wasip1_Response")
				}
				fmt.Printf("Task OK: %10s on %s\n", *r.Info.Id, *r.Info.Provider)
				if output != nil {
					if err := output.Write(r.Info); err != nil {
						log.Fatalf("ERR: failed writing trace log: %s", err)
					}
				} else {
					buf, err := protojson.Marshal(r.Info)
					if err != nil {
						panic(err)
					}
					log.Printf("%s", buf)
				}
			}
		}
	}()

	// everyone should be set up, start!
	starter.Wait()
	starter.Broadcast(time.Now())

	// wait for tickers to finish for clean exit
	// TODO: this does NOT wait for all responses to arrive
	threads.Wait()

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
