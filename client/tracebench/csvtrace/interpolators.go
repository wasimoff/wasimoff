package csvtrace

import (
	"math"
	"time"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
)

// Wrap two predictors to evaluate dataset at specific time instants.
type TracePredictors struct {
	RequestsPerMinute interp.Predictor
	Tasklen           interp.Predictor
}

// Evaluate the average task length at a specific time.
func (tp *TracePredictors) TasklenAt(i time.Duration) time.Duration {
	p := tp.Tasklen.Predict(i.Seconds())
	return time.Duration(p * float64(time.Second))
}

// Evaluate the interval to next task at a specific time.
func (tp *TracePredictors) IntervalAt(i time.Duration) time.Duration {

	// maximum values needed to detect duration overflow
	const maximumDuration = time.Duration(1<<63 - 1)
	const maximumDurationf64 = float64(maximumDuration)

	// predict requests/minute and time interval in nanoseconds
	rpm := tp.Tasklen.Predict(i.Seconds())
	waitns := float64(time.Minute) / rpm
	if rpm <= 0 || waitns >= maximumDurationf64 { // infinite duration
		return maximumDuration
	}
	return time.Duration(waitns)
}

func FitTracePredictors(dataset HuaweiDataset, col string) TracePredictors {
	return TracePredictors{
		RequestsPerMinute: fitPredictor(dataset.RequestsPerMinute, col),
		Tasklen:           fitPredictor(dataset.FunctionDelayAvgPerMinute, col),
	}
}

// Fit an AkimaSplite predictor on a given QFrame dataset column
func fitPredictor(frame qframe.QFrame, col string) interp.Predictor {

	// make sure the column does not contain NaNs or negative numbers
	fr := frame.Apply(qframe.Instruction{Fn: clampPositive, SrcCol1: col, DstCol: col})

	// get dataset as float slices
	time := fr.MustFloatView("time").Slice()
	values := fr.MustFloatView(col).Slice()

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
