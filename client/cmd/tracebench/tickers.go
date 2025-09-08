package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"gonum.org/v1/gonum/interp"
)

func InterpolatedFunctionTicker(ctx context.Context, start time.Time, rate interp.Predictor, ticker chan<- time.Time) {

	// use deteministic step instants and try to tick as close to it as possible
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxstep := time.Second
	armed := false

	for {

		// exit the loop with context
		select {
		case <-ctx.Done():
			return
		default:
		}

		// tick the channel if armed
		if armed {
			ticker <- instant
		}

		// interpolate the requests/sec and compute time until next tick
		rps := math.Max(rate.Predict(instant.Sub(start).Seconds()), 0)
		waitns := float64(time.Second) / rps
		wait := time.Duration(waitns)
		fmt.Printf("ticker: %s, %f rps, wait for %s\n", instant.Sub(start), rps, wait)

		// if the step is too long, do not tick on next loop
		if math.IsInf(waitns, 0) || wait > maxstep {
			armed = false
			wait = maxstep
		} else {
			armed = true
		}

		// go to sleep until next tick
		instant = instant.Add(wait)
		time.Sleep(time.Until(instant))

	}

}
