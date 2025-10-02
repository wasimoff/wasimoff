package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"wasi.team/broker/net/transport"
	wasimoffv1 "wasi.team/proto/v1"
)

func main() {

	args := cmdline()

	// read input file
	dataset := ReadDataset(args.Dataset)
	dataset.SelectColumns(args.Columns)
	dataset.ScaleDatasets(args.ScaleRate, args.ScaleTasklen)

	// contexts for app cancellation
	background, shutdown := context.WithCancel(context.Background())
	timeout, cancel := context.WithTimeout(background, 10*time.Second)
	defer shutdown()
	defer cancel()
	go func(ctx context.Context) {
		<-ctx.Done()
		log.Println("timeout reached, exiting ...")
	}(timeout)

	// TODO: use args.DryRun to debug without any Broker connection
	wasimoffClient := Connect(background, args.Broker)

	output := OpenOutputLog(args.Tracefile)
	if output != nil {
		defer output.Close()
	}

	responses := make(chan *transport.PendingCall, 2048)

	threads := sync.WaitGroup{}
	starter := NewStarter[time.Time]()

	for _, col := range args.Columns {
		threads.Add(1)
		starter.Add(1)

		ticker := make(chan Tick, 10)
		argon := NewArgonTasker(wasimoffClient)

		go dataset.InterpolatedRateTicker(timeout, col, starter, ticker)

		go func() { // TODO: make this a func on ArgonTasker
			defer threads.Done()
			for tick := range ticker {
				if diff := time.Since(tick.Scheduled); diff > 10*time.Millisecond {
					fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, tick.Sequence, diff)
				}
				fmt.Printf("[ %3s ] tick %8d / %10s --> %f\n", col, tick.Sequence, tick.Elapsed, tick.TasklenSec)
				argon.Run(responses, tick.TasklenSec*args.ScaleTasklen)
			}
		}()
	}

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
					if err := output.EncodeProto(r.Info.Trace); err != nil {
						log.Fatalf("ERR: failed writing trace log: %s", err)
					}
				}
			}
		}
	}()

	// everyone should be set up, start!
	starter.Wait()
	starter.Broadcast(time.Now())

	threads.Wait()

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
