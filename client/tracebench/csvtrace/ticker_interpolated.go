package csvtrace

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
	"wasi.team/client/tracebench"
)

func (hw *HuaweiDataset) InterpolatedRateTicker(
	ctx context.Context,
	column string,
	starter *tracebench.Starter[time.Time],
	ticker chan<- tracebench.TaskTick,
) {

	// fit interpolators on function column
	requestsPerMinute := fitPredictor(hw.RequestsPerMinute, column)
	taskDuration := fitPredictor(hw.FunctionDelayAvgPerMinute, column)

	// use deteministic step instants and try to tick as close to it as possible
	start := starter.WaitForValue()
	instant := start

	// only tick when armed (i.e. rate is below the maximum sleep length)
	maxSleep := 10 * time.Second
	armed := false
	seq := uint64(0)

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
			ticker <- tracebench.TaskTick{
				Scheduled: instant,
				Elapsed:   elapsed,
				Tasklen:   taskDuration.PredictDuration(elapsed),
				Sequence:  seq,
			}
			seq += 1
			// detect overflow
			if seq == 0 {
				panic("sequence number in ticker overflowed")
			}
		}

		// interpolate the requests/sec and compute time until next tick
		requestsPerSecond := requestsPerMinute.PredictRequestsPerSecond(elapsed)
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

// Fit an AkimaSplite predictor on a given QFrame dataset column
func fitPredictor(frame qframe.QFrame, col string) predictor {

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
	return predictor{spline}

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

// Wrap a predictor to take and return time instants more easily.
type predictor struct {
	predictor interp.Predictor
}

func (ti *predictor) PredictDuration(i time.Duration) time.Duration {
	p := ti.predictor.Predict(i.Seconds())
	return time.Duration(p * float64(time.Second))
}

func (ti *predictor) PredictRequestsPerSecond(i time.Duration) float64 {
	return math.Max(ti.predictor.Predict(i.Seconds())/60.0, 0)
}

const maximumDuration = time.Duration(1<<63 - 1)
const maximumDurationf64 = float64(maximumDuration)

func (ti *predictor) PredictNextTick(i time.Duration) time.Duration {
	// predict requests/minute at time instant
	rpm := ti.predictor.Predict(i.Seconds())
	// and compute time interval in nanoseconds
	wait := float64(time.Minute) / rpm
	if rpm <= 0 || wait >= maximumDurationf64 { // infinite duration
		return maximumDuration
	}
	return time.Duration(wait)
}
