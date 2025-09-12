package main

import (
	"context"
	"math"
	"time"

	"gonum.org/v1/gonum/interp"
)

func InterpolatedFunctionTicker(ctx context.Context, start time.Time, rate interp.Predictor, ticker chan<- Tick) {

	// use deteministic step instants and try to tick as close to it as possible
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxstep := time.Second
	armed := false
	count := 0

	for {

		// exit the loop with context
		select {
		case <-ctx.Done():
			close(ticker)
			return
		default:
		}

		elapsed := instant.Sub(start)

		// tick the channel if armed
		if armed {
			ticker <- Tick{instant, elapsed, count}
			count += 1
		}

		// interpolate the requests/sec and compute time until next tick
		rps := math.Max(rate.Predict(elapsed.Seconds()), 0)
		waitns := float64(time.Second) / rps
		wait := time.Duration(waitns)

		// if the step is too long, do not tick on next loop
		if math.IsInf(waitns, 0) || wait > maxstep {
			armed = false
			wait = maxstep
		} else {
			armed = true
		}

		// fmt.Printf("ticker %.2f rps @ %s, armed: %v, wait %s\n", rps, since, armed, wait)

		// go to sleep until next tick
		instant = instant.Add(wait)
		time.Sleep(time.Until(instant))

	}

}

type Tick struct {
	Scheduled time.Time
	Elapsed   time.Duration
	Counter   int
}
