package timing

import (
	"runtime"
	"sort"
	"time"
)

type BenchResult struct {
	Strategy   Strategy        `json:"strategy"`
	TargetNs   int64           `json:"target_ns"`
	Samples    int             `json:"samples"`
	P50Ns      int64           `json:"p50_ns"`
	P90Ns      int64           `json:"p90_ns"`
	P99Ns      int64           `json:"p99_ns"`
	P999Ns     int64           `json:"p999_ns"`
	MaxNs      int64           `json:"max_ns"`
	Allocs     uint64          `json:"allocs"`
	GOMAXPROCS int             `json:"gomaxprocs"`
	Band       string          `json:"band"`
}

func BandForEnv(rtOK bool, inContainer bool) string {
	if rtOK && !inContainer {
		return "E1"
	}
	if !inContainer {
		return "E2"
	}
	return "E3"
}

func InContainer() bool {
	if _, err := osStat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

var osStat = func(p string) (any, error) {
	return osStatImpl(p)
}

func RunBench(p *Pacer, strat Strategy, target time.Duration, n int) BenchResult {
	if n < 20 {
		n = 20
	}
	errs := make([]time.Duration, 0, n)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < n; i++ {
		s := time.Now()
		p.Wait(target, strat)
		e := time.Since(s) - target
		if e < 0 {
			e = -e
		}
		errs = append(errs, e)
	}
	runtime.ReadMemStats(&after)
	sort.Slice(errs, func(i, j int) bool { return errs[i] < errs[j] })
	q := func(f float64) int64 { return int64(quantile(errs, f)) }
	return BenchResult{
		Strategy:   strat,
		TargetNs:   target.Nanoseconds(),
		Samples:    n,
		P50Ns:      q(0.50),
		P90Ns:      q(0.90),
		P99Ns:      q(0.99),
		P999Ns:     q(0.999),
		MaxNs:      int64(errs[len(errs)-1]),
		Allocs:     after.Mallocs - before.Mallocs,
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		Band:       BandForEnv(false, InContainer()),
	}
}
