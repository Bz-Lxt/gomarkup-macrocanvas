package macro

type Precision string

const (
	PrecisionRealtime  Precision = "realtime"
	PrecisionBalanced  Precision = "balanced"
	PrecisionEfficient Precision = "efficient"
)

type Macro struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Deployed    bool      `json:"deployed"`
	Precision   Precision `json:"precision"`
	Trigger     Trigger   `json:"trigger"`
	Nodes       []Node    `json:"nodes"`
	Edges       []Edge    `json:"edges"`
	Budget      Budget    `json:"budget"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	Version     int       `json:"version"`
}

type Trigger struct {
	Kind   string `json:"kind"` // manual | hotkey | http
	Hotkey string `json:"hotkey"`
}

type Budget struct {
	MaxIters      int `json:"max_iters"`
	MaxWallMs     int `json:"max_wall_ms"`
	WatchdogMs    int `json:"watchdog_ms"`
}

type Node struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	X      float64        `json:"x"`
	Y      float64        `json:"y"`
	Params map[string]any `json:"params"`
}

type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Port string `json:"port"` // out | true | false | loop | body
}

func (m *Macro) Node(id string) (Node, bool) {
	for _, n := range m.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

func ParamString(p map[string]any, k, def string) string {
	if p == nil {
		return def
	}
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func ParamInt(p map[string]any, k string, def int) int {
	if p == nil {
		return def
	}
	v, ok := p[k]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return def
}

func ParamBool(p map[string]any, k string, def bool) bool {
	if p == nil {
		return def
	}
	if v, ok := p[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

var NodeTypes = []string{
	"flow.start", "flow.end", "flow.loop", "flow.if", "flow.break",
	"key.down", "key.up", "key.tap", "key.combo", "text.type",
	"mouse.move.rel", "mouse.move.abs", "mouse.click", "mouse.scroll", "mouse.drag",
	"wait.fixed", "wait.random",
	"var.set", "var.inc",
	"debug.marker", "comment",
}
