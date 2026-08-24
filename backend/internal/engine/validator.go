package engine

import (
	"fmt"

	"github.com/macrocanvas/macrocanvas/internal/macro"
)

type Issue struct {
	NodeID  string `json:"node_id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e Issue) Error() string {
	if e.NodeID == "" {
		return e.Message
	}
	return e.NodeID + ": " + e.Message
}

func ValidateGraph(m macro.Macro) []Issue {
	var issues []Issue
	known := map[string]macro.Node{}
	types := map[string]bool{}
	for _, t := range macro.NodeTypes {
		types[t] = true
	}
	starts := 0
	for _, n := range m.Nodes {
		if _, ok := known[n.ID]; ok {
			issues = append(issues, Issue{n.ID, "DUP_ID", "duplicate node id"})
		}
		known[n.ID] = n
		if !types[n.Type] {
			issues = append(issues, Issue{n.ID, "BAD_TYPE", "unknown node type " + n.Type})
		}
		if n.Type == "flow.start" {
			starts++
		}
	}
	if starts != 1 {
		issues = append(issues, Issue{Code: "START", Message: "exactly one flow.start required"})
	}
	adj := map[string][]string{}
	for _, e := range m.Edges {
		if _, ok := known[e.From]; !ok {
			issues = append(issues, Issue{e.From, "BAD_EDGE", "edge.from missing"})
		}
		if _, ok := known[e.To]; !ok {
			issues = append(issues, Issue{e.To, "BAD_EDGE", "edge.to missing"})
		}
		port := e.Port
		if port == "" {
			port = "out"
		}
		from := known[e.From]
		if !portOK(from.Type, port) {
			issues = append(issues, Issue{e.From, "BAD_PORT", "port " + port + " illegal on " + from.Type})
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	// reachability
	start := ""
	for _, n := range m.Nodes {
		if n.Type == "flow.start" {
			start = n.ID
		}
	}
	seen := map[string]bool{}
	var dfs func(string)
	dfs = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, t := range adj[id] {
			dfs(t)
		}
	}
	if start != "" {
		dfs(start)
	}
	for _, n := range m.Nodes {
		if n.Type == "comment" {
			continue
		}
		if !seen[n.ID] {
			issues = append(issues, Issue{n.ID, "UNREACHABLE", "node not reachable from start"})
		}
	}
	// cycles that are not under a loop
	loopNodes := map[string]bool{}
	for _, n := range m.Nodes {
		if n.Type == "flow.loop" {
			for _, e := range m.Edges {
				if e.From == n.ID && (e.Port == "body" || e.Port == "") {
					markBody(e.To, adj, known, loopNodes)
				}
			}
		}
	}
	if cycle := findIllegalCycle(start, adj, known, loopNodes); cycle != "" {
		issues = append(issues, Issue{Code: "CYCLE", Message: "illegal cycle " + cycle})
	}
	return issues
}

func portOK(typ, port string) bool {
	switch typ {
	case "flow.if":
		return port == "true" || port == "false"
	case "flow.loop":
		return port == "body" || port == "out"
	default:
		return port == "out"
	}
}

func markBody(id string, adj map[string][]string, known map[string]macro.Node, out map[string]bool) {
	if out[id] {
		return
	}
	n := known[id]
	if n.Type == "flow.end" || n.Type == "flow.start" {
		return
	}
	out[id] = true
	for _, t := range adj[id] {
		markBody(t, adj, known, out)
	}
}

func findIllegalCycle(start string, adj map[string][]string, known map[string]macro.Node, loop map[string]bool) string {
	color := map[string]int{}
	var walk func(string) string
	walk = func(id string) string {
		color[id] = 1
		for _, t := range adj[id] {
			if loop[id] && loop[t] {
				continue
			}
			if known[id].Type == "flow.loop" {
				continue
			}
			if color[t] == 1 {
				return id + "->" + t
			}
			if color[t] == 0 {
				if s := walk(t); s != "" {
					return s
				}
			}
		}
		color[id] = 2
		return ""
	}
	if start == "" {
		return ""
	}
	return walk(start)
}

func CheckUnpairedKeys(p *Program) []Issue {
	down := map[uint16]int{}
	var issues []Issue
	for _, op := range p.Ops {
		switch op.Kind {
		case OpKeyDown, OpMouseBtn:
			if op.Value != 0 {
				down[op.Usage]++
			} else {
				down[op.Usage]--
			}
		case OpKeyUp:
			down[op.Usage]--
		}
	}
	for u, n := range down {
		if n > 0 {
			issues = append(issues, Issue{Code: "UNPAIRED_KEY", Message: fmt.Sprintf("usage 0x%02X left down (%d)", u, n)})
		}
	}
	return issues
}
