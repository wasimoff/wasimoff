package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
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
	coln := os.Args[3]

	// read input file
	t0 := time.Now()
	trace := ReadDay(file, day)
	fmt.Printf("read full trace in %s\n", time.Since(t0))
	trace.SelectColumns([]string{coln})

	t0 = time.Now()
	spline := FitInterpolator(trace.RequestsPerSecond, coln)
	triggers := make(chan time.Time, 100)
	go InterpolatedFunctionTicker(context.Background(), t0, spline, triggers)

	triggered := make([]string, 0, 200)

	for t := range triggers {
		inst := t.Sub(t0)
		diff := time.Since(t)
		fmt.Printf("---> trigger! %s (%s)\n", inst, diff)
		triggered = append(triggered, inst.String())
		if t.Sub(t0) > 3*time.Second {
			break
		}
	}

	h := sha256.New()
	for _, s := range triggered {
		if _, err := h.Write([]byte(s)); err != nil {
			panic(err)
		}
	}
	fmt.Printf("hash so far: %s\n", hex.EncodeToString(h.Sum(nil)))
	time.Sleep(time.Minute)

	tickerUpdateTicker := time.Tick(time.Second)

	initial := spline.Predict(0)
	fmt.Printf("initial prediction: %f rq/s\n", initial)
	funcTicker := time.NewTicker(time.Duration(float64(time.Second) / initial))
	funcCounter := 0
	go func() {
		for t := range funcTicker.C {
			funcCounter += 1
			nominal := t.Sub(t0)
			fmt.Printf("function %6d at %s (%s)\n", funcCounter, nominal, time.Since(t0)-nominal)
		}
	}()

	for t := range tickerUpdateTicker {
		now := float64(t.Sub(t0).Seconds())
		rate := spline.Predict(now)
		funcTicker.Reset(time.Duration(float64(time.Second) / rate))
		fmt.Printf("TICKER --> %s set to Predict(%f) = %7.2f req/s\n", t, now, rate)
	}

}

func must[T any](v T, e error) T {
	if e != nil {
		panic(e)
	}
	return v
}
