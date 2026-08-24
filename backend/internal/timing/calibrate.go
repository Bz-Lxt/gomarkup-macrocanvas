package timing

import (
	"sort"
	"time"
)

type Calibration struct {
	SleepP50 time.Duration
	SleepP99 time.Duration
	SleepP999 time.Duration
	Margin   time.Duration
	Samples  int
}

// CalibrateSleep measures time.Sleep overshoot and returns a spin margin
// at the p99.9 of the 1ms overshoot distribution (feasibility REPORT B-3).
func CalibrateSleep(n int) Calibration {
	if n < 32 {
		n = 32
	}
	target := time.Millisecond
	ov := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		s := time.Now()
		time.Sleep(target)
		d := time.Since(s) - target
		if d < 0 {
			d = 0
		}
		ov = append(ov, d)
	}
	sort.Slice(ov, func(i, j int) bool { return ov[i] < ov[j] })
	c := Calibration{
		SleepP50:  quantile(ov, 0.50),
		SleepP99:  quantile(ov, 0.99),
		SleepP999: quantile(ov, 0.999),
		Samples:   n,
	}
	c.Margin = c.SleepP999
	if c.Margin < 250*time.Microsecond {
		c.Margin = 250 * time.Microsecond
	}
	// cap: never spin-sleep more than 25ms of margin (CPU budget)
	if c.Margin > 25*time.Millisecond {
		c.Margin = 25 * time.Millisecond
	}
	return c
}

func quantile(s []time.Duration, q float64) time.Duration {
	if len(s) == 0 {
		return 0
	}
	i := int(q * float64(len(s)))
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}
