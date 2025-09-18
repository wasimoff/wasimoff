package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"wasi.team/broker/net/transport"
	wasimoffv1 "wasi.team/proto/v1"
)

const broker = "http://localhost:4080/"

func main() {

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: tracebench <download:dir> <day:int> <col:str>")
		os.Exit(1)
	}

	// parse args
	file := os.Args[1]
	day := must(strconv.Atoi(os.Args[2]))
	columns := os.Args[3:]

	// read input file
	trace := ReadDay(file, day)
	trace.SelectColumns(columns)

	timeout, _ := context.WithTimeout(context.Background(), 120*time.Second)

	tb := Connect(timeout, broker)

	start := time.Now()

	responses := make(chan *transport.PendingCall, 2048)

	wg := sync.WaitGroup{}
	for _, col := range columns {
		wg.Add(1)

		rps := FitInterpolator(trace.RequestsPerSecond, col)
		delay := FitInterpolator(trace.FunctionDelayAvgPerMinute, col)
		ticker := make(chan Tick, 100)
		at := NewArgonTasker(tb)
		go InterpolatedRateTicker(timeout, start, rps, ticker)

		go func() {
			i := 0
			for t := range ticker {
				i += 1
				diff := time.Since(t.Scheduled)
				funcdelay := delay.Predict(t.Elapsed.Seconds())
				if diff > 10*time.Millisecond {
					fmt.Fprintf(os.Stderr, "WARN: [ %3s : %4d ] far from scheduled tick: %s\n", col, i, diff)
				}
				fmt.Printf("tick[ %3s ]  %s --> %f ms\n", col, t.Elapsed, funcdelay)
				// TODO: needs to actually vary the task duration now
				at.Run(responses, 10)
			}
			wg.Done()
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
			}
		}
	}()

	wg.Wait()

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
