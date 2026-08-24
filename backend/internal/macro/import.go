package macro

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ImportError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ImportError) Error() string { return e.Field + ": " + e.Message }

// DecodeAndValidate unmarshals a macro and checks structural integrity
// (field presence, types, bounds) — not just "is it JSON".
func DecodeAndValidate(raw []byte) (Macro, []ImportError) {
	var m Macro
	if err := json.Unmarshal(raw, &m); err != nil {
		return Macro{}, []ImportError{{Field: "$", Message: "invalid json: " + err.Error()}}
	}
	var errs []ImportError
	if strings.TrimSpace(m.Name) == "" {
		errs = append(errs, ImportError{"name", "required"})
	}
	if len(m.Name) > 120 {
		errs = append(errs, ImportError{"name", "max 120 chars"})
	}
	if m.Precision != "" && m.Precision != PrecisionRealtime && m.Precision != PrecisionBalanced && m.Precision != PrecisionEfficient {
		errs = append(errs, ImportError{"precision", "must be realtime|balanced|efficient"})
	}
	if m.Budget.MaxIters < 0 || m.Budget.MaxIters > 1_000_000 {
		errs = append(errs, ImportError{"budget.max_iters", "out of range"})
	}
	if m.Budget.MaxWallMs < 0 || m.Budget.MaxWallMs > 3_600_000 {
		errs = append(errs, ImportError{"budget.max_wall_ms", "out of range"})
	}
	if m.Nodes == nil {
		errs = append(errs, ImportError{"nodes", "required array"})
	}
	if m.Edges == nil {
		errs = append(errs, ImportError{"edges", "required array"})
	}
	seen := map[string]bool{}
	allowed := map[string]bool{}
	for _, t := range NodeTypes {
		allowed[t] = true
	}
	for i, n := range m.Nodes {
		p := fmt.Sprintf("nodes[%d]", i)
		if n.ID == "" {
			errs = append(errs, ImportError{p + ".id", "required"})
		}
		if seen[n.ID] {
			errs = append(errs, ImportError{p + ".id", "duplicate"})
		}
		seen[n.ID] = true
		if !allowed[n.Type] {
			errs = append(errs, ImportError{p + ".type", "unknown " + n.Type})
		}
		if n.Params == nil {
			m.Nodes[i].Params = map[string]any{}
		}
	}
	for i, e := range m.Edges {
		p := fmt.Sprintf("edges[%d]", i)
		if e.From == "" || e.To == "" {
			errs = append(errs, ImportError{p, "from/to required"})
		}
	}
	if m.Trigger.Kind == "" {
		m.Trigger.Kind = "manual"
	}
	switch m.Trigger.Kind {
	case "manual", "hotkey", "http":
	default:
		errs = append(errs, ImportError{"trigger.kind", "must be manual|hotkey|http"})
	}
	return m, errs
}
