package tracebench

// Limiter is using a buffered channel to limit the number of simultaneously
// in-flight requests. Acquiring puts an element into the chan, releaseing
// takes one out. If there is no space, acquiring fails immediately. When
// the limit is zero, acquiring always succeeds and releasing is a nop.
type Limiter struct {
	limit chan struct{}
}

func NewLimiter(limit uint64) *Limiter {
	if limit == 0 {
		return nil
	}
	return &Limiter{
		limit: make(chan struct{}, limit),
	}
}

func (l *Limiter) Acquire() bool {
	if l == nil {
		return true
	}
	select {
	case l.limit <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *Limiter) Release() {
	if l == nil {
		return
	}
	select {
	case <-l.limit:
		return
	default:
		panic("released more than acquired!")
	}
}
