package csvtrace

import (
	"time"

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
	rpm := tp.RequestsPerMinute.Predict(i.Seconds())
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

// Fit an AkimaSplite predictor on a given dataset column
func fitPredictor(frame ColumnSet, col string) interp.Predictor {

	time := frame["time"]
	values := frame[col]

	// instantiate the predictor and fit
	spline := &interp.AkimaSpline{}
	// according to the docs it always returns nil ..
	err := spline.Fit(time, values)
	if err != nil {
		panic(err)
	}
	return spline

}
