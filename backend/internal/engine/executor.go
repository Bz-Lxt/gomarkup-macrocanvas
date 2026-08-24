package engine

import (
	"context"
	"math/rand"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

type Injector interface {
	Inject(ev hid.Event) error
}

type Result struct {
	RunID     string    `json:"run_id"`
	MacroID   string    `json:"macro_id"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	Iters     int       `json:"iters"`
	ElapsedNs int64     `json:"elapsed_ns"`
	Trace     Trace     `json:"trace"`
	Markers   []string  `json:"markers"`
	StartedAt string    `json:"started_at"`
	EndedAt   string    `json:"ended_at"`
}

type Executor struct {
	sink   Injector
	pacer  *timing.Pacer
	safety *Safety
	rng    *rand.Rand
}

func NewExecutor(sink Injector, pacer *timing.Pacer, safety *Safety) *Executor {
	return &Executor{sink: sink, pacer: pacer, safety: safety, rng: rand.New(rand.NewSource(1))}
}

func (e *Executor) Safety() *Safety { return e.safety }

func (e *Executor) Run(ctx context.Context, runID, macroID string, p *Program) Result {
	unlock := timing.LockThread(p.Precision == "realtime")
	defer unlock()

	strat := timing.Balanced
	switch p.Precision {
	case "realtime":
		strat = timing.Realtime
	case "efficient":
		strat = timing.Efficient
	}
	budget := timing.DefaultSpinBudget()

	var keys [256]bool
	var btns [8]bool
	var vars [16]int64
	markers := make([]string, 0, 8)
	trace := Trace{Entries: make([]TraceEntry, 0, 128)}

	started := time.Now()
	plan := int64(0)
	pc := 0
	iters := 0
	status, reason := "succeeded", ""

	inject := func(kind hid.EventKind, page, usage uint16, val int32) {
		if ctx.Err() != nil {
			return
		}
		_ = e.sink.Inject(hid.Event{Kind: kind, Page: page, Usage: usage, Value: val, Source: hid.SourceInjected})
	}
	releaseAll := func() {
		for u, d := range keys {
			if d {
				inject(hid.KindKey, hid.PageKeyboard, uint16(u), 0)
				keys[u] = false
			}
		}
		for u, d := range btns {
			if d {
				inject(hid.KindButton, hid.PageButton, uint16(u), 0)
				btns[u] = false
			}
		}
	}
	defer releaseAll()

	for pc >= 0 && pc < len(p.Ops) {
		if ctx.Err() != nil {
			status, reason = "cancelled", "context"
			break
		}
		if trip, why := e.safety.Tripped(iters, started); trip {
			status, reason = "stopped", why
			break
		}
		op := p.Ops[pc]
		iters++
		act0 := time.Since(started).Nanoseconds()
		switch op.Kind {
		case OpKeyDown:
			inject(hid.KindKey, op.Page, op.Usage, 1)
			if op.Usage < 256 {
				keys[op.Usage] = true
			}
		case OpKeyUp:
			inject(hid.KindKey, op.Page, op.Usage, 0)
			if op.Usage < 256 {
				keys[op.Usage] = false
			}
		case OpMouseRel:
			inject(hid.KindPointer, op.Page, op.Usage, op.Value)
		case OpMouseAbs:
			inject(hid.KindPointer, hid.PageGenericDesktop, hid.GDX, op.Value)
			inject(hid.KindPointer, hid.PageGenericDesktop, hid.GDY, int32(op.Imm))
		case OpMouseBtn:
			inject(hid.KindButton, op.Page, op.Usage, op.Value)
			if op.Usage < 8 {
				btns[op.Usage] = op.Value != 0
			}
		case OpMouseWheel:
			inject(hid.KindPointer, op.Page, op.Usage, op.Value)
		case OpWait:
			d, st, _ := budget.Clamp(time.Duration(op.DelayNs), strat)
			e.pacer.Wait(d, st)
			plan += op.DelayNs
		case OpWaitRand:
			lo, hi := op.DelayNs, op.Imm
			if hi < lo {
				hi = lo
			}
			span := hi - lo
			var n int64
			if span > 0 {
				n = lo + e.rng.Int63n(span+1)
			} else {
				n = lo
			}
			d, st, _ := budget.Clamp(time.Duration(n), strat)
			e.pacer.Wait(d, st)
			plan += n
		case OpJump:
			pc += int(op.Jump)
			continue
		case OpJumpIf:
			if evalCond(op, vars, time.Since(started)) {
				pc += int(op.Jump)
				continue
			}
		case OpVarSet:
			if op.VarID >= 0 && int(op.VarID) < len(vars) {
				vars[op.VarID] = op.Imm
			}
		case OpVarInc:
			if op.VarID >= 0 && int(op.VarID) < len(vars) {
				vars[op.VarID] += op.Imm
			}
		case OpMarker:
			markers = append(markers, op.Label)
		case OpBreak:
			// fall through to next
		case OpHalt:
			status = "succeeded"
			pc = len(p.Ops)
			continue
		}
		act := time.Since(started).Nanoseconds()
		if op.Kind == OpWait || op.Kind == OpWaitRand || op.Kind == OpKeyDown || op.Kind == OpMarker {
			trace.Entries = append(trace.Entries, TraceEntry{
				Index: len(trace.Entries), NodeID: op.NodeID, Kind: kindName(op.Kind),
				PlanNs: plan, ActualNs: act, ErrorNs: act - act0 - op.DelayNs, Label: op.Label,
			})
		}
		pc++
	}
	summarizeTrace(&trace)
	if status == "succeeded" && reason == "" && ctx.Err() != nil {
		status, reason = "cancelled", "context"
	}
	return Result{
		RunID: runID, MacroID: macroID, Status: status, Reason: reason,
		Iters: iters, ElapsedNs: time.Since(started).Nanoseconds(), Trace: trace,
		Markers: markers,
	}
}

func evalCond(op Op, vars [16]int64, elapsed time.Duration) bool {
	v := int64(0)
	if op.VarID >= 0 && int(op.VarID) < len(vars) {
		v = vars[op.VarID]
	}
	switch op.Cond {
	case CondAlways:
		return true
	case CondLoopGE, CondVarGE:
		return v >= op.Imm
	case CondLoopLT, CondVarLT:
		return v < op.Imm
	case CondElapsedGE:
		return elapsed.Milliseconds() >= op.Imm
	default:
		return false
	}
}
