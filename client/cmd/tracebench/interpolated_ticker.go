package main

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
)

func (h *HuaweiDataset) InterpolatedRateTicker(ctx context.Context, column string, starter *Starter[time.Time], ticker chan<- Tick) {

	// fit interpolators on function column
	requestsPerMinute := FitInterpolator(h.RequestsPerMinute, column)
	taskDuration := FitInterpolator(h.FunctionDelayAvgPerMinute, column)

	// use deteministic step instants and try to tick as close to it as possible
	start := starter.WaitForValue()
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxSleep := 60 * time.Second
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
			dur := taskDuration.Predict(elapsed.Seconds())
			ticker <- Tick{instant, elapsed, dur, seq}
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

type Tick struct {
	Scheduled time.Time
	Elapsed   time.Duration
	Tasklen   float64
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

type Starter[T any] struct {
	setup  sync.WaitGroup
	signal chan struct{}
	value  T
}

func NewStarter[T any]() *Starter[T] {
	return &Starter[T]{signal: make(chan struct{})}
}

func (s *Starter[T]) Add(delta int) {
	s.setup.Add(delta)
}

func (s *Starter[T]) Wait() {
	s.setup.Wait()
}

func (s *Starter[T]) Broadcast(value T) {
	s.setup.Wait()
	s.value = value
	close(s.signal)
}

func (s *Starter[T]) WaitForValue() T {
	s.setup.Done()
	<-s.signal
	return s.value
}
