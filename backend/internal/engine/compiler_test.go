package engine

import (
	"testing"

	"github.com/macrocanvas/macrocanvas/internal/macro"
)

func TestCompileP10(t *testing.T) {
	m := macro.P10Sample()
	if issues := ValidateGraph(m); len(issues) > 0 {
		t.Fatalf("validate: %v", issues)
	}
	p, errs := Compile(m)
	if len(errs) > 0 {
		t.Fatalf("compile: %v", errs)
	}
	if len(p.Ops) < 8 {
		t.Fatalf("too few ops: %d", len(p.Ops))
	}
	var hasWait, hasRel, hasJump, hasMark bool
	for _, op := range p.Ops {
		switch op.Kind {
		case OpWait:
			if op.DelayNs == 15_000_000 {
				hasWait = true
			}
		case OpMouseRel:
			if op.Value == 50 {
				hasRel = true
			}
		case OpJumpIf:
			hasJump = true
		case OpMarker:
			hasMark = true
		}
	}
	if !hasWait || !hasRel || !hasJump || !hasMark {
		t.Fatalf("missing pieces wait=%v rel=%v jump=%v mark=%v ops=%+v", hasWait, hasRel, hasJump, hasMark, p.Ops)
	}
	if u := CheckUnpairedKeys(p); len(u) > 0 {
		t.Fatalf("unpaired: %v", u)
	}
}

func TestUnreachable(t *testing.T) {
	m := macro.P10Sample()
	m.Nodes = append(m.Nodes, macro.Node{ID: "orphan", Type: "debug.marker", Params: map[string]any{"label": "x"}})
	issues := ValidateGraph(m)
	found := false
	for _, i := range issues {
		if i.Code == "UNREACHABLE" {
			found = true
		}
	}
	if !found {
		t.Fatal(issues)
	}
}
