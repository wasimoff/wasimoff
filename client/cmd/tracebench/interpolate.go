package main

import (
	"fmt"

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
