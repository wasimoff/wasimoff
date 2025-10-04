package funcgen

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
	"gopkg.in/yaml.v3"
	"wasi.team/client/tracebench/rng"
)

type TraceConfig struct {
	Name      string
	Seed      uint64
	Duration  time.Duration
	Workloads []WorkloadConfig
}

type WorkloadConfig struct {
	Name string
	Seed uint64
	Skip float64
	skip *CoinFlip
	Rate JitterDuration
	Task JitterDuration
}

// Predefined seed offsets for the various RNGs needed in a workload.
const (
	SeedSkip       uint64 = 10
	SeedRateJitter uint64 = 20
	SeedTaskJitter uint64 = 30
)

type JitterDuration struct {
	Fixed  time.Duration
	Jitter string
	jitter distuv.Rander
}

func ReadTraceConfig(filename string) (trace *TraceConfig, err error) {

	// open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open file: %w", err)
	}
	defer file.Close()

	// read config as yaml format
	trace = &TraceConfig{}
	err = yaml.NewDecoder(file).Decode(&trace)
	if err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}

	// make sure we have a name
	if trace.Name == "" {
		return nil, fmt.Errorf("please provide a name in config")
	}

	// make sure duration isn't zero
	if trace.Duration == 0 {
		return nil, fmt.Errorf("must provide a run duration")
	}

	// make sure we have workloads
	if len(trace.Workloads) == 0 {
		return nil, fmt.Errorf("must provide workloads, can't be empty")
	}

	// set all zero seeds to true random numbers
	if trace.Seed == 0 {
		trace.Seed = rng.TrueRandom()
	}
	for i := range trace.Workloads {
		w := &trace.Workloads[i]
		if w.Seed == 0 {
			w.Seed = rng.TrueRandom()
		}
	}

	// use deterministic rng sources for all distributions
	rngsource := rng.NewSeededSourcer(trace.Seed)

	// prepare all workloads
	for i := range trace.Workloads {
		w := &trace.Workloads[i]

		// coin flip for random skip
		w.skip, err = NewCoinFlip(w.Skip, rngsource.NewAtOffset(w.Seed, SeedSkip))
		if err != nil {
			return nil, fmt.Errorf("workload[%d].skip: %w", i, err)
		}

		// task interval distribution
		_, err = w.Rate.Prepare(rngsource.NewAtOffset(w.Seed, SeedRateJitter))
		if err != nil {
			return nil, fmt.Errorf("workload[%d].rate: %w", i, err)
		}

		// task length distribution
		_, err = w.Task.Prepare(rngsource.NewAtOffset(w.Seed, SeedTaskJitter))
		if err != nil {
			return nil, fmt.Errorf("workload[%d].task: %w", i, err)
		}

	}

	return trace, nil

}

// Preflight checks on the arguments and instantiate the distribution.
func (jd *JitterDuration) Prepare(rng rand.Source) (*JitterDuration, error) {
	if jd.Fixed < 0 {
		return nil, fmt.Errorf("fixed duration must not be zero or negative")
	}
	if jd.Fixed == 0 && jd.Jitter == "" {
		return nil, fmt.Errorf("must not have both zero fixed duration and no jitter")
	}
	jitter, err := ParseDistribution(jd.Jitter, rng)
	if err != nil {
		return nil, fmt.Errorf("failed to parse distribution: %w", err)
	}
	jd.jitter = jitter
	return jd, nil
}

func (w *WorkloadConfig) check() {
	if w.Rate.jitter == nil || w.Task.jitter == nil || w.skip == nil {
		panic("workload not properly initialized")
	}
}
