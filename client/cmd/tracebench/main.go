package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

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
	// t0 := time.Now()
	trace := ReadDay(file, day)
	// fmt.Printf("read full trace in %s\n", time.Since(t0))
	trace.SelectColumns(columns)

	timeout, _ := context.WithTimeout(context.Background(), 120*time.Second)

	start := time.Now()

	wg := sync.WaitGroup{}
	for _, col := range columns {
		wg.Add(1)

		rps := FitInterpolator(trace.RequestsPerSecond, col)
		delay := FitInterpolator(trace.FunctionDelayAvgPerMinute, col)
		ticker := make(chan Tick, 100)
		go InterpolatedFunctionTicker(timeout, start, rps, ticker)

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
			}
			wg.Done()
		}()

	}

	wg.Wait()

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
