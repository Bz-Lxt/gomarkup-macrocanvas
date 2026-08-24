package engine

import (
	"sync/atomic"
	"time"
)

type Safety struct {
	Stop       atomic.Bool
	DeadMan    atomic.Int64 // unix nano of last heartbeat
	MaxIters   int
	MaxWall    time.Duration
	Watchdog   time.Duration
}

func NewSafety(iters int, wallMs, watchMs int) *Safety {
	s := &Safety{MaxIters: iters, MaxWall: time.Duration(wallMs) * time.Millisecond}
	if watchMs <= 0 {
		watchMs = 5000
	}
	s.Watchdog = time.Duration(watchMs) * time.Millisecond
	s.Heartbeat()
	return s
}

func (s *Safety) Heartbeat() { s.DeadMan.Store(time.Now().UnixNano()) }

func (s *Safety) EmergencyStop() { s.Stop.Store(true) }

func (s *Safety) Reset() { s.Stop.Store(false); s.Heartbeat() }

func (s *Safety) Tripped(iters int, started time.Time) (bool, string) {
	if s.Stop.Load() {
		return true, "emergency_stop"
	}
	if s.MaxIters > 0 && iters >= s.MaxIters {
		return true, "max_iters"
	}
	if s.MaxWall > 0 && time.Since(started) >= s.MaxWall {
		return true, "wall_clock"
	}
	last := time.Unix(0, s.DeadMan.Load())
	if s.Watchdog > 0 && time.Since(last) > s.Watchdog {
		return true, "dead_man"
	}
	return false, ""
}
