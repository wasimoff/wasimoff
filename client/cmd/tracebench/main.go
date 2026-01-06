package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
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

	// child context for task iterators
	iterctx, canceliters := context.WithCancel(ctx)

	// waitgroup to count number of requests in-flight
	var tasks sync.WaitGroup

	// create the request generator
	// allows for different task generators, could be string-matched like distributions
	tasker := NewArgonTasker(ctx, &tasks, args.Broker)

	// open output file
	var output TraceOutputEncoder
	if args.Tracefile != "" && args.Broker != "" {

		output, err = OpenTraceOutputEncoder(args.Tracefile)
		if err != nil {
			log.Fatal(err)
		}
		defer output.Close()

		// maybe start profiling as well
		if args.Pprof {
			stopProfiling, err := StartProfiling(args.Tracefile)
			if err != nil {
				log.Fatal(err)
			}
			defer stopProfiling()
		}

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
				for t := range w.TaskTriggers(iterctx, starter) {
					fmt.Printf("%20s [%3d] elapsed: %9.6f, task: %9.6f\n", w.Name,
						t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
					// if diff := time.Since(t.Scheduled); diff > 10*time.Millisecond {
					// 	fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, t.Sequence, diff)
					// }
					tasker.Run(responses, t.Tasklen, w.Name)
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
				name := "col:" + col
				for t := range dataset.TaskTriggers(iterctx, starter, col) {
					fmt.Printf("column %3s [%3d] elapsed: %9.6f, task: %9.6f\n", col,
						t.Sequence, t.Elapsed.Seconds(), t.Tasklen.Seconds())
					if diff := time.Since(t.Scheduled); diff > 10*time.Millisecond {
						fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, t.Sequence, diff)
					}
					tasker.Run(responses, t.Tasklen, name)
				}
			}(column)
		}
	}

	// if the configured duration is zero, effectively run ~forever
	if runfor == 0 {
		runfor = 1<<63 - 1 // 292 years
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
			tasks.Done()
		}
	}()

	// everyone should be set up, start!
	starter.Wait()
	starter.Broadcast(time.Now())

	// signal handler to receive CTRL-C
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// wait for signal or timeout
	select {
	case <-sigint:
		fmt.Println(" interrupt!")
		log.Println("signal received, stopping workloads ...")
	case <-time.After(runfor):
		canceliters()
		log.Println("configured runtime reached, stopping workloads ...")
	}
	canceliters()

	// quit immediately if not instructed to wait
	if !args.Wait {
		log.Println("quit immediately. pass -wait to wait for all tasks.")
		return
	}

	// setup to wait for the waitgroup asynchronously
	quit := make(chan struct{})
	go func() {
		tasks.Wait()
		quit <- struct{}{}
	}()

	// setup a grace timeout
	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(60 * time.Second):
			log.Println("60s grace timeout reached, exit now")
			quit <- struct{}{}
		}
	}()

	// wait for finish, with grace period or instant abort on (another) signal
	select {

	case <-quit: // normal exit

	case <-sigint: // cancel immediately
		fmt.Println(" abort!")
		os.Exit(1)

	}

}
