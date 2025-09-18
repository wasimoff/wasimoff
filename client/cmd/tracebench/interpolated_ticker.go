package main

import (
	"context"
	"math"
	"time"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
)

func InterpolatedRateTicker(ctx context.Context, start time.Time, rate interp.Predictor, ticker chan<- Tick) {

	// use deteministic step instants and try to tick as close to it as possible
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxstep := time.Second
	armed := false
	seq := uint(0)

	for {

		// cancel the loop with context
		select {
		case <-ctx.Done():
			close(ticker)
			return
		default:
		}

		// current time since start
		elapsed := instant.Sub(start)

		// tick the channel if armed
		if armed {
			ticker <- Tick{instant, elapsed, seq}
			seq += 1
			// detect overflow
			if seq == 0 {
				panic("sequence number in ticker overflowed")
			}
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
	Sequence  uint
}

func FitInterpolator(frame qframe.QFrame, col string) interp.Predictor {

	// make sure the column does not contain NaNs or negative numbers
	frame = frame.Apply(qframe.Instruction{Fn: clampPositive, SrcCol1: col, DstCol: col})

	// get dataset as float slices
	time := frame.MustFloatView("time").Slice()
	values := frame.MustFloatView(col).Slice()

	// log.Println("values in column", col, "=>", values[:100], "...")

	// instantiate the predictor and fit
	spline := &interp.AkimaSpline{}
	// according to the docs it always returns nil ..
	err := spline.Fit(time, values)
	if err != nil {
		panic(err)
	}
	return spline

}

// Basically math.Max(0, f) but returns 0 on NaN as well.
func clampPositive(f float64) float64 {
	if math.IsNaN(f) || f < 0 {
		return 0.0
	}
	return f
}
