package tracebench

import "time"

type TaskTick struct {
	Sequence  uint64
	Elapsed   time.Duration
	Scheduled time.Time
	Tasklen   time.Duration
}
