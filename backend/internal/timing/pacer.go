package timing

import (
	"context"
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
	p.waitCtx(context.Background(), d, strat)
}

// WaitCtx blocks until d elapses or ctx is cancelled, whichever comes first.
// Returns ctx.Err() (nil on natural completion). This is the cancellation-aware
// entry point used by the executor so that long holds inside key.tap / waits
// can be interrupted promptly by an emergency stop or run cancellation.
func (p *Pacer) WaitCtx(ctx context.Context, d time.Duration, strat Strategy) error {
	return p.waitCtx(ctx, d, strat)
}

func (p *Pacer) waitCtx(ctx context.Context, d time.Duration, strat Strategy) error {
	if d <= 0 {
		return ctx.Err()
	}
	deadline := time.Now().Add(d)
	switch strat {
	case Efficient:
		if ctx.Done() != nil {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-t.C:
			case <-ctx.Done():
			}
		} else {
			time.Sleep(d)
		}
	case Realtime:
		for time.Now().Before(deadline) {
			p.spins.Add(1)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	default:
		m := time.Duration(p.margin.Load())
		if d > m && m > 0 {
			sleep := d - m
			if ctx.Done() != nil {
				t := time.NewTimer(sleep)
				select {
				case <-t.C:
				case <-ctx.Done():
					t.Stop()
					return ctx.Err()
				}
			} else {
				time.Sleep(sleep)
			}
		}
		for time.Now().Before(deadline) {
			p.spins.Add(1)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
	if time.Now().After(deadline.Add(time.Millisecond)) {
		p.overruns.Add(1)
	}
	return ctx.Err()
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
