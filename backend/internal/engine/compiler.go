package engine

import (
	"fmt"
	"strings"

	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/macro"
)

type CompileError struct {
	NodeID  string `json:"node_id"`
	Message string `json:"message"`
}

func (e CompileError) Error() string {
	if e.NodeID == "" {
		return e.Message
	}
	return e.NodeID + ": " + e.Message
}

type compiler struct {
	m      macro.Macro
	succ   map[string]map[string]string
	ops    []Op
	loopV  int32
	nextV  int32
}

func Compile(m macro.Macro) (*Program, []CompileError) {
	c := &compiler{m: m, succ: map[string]map[string]string{}, nextV: 1}
	for _, e := range m.Edges {
		if c.succ[e.From] == nil {
			c.succ[e.From] = map[string]string{}
		}
		port := e.Port
		if port == "" {
			port = "out"
		}
		c.succ[e.From][port] = e.To
	}
	start := ""
	for _, n := range m.Nodes {
		if n.Type == "flow.start" {
			start = n.ID
			break
		}
	}
	if start == "" {
		return nil, []CompileError{{Message: "missing flow.start"}}
	}
	if err := c.emitNode(start, nil); err != nil {
		return nil, []CompileError{*err}
	}
	c.ops = append(c.ops, Op{Kind: OpHalt})
	p := &Program{
		Ops:       c.ops,
		Precision: string(m.Precision),
		MaxIters:  m.Budget.MaxIters,
		MaxWallMs: m.Budget.MaxWallMs,
	}
	if p.MaxIters <= 0 {
		p.MaxIters = 10000
	}
	if p.MaxWallMs <= 0 {
		p.MaxWallMs = 120000
	}
	if p.Precision == "" {
		p.Precision = string(macro.PrecisionBalanced)
	}
	return p, nil
}

type loopFrame struct {
	varID     int32
	count     int32
	bodyStart int
	endJumpAt int
}

