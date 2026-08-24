package timing

import (
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"
)

type Strategy string

const (
	Realtime  Strategy = "realtime"
	Balanced  Strategy = "balanced"
	Efficient Strategy = "efficient"
)

type Pacer struct {
	margin   atomic.Int64
	spins    atomic.Uint64
	overruns atomic.Uint64
}

func NewPacer(cal Calibration) *Pacer {
	p := &Pacer{}
	p.SetMargin(cal.Margin)
	return p
}

func (p *Pacer) SetMargin(d time.Duration) { p.margin.Store(int64(d)) }
func (p *Pacer) Margin() time.Duration     { return time.Duration(p.margin.Load()) }

// Wait blocks until d has elapsed. Hot path allocates nothing.
func (p *Pacer) Wait(d time.Duration, strat Strategy) {
	if d <= 0 {
		return
	}
	deadline := time.Now().Add(d)
	switch strat {
	case Efficient:
		time.Sleep(d)
	case Realtime:
		for time.Now().Before(deadline) {
			p.spins.Add(1)
		}
	default:
		m := time.Duration(p.margin.Load())
		if d > m && m > 0 {
			time.Sleep(d - m)
		}
		for time.Now().Before(deadline) {
			p.spins.Add(1)
		}
	}
	if time.Now().After(deadline.Add(time.Millisecond)) {
		p.overruns.Add(1)
	}
}

func (p *Pacer) Stats() (spins, overruns uint64) {
	return p.spins.Load(), p.overruns.Load()
}

// LockThread pins the calling goroutine and optionally suppresses GC.
func LockThread(disableGC bool) (unlock func()) {
	runtime.LockOSThread()
	prev := debug.SetGCPercent(100)
	if disableGC {
		prev = debug.SetGCPercent(-1)
	}
	return func() {
		debug.SetGCPercent(prev)
		runtime.UnlockOSThread()
	}
}
