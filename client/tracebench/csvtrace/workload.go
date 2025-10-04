package csvtrace

import (
	"iter"
	"log"
	"time"

	"wasi.team/client/tracebench"
)

type NextTask struct {
	Elapsed time.Duration
	Tasklen time.Duration
}

// Iterator that yields the next task instant without actually sleeping.
func (hw HuaweiDataset) TaskIterator(column string) iter.Seq[NextTask] {

	// fit interpolators on function column
	tp := FitTracePredictors(hw, column)

	// get the timestamp from last row to detect (unrealistic) overflow
	limit := lastRowTime(hw.RequestsPerMinute)

	// track current time instant as elapsed time for interpolation
	elapsed := time.Duration(0)

	return func(yield func(NextTask) bool) {
		for {

			// overflowed the dataset
			if elapsed > limit {
				log.Printf("WARNING: overflowed the dataset in column %s, no more ticks", column)
				return
			}

			// use current elapsed to predict interval and next task instant
			interval := tp.IntervalAt(elapsed)
			if interval > time.Minute {
				elapsed = elapsed + time.Second
				continue
			}

			// add sleep interval and yield a tick
			elapsed = elapsed + interval
			if !yield(NextTask{
				Elapsed: elapsed,
				Tasklen: tp.TasklenAt(elapsed),
			}) {
				return
			}

		}
	}

}

// Iterator that follows a trace column. Attention: that means that this iterator will
// block and sleep according to the task instants! Use this in a goroutine to
// trigger function requests asynchronously.
func (hw HuaweiDataset) TaskTriggers(starter *tracebench.Starter[time.Time], column string) iter.Seq[tracebench.TaskTick] {

	tasks := hw.TaskIterator(column)
	sequence := uint64(0)

	start := starter.WaitForValue()
	instant := start

	return func(yield func(tracebench.TaskTick) bool) {
		for task := range tasks {

			// compute the next task trigger instant and sleep until then
			instant = start.Add(task.Elapsed)
			sleep := time.Until(instant)
			if sleep > time.Hour {
				log.Printf("WARNING: next task in column %s is far away: %v", column, sleep)
			}
			time.Sleep(sleep)

			// yield the next task
			if !yield(tracebench.TaskTick{
				Sequence:  sequence,
				Elapsed:   task.Elapsed,
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

// Find the last known instant in the frame.
func lastRowTime(column ColumnSet) time.Duration {
	t := column["time"]
	last := t[len(t)-1]
	return time.Duration(last * float64(time.Second))
}
