package engine

import (
	"fmt"
	"strings"
)

func Disassemble(p *Program) []string {
	if p == nil {
		return []string{}
	}
	out := make([]string, 0, len(p.Ops))
	for i, op := range p.Ops {
		out = append(out, fmt.Sprintf("%04d  %s", i, formatOp(op)))
	}
	return out
}

func formatOp(op Op) string {
	var b strings.Builder
	b.WriteString(kindName(op.Kind))
	if op.Usage != 0 {
		fmt.Fprintf(&b, " usage=0x%02X", op.Usage)
	}
	if op.Value != 0 {
		fmt.Fprintf(&b, " val=%d", op.Value)
	}
	if op.DelayNs != 0 {
		fmt.Fprintf(&b, " delay=%dns", op.DelayNs)
	}
	if op.Jump != 0 {
		fmt.Fprintf(&b, " jmp=%+d", op.Jump)
	}
	if op.Label != "" {
		fmt.Fprintf(&b, " '%s'", op.Label)
	}
	if op.NodeID != "" {
		fmt.Fprintf(&b, " @%s", op.NodeID)
	}
	return b.String()
}
