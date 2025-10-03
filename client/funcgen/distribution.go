package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

// could use (https://pkg.go.dev/gonum.org/v1/gonum/stat/distuv)
// * Bernoulli(p)
// - Beta(alpha, beta)
// - Binomial(n, p)
// - Chi(k)
// * Exponential(rate)
// - Gamma(alpha, beta)
// + GumbelRight(mu, beta)
// * Laplace(mu, scale)
// * Normal(mu, sigma)
// + LogNormal(mu, sigma)
// - Logistic(mu, scale)
// + Pareto(xm, alpha)
// + Poisson(lamba)
// - StudentsT(mu, sigma, nu)
// - Triangle(a, b, c)
// * Uniform(min, max)
// - Weibull(k, lambda)

// Match a string prefix to a distuv distribution and parse its function
// arguments using a regular expression.
func ParseDistribution(s string, rng rand.Source) (distuv.Rander, error) {

	// use deterministic rng
	if rng == nil {
		rng = NewRand(0)
	}

	// find the correct parser for distribution
	s = strings.ToLower(strings.TrimSpace(s))
	switch true {

	case s == "":
		return &Never{}, nil

	case strings.HasPrefix(s, "bernoulli("):
		return ParseBernoulli(s, rng)

	case strings.HasPrefix(s, "exponential("):
		return ParseExponential(s, rng)

	case strings.HasPrefix(s, "laplace("):
		return ParseLaplace(s, rng)

	case strings.HasPrefix(s, "normal("):
		return ParseNormal(s, rng)

	case strings.HasPrefix(s, "uniform("):
		return ParseUniform(s, rng)

	default:
		return nil, fmt.Errorf("unknown distribution: %s", s)
	}
}

// parse bernoulli(:time, :probability) into distuv.Bernoulli
func ParseBernoulli(s string, rng rand.Source) (distuv.Rander, error) {
	// match the string
	const re = `^bernoulli\(\s*([-+\d\.nmush]+)\s*,\s*([+-]?\d*\.?\d*)\s*\)$`
	matches := regexp.MustCompile(re).FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid format: expected bernoulli(:time, :prob)")
	}
	// convert the arguments
	t, err := time.ParseDuration(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid time: %v", err)
	}
	prob, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid probability: %v", err)
	}
	if !(0 < prob && prob < 1) {
		return nil, fmt.Errorf("invalid probability: must be between 0 and 1")
	}
	// instantiate distribution
	return &bernoulliRander{
		bernoulli: &distuv.Bernoulli{
			Src: rng,
			P:   prob,
		},
		seconds: t.Seconds(),
	}, nil
}

type bernoulliRander struct {
	bernoulli *distuv.Bernoulli
	seconds   float64
}

func (b *bernoulliRander) Rand() float64 {
	return b.bernoulli.Rand() * b.seconds
}

// parse exponential(:rate) into distuv.Exponential
func ParseExponential(s string, rng rand.Source) (distuv.Rander, error) {
	// match the string
	const re = `^exponential\(\s*([+-]?\d*\.?\d*)\s*\)$`
	matches := regexp.MustCompile(re).FindStringSubmatch(s)
	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid format: expected exponential(:rate/s)")
	}
	// convert the arguments
	rate, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rate: %v", err)
	}
	// instantiate distribution
	return &distuv.Exponential{
		Src:  rng,
		Rate: rate,
	}, nil
}

// parse laplace(:mu, :scale) into distuv.Exponential
func ParseLaplace(s string, rng rand.Source) (distuv.Rander, error) {
	// match the string
	const re = `^laplace\(\s*([-+\d\.nmush]+)\s*,\s*([-+\d\.nmush]+)\s*\)$`
	matches := regexp.MustCompile(re).FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid format: expected laplace(:mu, :scale)")
	}
	// convert the arguments
	mu, err := time.ParseDuration(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid mu: %v", err)
	}
	scale, err := time.ParseDuration(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid scale: %v", err)
	}
	if !(scale > 0) {
		return nil, fmt.Errorf("invalid scale: must be larger than 0")
	}
	// instantiate distribution
	return &distuv.Laplace{
		Src:   rng,
		Mu:    mu.Seconds(),
		Scale: scale.Seconds(),
	}, nil
}

// parse normal(mu, sigma) into distuv.Normal
func ParseNormal(s string, rng rand.Source) (distuv.Rander, error) {
	// match the string
	// const re = `^normal\(\s*([+-]?\d*\.?\d*)\s*,\s*([+-]?\d*\.?\d*)\s*\)$`
	const re = `^normal\(\s*([-+\d\.nmush]+)\s*,\s*([-+\d\.nmush]+)\s*\)$`
	matches := regexp.MustCompile(re).FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid format: expected normal(:mu, :sigma)")
	}
	// convert the arguments
	// mu, err := strconv.ParseFloat(matches[1], 64)
	mu, err := time.ParseDuration(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid mu: %v", err)
	}
	// sigma, err := strconv.ParseFloat(matches[2], 64)
	sigma, err := time.ParseDuration(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid sigma: %v", err)
	}
	if sigma < 0 {
		return nil, fmt.Errorf("invalid sigma: must be positive")
	}
	// instantiate distribution
	return &distuv.Normal{
		Src:   rng,
		Mu:    mu.Seconds(),
		Sigma: sigma.Seconds(),
	}, nil
}

// parse uniform(min, max) into distuv.Uniform
func ParseUniform(s string, rng rand.Source) (distuv.Rander, error) {
	// match the string
	// const re = `^uniform\(\s*([+-]?\d*\.?\d*)\s*,\s*([+-]?\d*\.?\d*)\s*\)$`
	const re = `^uniform\(\s*([-+\d\.nmush]+)\s*,\s*([-+\d\.nmush]+)\s*\)$`
	matches := regexp.MustCompile(re).FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid format: expected uniform(:min, :max)")
	}
	// convert the arguments
	min, err := time.ParseDuration(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid min: %v", err)
	}
	max, err := time.ParseDuration(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid max: %v", err)
	}
	if !(min < max) {
		return nil, fmt.Errorf("invalid parameters: min should be smaller than max")
	}
	// instantiate distribution
	return &distuv.Uniform{
		Src: rng,
		Min: min.Seconds(),
		Max: max.Seconds(),
	}, nil
}

// Simplest distribution which always returns 0.
type Never struct{}

func (*Never) Rand() float64 {
	return 0
}

// Boolean coin flip based on a bernoulli distribution.
func NewCoinFlip(p float64, rng rand.Source) CoinFlip {
	if rng == nil {
		rng = NewRand(0)
	}
	if p == 0 {
		return CoinFlip{&Never{}}
	} else {
		if !(0 <= p && p <= 1) {
			panic("Bernoulli probability must be in [0, 1]")
		}
		return CoinFlip{distuv.Bernoulli{
			Src: rng,
			P:   p,
		}}
	}
}

type CoinFlip struct {
	dist distuv.Rander
}

func (cf *CoinFlip) Next() bool {
	return cf.dist.Rand() == 1
}
