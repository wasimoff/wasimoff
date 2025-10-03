package funcgen

import (
	"iter"
	"math/rand/v2"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
	"wasi.team/client/tracebench"
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

// Preflight checks on the arguments and instantiate the distribution.
func (jd *JitterDuration) Prepare(rng rand.Source) *JitterDuration {
	if jd.Fixed < 0 {
		panic("fixed duration must not be less than zero")
	}
	if jd.Fixed == 0 && jd.Jitter == "" {
		panic("must not have both zero fixed duration and no jitter")
	}
	jitter, err := ParseDistribution(jd.Jitter, rng)
	if err != nil {
		panic(err)
	}
	jd.jitter = jitter
	return jd
}

// Return the next random duration as Fixed + Jitter.Rand()
func (jd *JitterDuration) NextDuration() time.Duration {
	return jd.Fixed + time.Duration(jd.jitter.Rand()*float64(time.Second))
}

type NextTask struct {
	Sleep   time.Duration
	Tasklen time.Duration
	Skip    bool
}

// Iterator that yields the next task properties without actually sleeping.
func (w WorkloadConfig) TaskIterator() iter.Seq[NextTask] {

	if w.Seed == 0 {
		w.Seed = rng.TrueRandom()
	}

	// preflight checks for rate and tasklen
	rate := w.Rate.Prepare(rng.NewOffsetRand(w.Seed, SeedRateJitter))
	task := w.Task.Prepare(rng.NewOffsetRand(w.Seed, SeedTaskJitter))
	skip := NewCoinFlip(w.Skip, rng.NewOffsetRand(w.Seed, SeedSkip))

	return func(yield func(NextTask) bool) {
		for {

			interval := rate.NextDuration()
			tasklen := task.NextDuration()

			if !yield(NextTask{
				Sleep:   max(interval, 0),
				Tasklen: max(tasklen, time.Millisecond),
				Skip:    skip.Next(),
			}) {
				return
			}

		}
	}

}

// Iterator that runs a workload. Attention: that means that this iterator will
// block and sleep according to the task instants! Use this in a goroutine to
// trigger function requests asynchronously.
func (w WorkloadConfig) TaskTriggers(starter *tracebench.Starter[time.Time]) iter.Seq[tracebench.TaskTick] {

	tasks := w.TaskIterator()
	sequence := uint64(0)

	start := starter.WaitForValue()
	instant := start
	elapsed := time.Duration(0)

	return func(yield func(tracebench.TaskTick) bool) {
		for task := range tasks {

			// compute the next task trigger instant and sleep until then
			instant = instant.Add(task.Sleep)
			elapsed = instant.Sub(start)
			time.Sleep(time.Until(instant))

			// maybe skip this task trigger
			if task.Skip {
				continue
			}

			// yield the next task
			if !yield(tracebench.TaskTick{
				Sequence:  sequence,
				Elapsed:   elapsed,
				Scheduled: instant,
				Tasklen:   task.Tasklen,
			}) {
				return
			}

			// increment and loop
			sequence++
			if sequence == 0 {
				panic("sequence number overflowed")
			}

		}
	}

}
