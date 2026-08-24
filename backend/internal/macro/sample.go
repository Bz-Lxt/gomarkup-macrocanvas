package macro

// P10Sample is the prompt example: press A → wait 15ms → mouse +50px → loop 3 → branch.
func P10Sample() Macro {
	return Macro{
		ID:          "sample-p10",
		Name:        "P10 样例：A → 15ms → 平移 50px ×3",
		Description: "按下 A 键 → 等待 15ms → 鼠标平移 50 像素 → 循环 3 次 → 条件分支",
		Enabled:     true,
		Deployed:    true,
		Precision:   PrecisionBalanced,
		Trigger:     Trigger{Kind: "manual"},
		Budget:      Budget{MaxIters: 1000, MaxWallMs: 5000, WatchdogMs: 2000},
		Nodes: []Node{
			{ID: "n-start", Type: "flow.start", X: 80, Y: 180, Params: map[string]any{}},
			{ID: "n-loop", Type: "flow.loop", X: 280, Y: 180, Params: map[string]any{"count": 3}},
			{ID: "n-a", Type: "key.tap", X: 500, Y: 80, Params: map[string]any{"key": "A", "hold_us": 2000}},
			{ID: "n-wait", Type: "wait.fixed", X: 720, Y: 80, Params: map[string]any{"us": 15000}},
			{ID: "n-move", Type: "mouse.move.rel", X: 940, Y: 80, Params: map[string]any{"dx": 50, "dy": 0}},
			{ID: "n-if", Type: "flow.if", X: 500, Y: 320, Params: map[string]any{"cond": "loop_i>=2"}},
			{ID: "n-mark-t", Type: "debug.marker", X: 720, Y: 260, Params: map[string]any{"label": "branch-true"}},
			{ID: "n-mark-f", Type: "debug.marker", X: 720, Y: 400, Params: map[string]any{"label": "branch-false"}},
			{ID: "n-end", Type: "flow.end", X: 1160, Y: 180, Params: map[string]any{}},
		},
		Edges: []Edge{
			{ID: "e1", From: "n-start", To: "n-loop", Port: "out"},
			{ID: "e2", From: "n-loop", To: "n-a", Port: "body"},
			{ID: "e3", From: "n-a", To: "n-wait", Port: "out"},
			{ID: "e4", From: "n-wait", To: "n-move", Port: "out"},
			{ID: "e5", From: "n-move", To: "n-if", Port: "out"},
			{ID: "e6", From: "n-if", To: "n-mark-t", Port: "true"},
			{ID: "e7", From: "n-if", To: "n-mark-f", Port: "false"},
			{ID: "e8", From: "n-mark-t", To: "n-end", Port: "out"},
			{ID: "e9", From: "n-mark-f", To: "n-end", Port: "out"},
		},
	}
}
