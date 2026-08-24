package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/engine"
	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/macro"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

type gatedInjector struct {
	mu      sync.Mutex
	events  []hid.Event
	down    chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *gatedInjector) Inject(ev hid.Event) error {
	g.mu.Lock()
	g.events = append(g.events, ev)
	g.mu.Unlock()
	if ev.Kind == hid.KindKey && ev.Page == hid.PageKeyboard && ev.Usage == hid.KeyA && ev.Value == 1 {
		g.once.Do(func() { close(g.down) })
		<-g.release
	}
	return nil
}

func (g *gatedInjector) Events() []hid.Event {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]hid.Event(nil), g.events...)
}

func TestCancelledRunReleasesPressedKey(t *testing.T) {
	m := macro.Macro{
		ID:        "cancel-key-tap",
		Precision: macro.PrecisionEfficient,
		Budget:    macro.Budget{MaxIters: 100, MaxWallMs: 1000},
		Nodes: []macro.Node{
			{ID: "start", Type: "flow.start"},
			{ID: "tap", Type: "key.tap", Params: map[string]any{"key": "A", "hold_us": 1}},
			{ID: "end", Type: "flow.end"},
		},
		Edges: []macro.Edge{
			{From: "start", To: "tap", Port: "out"},
			{From: "tap", To: "end", Port: "out"},
		},
	}
	program, errs := engine.Compile(m)
	if len(errs) != 0 {
		t.Fatalf("compile key tap: %v", errs)
	}

	sink := &gatedInjector{down: make(chan struct{}), release: make(chan struct{})}
	executor := engine.NewExecutor(
		sink,
		timing.NewPacer(timing.Calibration{Margin: time.Millisecond}),
		engine.NewSafety(100, 1000, 1000),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan engine.Result, 1)
	go func() {
		resultCh <- executor.Run(ctx, "run-cancel-key", m.ID, program)
	}()

	select {
	case <-sink.down:
	case <-time.After(time.Second):
		cancel()
		close(sink.release)
		t.Fatal("executor did not inject the key-down event")
	}
	cancel()
	close(sink.release)

	var result engine.Result
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("executor did not return after cancellation")
	}
	if result.Status != "cancelled" || result.Reason != "context" {
		t.Fatalf("result = %s/%s, want cancelled/context", result.Status, result.Reason)
	}

	var keyValues []int32
	for _, ev := range sink.Events() {
		if ev.Kind == hid.KindKey && ev.Page == hid.PageKeyboard && ev.Usage == hid.KeyA {
			keyValues = append(keyValues, ev.Value)
		}
	}
	if len(keyValues) != 2 || keyValues[0] != 1 || keyValues[1] != 0 {
		t.Fatalf("A key event values = %v, want [1 0]", keyValues)
	}
}
