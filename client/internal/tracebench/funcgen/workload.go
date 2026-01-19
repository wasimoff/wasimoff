package funcgen

import (
	"context"
	"iter"
	"time"

	"wasi.team/client/internal/tracebench"
)

// Return the next random duration as Fixed + Jitter.Rand()
func (jd *JitterDuration) Next() time.Duration {
	return jd.Fixed + time.Duration(jd.jitter.Rand()*float64(time.Second))
}

type NextTask struct {
	Sleep   time.Duration
	Tasklen time.Duration
	Skip    bool
}

// Iterator that yields the next task properties without actually sleeping.
func (w WorkloadConfig) TaskIterator() iter.Seq[NextTask] {
	w.check()

	return func(yield func(NextTask) bool) {
		for {

			interval := w.Rate.Next()
			tasklen := w.Task.Next()

			if !yield(NextTask{
				Sleep:   max(interval, 0),
				Tasklen: max(tasklen, time.Millisecond),
				Skip:    w.skip.Next(),
			}) {
				return
			}

		}
	}

}

// Iterator that runs a workload. Attention: that means that this iterator will
// block and sleep according to the task instants! Use this in a goroutine to
// trigger function requests asynchronously.
func (w WorkloadConfig) TaskTriggers(ctx context.Context, starter *tracebench.Starter[time.Time]) iter.Seq[tracebench.TaskTick] {

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
			// sleep with cancellation
			select {
			case <-time.After(time.Until(instant)): // ok
			case <-ctx.Done(): // cancelled
				return
			}

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
