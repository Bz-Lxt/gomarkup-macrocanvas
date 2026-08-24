package timing

import "time"

// SpinBudget caps how long a Realtime wait may burn a core.
type SpinBudget struct {
	MaxSpin time.Duration
}

func DefaultSpinBudget() SpinBudget { return SpinBudget{MaxSpin: 50 * time.Millisecond} }

func (b SpinBudget) Clamp(d time.Duration, strat Strategy) (time.Duration, Strategy, bool) {
	if strat != Realtime {
		return d, strat, false
	}
	if d > b.MaxSpin {
		return d, Balanced, true
	}
	return d, strat, false
}
