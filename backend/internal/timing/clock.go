package timing

import "time"

// Mono returns a monotonic nanosecond reading via time.Since(anchor).
type Mono struct {
	anchor time.Time
}

func NewMono() *Mono { return &Mono{anchor: time.Now()} }

func (m *Mono) Ns() int64 { return time.Since(m.anchor).Nanoseconds() }

func (m *Mono) Now() time.Time { return time.Now() }
