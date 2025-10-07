package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"wasi.team/broker/net/transport"
	"wasi.team/client/tracebench"
	"wasi.team/client/tracebench/funcgen"
	wasimoffv1 "wasi.team/proto/v1"
)

func main() {

	// parse commandline arguments
	args, err := cmdline()
	if err != nil {
		log.Fatalf("cmdline: %v", err)
	}

	// cancellable background context for app termination
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create the request generator
	// allows for different task generators, could be string-matched like distributions
	tasker := NewArgonTasker(ctx, args.Broker)

	// open output file
	var output TraceOutputEncoder
	if args.Tracefile != "" && args.Broker != "" {
		output, err = OpenTraceOutputEncoder(args.Tracefile)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()
	}

	// receive all task responses in a single channel for logging
	responses := make(chan *transport.PendingCall, 2048)

	starter := tracebench.NewStarter[time.Time]()
	var runfor time.Duration

	// funcgen: spawn function generators
	if args.RunFuncgen != nil {
		cfg := args.RunFuncgen
		fmt.Printf("Loaded workload generator: %#v\n\n", cfg)
		runfor = cfg.Duration

		for _, workload := range cfg.Workloads {
			starter.Add(1)
			go func(w funcgen.WorkloadConfig) {
				for t := range w.TaskTriggers(starter) {
					fmt.Printf("%20s [%3d] elapsed: %9.6f, task: %9.6f\n", w.Name,
						t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
					// if diff := time.Since(t.Scheduled); diff > 10*time.Millisecond {
					// 	fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, t.Sequence, diff)
					// }
					tasker.Run(responses, t.Tasklen)
				}
			}(workload)
		}
	}

	// csvtrace: spawn function generators
	if args.RunCsvTrace != nil {
		cfg := args.RunCsvTrace
		fmt.Printf("Loaded CSV trace generator: %#v\n\n", cfg)
		runfor = cfg.Duration

		dataset := cfg.GetDataset()
		for _, column := range cfg.Columns {
			starter.Add(1)
			go func(col string) {
				for t := range dataset.TaskTriggers(starter, col) {
					fmt.Printf("column %3s [%3d] elapsed: %9.6f, task: %9.6f\n", col,
						t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
					if diff := time.Since(t.Scheduled); diff > 10*time.Millisecond {
						fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, t.Sequence, diff)
					}
					tasker.Run(responses, t.Tasklen)
				}
			}(column)
		}
	}

	// a single function to handle responses
	// TODO:
	// - tasks that never receive a response are lost
	// - erroneous responses are never logged
	go func() {
		for c := range responses {
			if c.Error != nil {
				fmt.Fprintf(os.Stderr, "ERR: %s\n%#v\n", c.Error, c.Response)
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
				}
			}
		}
	}()

	// everyone should be set up, start!
	starter.Wait()
	starter.Broadcast(time.Now())

	// signal handler to receive CTRL-C
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	select {

	case <-sigint:
		fmt.Println(" quit!")
		log.Println("interrupt received, exit ...")

	case <-time.After(runfor):
		// this does not wait for all responses to arrive
		log.Println("configured runtime reached, exit ...")

	}

}
