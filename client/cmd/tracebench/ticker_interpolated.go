package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
)

func (t *HuaweiDataset) InterpolatedRateTicker(ctx context.Context, column string, starter *Starter[time.Time], ticker chan<- Tick) {

	// fit interpolators on function column
	col := column
	requestsPerMinute := fitPredictor(t.RequestsPerMinute, column)
	taskDuration := fitPredictor(t.FunctionDelayAvgPerMinute, column)

	// use deteministic step instants and try to tick as close to it as possible
	start := starter.WaitForValue()
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxSleep := 10 * time.Second
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
			tasklen := taskDuration.Predict(elapsed.Seconds())
			ticker <- Tick{
				Column:     &col,
				Scheduled:  instant,
				Elapsed:    elapsed,
				TasklenSec: tasklen,
				Sequence:   seq,
			}
			seq += 1
			// detect overflow
			if seq == 0 {
				panic("sequence number in ticker overflowed")
			}
		}

		// interpolate the requests/sec and compute time until next tick
		requestsPerSecond := math.Max(requestsPerMinute.Predict(elapsed.Seconds())/60.0, 0)
		waitNano := float64(time.Second) / requestsPerSecond
		wait := time.Duration(waitNano)

		// if the step is too long, do not tick on next loop
		if math.IsInf(waitNano, 0) || wait > maxSleep {
			if armed || seq == 0 {
				fmt.Printf("[ %3s ] Note: Ticker disarmed because req/s at is zero at %v\n", column, elapsed)
			}
			armed = false
			wait = maxSleep
		} else {
			armed = true
		}

		// fmt.Printf("ticker %.2f rps @ %s, armed: %v, wait %s\n", rps, since, armed, wait)

		// go to sleep until next tick
		instant = instant.Add(wait)
		time.Sleep(time.Until(instant))

	}

}

// Tick is a single event that should trigger a request
type Tick struct {
	Column     *string
	Scheduled  time.Time
	Elapsed    time.Duration
	TasklenSec float64
	Sequence   uint
}

// Fit an AkimaSplite predictor on a given QFrame dataset column
func fitPredictor(frame qframe.QFrame, col string) interp.Predictor {

	// make sure the column does not contain NaNs or negative numbers
	fr := frame.Apply(qframe.Instruction{Fn: clampPositive, SrcCol1: col, DstCol: col})

	// get dataset as float slices
	time := fr.MustFloatView("time").Slice()
	values := fr.MustFloatView(col).Slice()

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

// Multiply all values in the column with a scalar.
func scaleColumn(scale float64) func(float64) float64 {
	return func(f float64) float64 {
		return f * scale
	}
}
