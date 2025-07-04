package provider

import (
	"sync"
	"time"
)

// RateCounter is used to observe the rate of requests manually
// as extracting this information from a Prometheus HistogramVec
// appears to be quite complicated unfortunately ..
type RateCounter struct {
	mu           sync.Mutex
	observations []time.Time
	windowlen    time.Duration
}

func NewRateCounter(window time.Duration) *RateCounter {
	return &RateCounter{
		observations: make([]time.Time, 0),
		windowlen:    window,
	}
}

// Observe adds a new observation to this counter
func (r *RateCounter) Observe() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observations = append(r.observations, time.Now())
	r.truncate()
}

// Clean up old observations by slicing from the oldest observation
// that is still within the window; loop exits as soon as the first
// one is newer than that.
func (r *RateCounter) truncate() {
	cutoff := time.Now().Add(-r.windowlen)
	for len(r.observations) > 0 && r.observations[0].Before(cutoff) {
		r.observations = r.observations[1:]
	}
}

// Return the current rate per window. Make sure to truncate before
// to clean up old observations even if there were no recent ones.
func (r *RateCounter) GetRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.truncate()
	return float64(len(r.observations)) / r.windowlen.Seconds()
}