func (c *compiler) emitNode(id string, lf *loopFrame) *CompileError {
	n, ok := c.m.Node(id)
	if !ok {
		return &CompileError{NodeID: id, Message: "unknown node"}
	}
	switch n.Type {
	case "flow.start", "comment":
		return c.emitNext(n.ID, "out", lf)
	case "flow.end":
		if lf != nil {
			// end of one iteration
			c.ops = append(c.ops, Op{Kind: OpVarInc, VarID: lf.varID, Imm: 1, NodeID: n.ID})
			c.ops = append(c.ops, Op{Kind: OpJumpIf, Cond: CondLoopLT, VarID: lf.varID, Imm: int64(lf.count), Jump: int32(lf.bodyStart - len(c.ops)), NodeID: n.ID})
			return nil
		}
		c.ops = append(c.ops, Op{Kind: OpHalt, NodeID: n.ID})
		return nil
	case "flow.break":
		c.ops = append(c.ops, Op{Kind: OpBreak, NodeID: n.ID})
		return c.emitNext(n.ID, "out", nil)
	case "flow.loop":
		return c.emitLoop(n, lf)
	case "flow.if":
		return c.emitIf(n, lf)
	case "key.down":
		u, err := hid.ParseUsage(macro.ParamString(n.Params, "key", ""))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		c.ops = append(c.ops, Op{Kind: OpKeyDown, Page: hid.PageKeyboard, Usage: u, Value: 1, NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "key.up":
		u, err := hid.ParseUsage(macro.ParamString(n.Params, "key", ""))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		c.ops = append(c.ops, Op{Kind: OpKeyUp, Page: hid.PageKeyboard, Usage: u, Value: 0, NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "key.tap":
		u, err := hid.ParseUsage(macro.ParamString(n.Params, "key", ""))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		hold := int64(macro.ParamInt(n.Params, "hold_us", 1000)) * 1000
		c.ops = append(c.ops,
			Op{Kind: OpKeyDown, Page: hid.PageKeyboard, Usage: u, Value: 1, NodeID: n.ID},
			Op{Kind: OpWait, DelayNs: hold, NodeID: n.ID},
			Op{Kind: OpKeyUp, Page: hid.PageKeyboard, Usage: u, Value: 0, NodeID: n.ID},
		)
		return c.emitNext(n.ID, "out", lf)
	case "key.combo":
		us, err := hid.ParseCombo(macro.ParamString(n.Params, "combo", ""))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		hold := int64(macro.ParamInt(n.Params, "hold_us", 2000)) * 1000
		for _, u := range us {
			c.ops = append(c.ops, Op{Kind: OpKeyDown, Page: hid.PageKeyboard, Usage: u, Value: 1, NodeID: n.ID})
		}
		c.ops = append(c.ops, Op{Kind: OpWait, DelayNs: hold, NodeID: n.ID})
		for i := len(us) - 1; i >= 0; i-- {
			c.ops = append(c.ops, Op{Kind: OpKeyUp, Page: hid.PageKeyboard, Usage: us[i], Value: 0, NodeID: n.ID})
		}
		return c.emitNext(n.ID, "out", lf)
	case "text.type":
		text := macro.ParamString(n.Params, "text", "")
		gap := int64(macro.ParamInt(n.Params, "gap_us", 8000)) * 1000
		for _, r := range text {
			u, shift, err := runeKey(r)
			if err != nil {
				return &CompileError{n.ID, err.Error()}
			}
			if shift {
				c.ops = append(c.ops, Op{Kind: OpKeyDown, Page: hid.PageKeyboard, Usage: hid.KeyLeftShift, Value: 1, NodeID: n.ID})
			}
			c.ops = append(c.ops,
				Op{Kind: OpKeyDown, Page: hid.PageKeyboard, Usage: u, Value: 1, NodeID: n.ID},
				Op{Kind: OpWait, DelayNs: 1000 * 1000, NodeID: n.ID},
				Op{Kind: OpKeyUp, Page: hid.PageKeyboard, Usage: u, Value: 0, NodeID: n.ID},
			)
			if shift {
				c.ops = append(c.ops, Op{Kind: OpKeyUp, Page: hid.PageKeyboard, Usage: hid.KeyLeftShift, Value: 0, NodeID: n.ID})
			}
			if gap > 0 {
				c.ops = append(c.ops, Op{Kind: OpWait, DelayNs: gap, NodeID: n.ID})
			}
		}
		return c.emitNext(n.ID, "out", lf)
	case "mouse.move.rel":
		dx := macro.ParamInt(n.Params, "dx", 0)
		dy := macro.ParamInt(n.Params, "dy", 0)
		if dx != 0 {
			c.ops = append(c.ops, Op{Kind: OpMouseRel, Page: hid.PageGenericDesktop, Usage: hid.GDX, Value: int32(dx), NodeID: n.ID})
		}
		if dy != 0 {
			c.ops = append(c.ops, Op{Kind: OpMouseRel, Page: hid.PageGenericDesktop, Usage: hid.GDY, Value: int32(dy), NodeID: n.ID})
		}
		return c.emitNext(n.ID, "out", lf)
	case "mouse.move.abs":
		c.ops = append(c.ops, Op{
			Kind: OpMouseAbs, Page: hid.PageGenericDesktop, Usage: hid.GDX,
			Value: int32(macro.ParamInt(n.Params, "x", 0)), Imm: int64(macro.ParamInt(n.Params, "y", 0)), NodeID: n.ID,
		})
		return c.emitNext(n.ID, "out", lf)
	case "mouse.click":
		btn, err := hid.ParseButton(macro.ParamString(n.Params, "button", "left"))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		c.ops = append(c.ops,
			Op{Kind: OpMouseBtn, Page: hid.PageButton, Usage: btn, Value: 1, NodeID: n.ID},
			Op{Kind: OpWait, DelayNs: 2_000_000, NodeID: n.ID},
			Op{Kind: OpMouseBtn, Page: hid.PageButton, Usage: btn, Value: 0, NodeID: n.ID},
		)
		return c.emitNext(n.ID, "out", lf)
	case "mouse.scroll":
		c.ops = append(c.ops, Op{Kind: OpMouseWheel, Page: hid.PageGenericDesktop, Usage: hid.GDWheel, Value: int32(macro.ParamInt(n.Params, "delta", 1)), NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "mouse.drag":
		btn, err := hid.ParseButton(macro.ParamString(n.Params, "button", "left"))
		if err != nil {
			return &CompileError{n.ID, err.Error()}
		}
		c.ops = append(c.ops,
			Op{Kind: OpMouseBtn, Page: hid.PageButton, Usage: btn, Value: 1, NodeID: n.ID},
			Op{Kind: OpMouseRel, Page: hid.PageGenericDesktop, Usage: hid.GDX, Value: int32(macro.ParamInt(n.Params, "dx", 0)), NodeID: n.ID},
			Op{Kind: OpMouseRel, Page: hid.PageGenericDesktop, Usage: hid.GDY, Value: int32(macro.ParamInt(n.Params, "dy", 0)), NodeID: n.ID},
			Op{Kind: OpMouseBtn, Page: hid.PageButton, Usage: btn, Value: 0, NodeID: n.ID},
		)
		return c.emitNext(n.ID, "out", lf)
	case "wait.fixed":
		us := int64(macro.ParamInt(n.Params, "us", 0))
		if us < 0 {
			return &CompileError{n.ID, "us must be >= 0"}
		}
		c.ops = append(c.ops, Op{Kind: OpWait, DelayNs: us * 1000, NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "wait.random":
		min := int64(macro.ParamInt(n.Params, "min_us", 0))
		max := int64(macro.ParamInt(n.Params, "max_us", 0))
		if max < min {
			return &CompileError{n.ID, "max_us < min_us"}
		}
		c.ops = append(c.ops, Op{Kind: OpWaitRand, DelayNs: min * 1000, Imm: max * 1000, NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "var.set":
		c.ops = append(c.ops, Op{Kind: OpVarSet, VarID: 8, Imm: int64(macro.ParamInt(n.Params, "value", 0)), NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "var.inc":
		c.ops = append(c.ops, Op{Kind: OpVarInc, VarID: 8, Imm: int64(macro.ParamInt(n.Params, "delta", 1)), NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	case "debug.marker":
		c.ops = append(c.ops, Op{Kind: OpMarker, Label: macro.ParamString(n.Params, "label", n.ID), NodeID: n.ID})
		return c.emitNext(n.ID, "out", lf)
	default:
		return &CompileError{n.ID, "unsupported node type " + n.Type}
	}
}

func (c *compiler) emitNext(from, port string, lf *loopFrame) *CompileError {
	to, ok := c.succ[from][port]
	if !ok {
		if lf != nil {
			// implicit end of iteration
			c.ops = append(c.ops, Op{Kind: OpVarInc, VarID: lf.varID, Imm: 1, NodeID: from})
			c.ops = append(c.ops, Op{Kind: OpJumpIf, Cond: CondLoopLT, VarID: lf.varID, Imm: int64(lf.count), Jump: int32(lf.bodyStart - len(c.ops)), NodeID: from})
		}
		return nil
	}
	return c.emitNode(to, lf)
}

func (c *compiler) emitLoop(n macro.Node, parent *loopFrame) *CompileError {
	count := macro.ParamInt(n.Params, "count", 1)
	if count < 1 {
		return &CompileError{n.ID, "loop count must be >= 1"}
	}
	vid := c.nextV
	c.nextV++
	c.ops = append(c.ops, Op{Kind: OpVarSet, VarID: vid, Imm: 0, NodeID: n.ID})
	body := len(c.ops)
	lf := &loopFrame{varID: vid, count: int32(count), bodyStart: body}
	bodyID, ok := c.succ[n.ID]["body"]
	if !ok {
		return &CompileError{n.ID, "loop missing body port"}
	}
	if err := c.emitNode(bodyID, lf); err != nil {
		return err
	}
	// if body didn't close the loop (no end / no implicit), close it
	if !c.loopClosed(lf) {
		c.ops = append(c.ops, Op{Kind: OpVarInc, VarID: vid, Imm: 1, NodeID: n.ID})
		c.ops = append(c.ops, Op{Kind: OpJumpIf, Cond: CondLoopLT, VarID: vid, Imm: int64(count), Jump: int32(body - len(c.ops)), NodeID: n.ID})
	}
	if out, ok := c.succ[n.ID]["out"]; ok {
		return c.emitNode(out, parent)
	}
	return nil
}

func (c *compiler) loopClosed(lf *loopFrame) bool {
	for _, op := range c.ops[lf.bodyStart:] {
		if op.Kind == OpJumpIf && op.VarID == lf.varID && op.Cond == CondLoopLT {
			return true
		}
	}
	return false
}

func (c *compiler) emitIf(n macro.Node, lf *loopFrame) *CompileError {
	cond := strings.ReplaceAll(macro.ParamString(n.Params, "cond", ""), " ", "")
	kind, imm, err := parseCond(cond)
	if err != nil {
		return &CompileError{n.ID, err.Error()}
	}
	varID := int32(0)
	if lf != nil {
		varID = lf.varID
	}
	// jump-if-true over the false branch
	j := len(c.ops)
	c.ops = append(c.ops, Op{Kind: OpJumpIf, Cond: kind, VarID: varID, Imm: imm, NodeID: n.ID})
	// false path first
	falseStart := len(c.ops)
	if fid, ok := c.succ[n.ID]["false"]; ok {
		if err := c.emitNode(fid, lf); err != nil {
			return err
		}
	}
	// jump over true path after false
	skip := len(c.ops)
	c.ops = append(c.ops, Op{Kind: OpJump, NodeID: n.ID})
	trueStart := len(c.ops)
	c.ops[j].Jump = int32(trueStart - j)
	if tid, ok := c.succ[n.ID]["true"]; ok {
		if err := c.emitNode(tid, lf); err != nil {
			return err
		}
	}
	c.ops[skip].Jump = int32(len(c.ops) - skip)
	_ = falseStart
	return nil
}

func parseCond(s string) (CondKind, int64, error) {
	switch {
	case strings.HasPrefix(s, "loop_i>="):
		return CondLoopGE, atoi(s[len("loop_i>="):]), nil
	case strings.HasPrefix(s, "loop_i<"):
		return CondLoopLT, atoi(s[len("loop_i<"):]), nil
	case strings.HasPrefix(s, "var>="):
		return CondVarGE, atoi(s[len("var>="):]), nil
	case strings.HasPrefix(s, "var<"):
		return CondVarLT, atoi(s[len("var<"):]), nil
	case strings.HasPrefix(s, "elapsed_ms>="):
		return CondElapsedGE, atoi(s[len("elapsed_ms>="):]), nil
	case s == "" || s == "true":
		return CondAlways, 0, nil
	default:
		return 0, 0, fmt.Errorf("unsupported cond %q", s)
	}
}

func atoi(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func runeKey(r rune) (uint16, bool, error) {
	if r >= 'a' && r <= 'z' {
		return hid.KeyA + uint16(r-'a'), false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return hid.KeyA + uint16(r-'A'), true, nil
	}
	if r >= '1' && r <= '9' {
		return hid.Key1 + uint16(r-'1'), false, nil
	}
	if r == '0' {
		return hid.Key0, false, nil
	}
	if r == ' ' {
		return hid.KeySpace, false, nil
	}
	return 0, false, fmt.Errorf("cannot type rune %q", r)
}
