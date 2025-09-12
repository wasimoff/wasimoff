package main

import (
	"fmt"
	"math"

	"github.com/tobgu/qframe"
	"gonum.org/v1/gonum/interp"
)

// compare different interpolators for the given dataset
func TryFitters(xs, ys []float64, end, step float64) {

	// instantiate and fit the predictors
	ak := &interp.AkimaSpline{}
	cc := &interp.ClampedCubic{}
	nc := &interp.NaturalCubic{}
	pl := &interp.PiecewiseLinear{}
	for _, predictor := range []interp.FittablePredictor{ak, cc, nc, pl} {
		predictor.Fit(xs, ys)
	}

	// predict interpolated values to stdout
	fmt.Printf("%9s, %12s, %12s, %12s, %16s\n", "time",
		"AkimaSpline", "ClampedCubic", "NaturalCubic", "PiecewiseLinear")
	for t := 0.0; t <= end; t += step {
		fmt.Printf("%9.1f, %12.1f, %12.1f, %12.1f, %16.1f\n", t,
			ak.Predict(t), cc.Predict(t), nc.Predict(t), pl.Predict(t))
	}

}

func FitInterpolator(frame qframe.QFrame, col string) interp.Predictor {

	// make sure the column does not contain NaNs or negative numbers
	frame = frame.Apply(qframe.Instruction{Fn: MustBePositiveNumber, SrcCol1: col, DstCol: col})

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

func MustBePositiveNumber(f float64) float64 {
	if math.IsNaN(f) || f < 0 {
		return 0.0
	}
	return f
}
