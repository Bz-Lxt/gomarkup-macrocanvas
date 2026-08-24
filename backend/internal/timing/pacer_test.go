package timing

import (
	"testing"
	"time"
)

func TestPacerZeroAllocRealtime(t *testing.T) {
	p := NewPacer(Calibration{Margin: time.Millisecond})
	allocs := testing.AllocsPerRun(200, func() {
		p.Wait(50*time.Microsecond, Realtime)
	})
	if allocs != 0 {
		t.Fatalf("allocs=%v want 0", allocs)
	}
}

func TestCalibratePositive(t *testing.T) {
	c := CalibrateSleep(40)
	if c.Margin <= 0 {
		t.Fatalf("margin=%v", c.Margin)
	}
}

func TestRTNeverPanics(t *testing.T) {
	_, reason := RTAvailable()
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestSpinBudgetDowngrade(t *testing.T) {
	b := DefaultSpinBudget()
	_, st, degraded := b.Clamp(200*time.Millisecond, Realtime)
	if !degraded || st != Balanced {
		t.Fatalf("st=%s deg=%v", st, degraded)
	}
}
