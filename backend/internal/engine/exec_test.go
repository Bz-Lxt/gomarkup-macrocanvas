package engine

import (
	"context"
	"testing"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/macro"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

type memSink struct {
	evs []hid.Event
}

func (m *memSink) Inject(ev hid.Event) error {
	m.evs = append(m.evs, ev)
	return nil
}

func TestExecuteP10(t *testing.T) {
	m := macro.P10Sample()
	p, errs := Compile(m)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	sink := &memSink{}
	pac := timing.NewPacer(timing.Calibration{Margin: time.Millisecond})
	saf := NewSafety(5000, 5000, 5000)
	ex := NewExecutor(sink, pac, saf)
	res := ex.Run(context.Background(), "t1", m.ID, p)
	if res.Status != "succeeded" {
		t.Fatalf("status=%s reason=%s", res.Status, res.Reason)
	}
	var taps, moves int
	for _, e := range sink.evs {
		if e.Kind == hid.KindKey && e.Usage == hid.KeyA && e.Value == 1 {
			taps++
		}
		if e.Kind == hid.KindPointer && e.Usage == hid.GDX && e.Value == 50 {
			moves++
		}
	}
	if taps != 3 || moves != 3 {
		t.Fatalf("taps=%d moves=%d evs=%d markers=%v", taps, moves, len(sink.evs), res.Markers)
	}
	if len(res.Markers) == 0 {
		t.Fatal("expected branch markers")
	}
}

func TestEmergencyStop(t *testing.T) {
	m := macro.Macro{
		ID: "x", Precision: macro.PrecisionEfficient,
		Budget: macro.Budget{MaxIters: 100000, MaxWallMs: 5000},
		Nodes: []macro.Node{
			{ID: "s", Type: "flow.start", Params: map[string]any{}},
			{ID: "l", Type: "flow.loop", Params: map[string]any{"count": 10000}},
			{ID: "w", Type: "wait.fixed", Params: map[string]any{"us": 2000}},
			{ID: "e", Type: "flow.end", Params: map[string]any{}},
		},
		Edges: []macro.Edge{
			{From: "s", To: "l", Port: "out"},
			{From: "l", To: "w", Port: "body"},
			{From: "w", To: "e", Port: "out"},
		},
	}
	p, errs := Compile(m)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	saf := NewSafety(100000, 5000, 5000)
	ex := NewExecutor(&memSink{}, timing.NewPacer(timing.Calibration{Margin: time.Millisecond}), saf)
	go func() {
		time.Sleep(5 * time.Millisecond)
		saf.EmergencyStop()
	}()
	res := ex.Run(context.Background(), "t2", "x", p)
	if res.Status != "stopped" {
		t.Fatalf("%s %s", res.Status, res.Reason)
	}
}

func TestImportRejectsGarbage(t *testing.T) {
	_, errs := macro.DecodeAndValidate([]byte(`{"name":""}`))
	if len(errs) == 0 {
		t.Fatal("expected import errors")
	}
}
