package engine

type TraceEntry struct {
	Index     int    `json:"index"`
	NodeID    string `json:"node_id"`
	Kind      string `json:"kind"`
	PlanNs    int64  `json:"plan_ns"`
	ActualNs  int64  `json:"actual_ns"`
	ErrorNs   int64  `json:"error_ns"`
	Label     string `json:"label"`
}

type Trace struct {
	Entries []TraceEntry `json:"entries"`
	P50Ns   int64        `json:"p50_ns"`
	P99Ns   int64        `json:"p99_ns"`
	MaxNs   int64        `json:"max_ns"`
}

func summarizeTrace(t *Trace) {
	if len(t.Entries) == 0 {
		return
	}
	abs := make([]int64, len(t.Entries))
	for i, e := range t.Entries {
		v := e.ErrorNs
		if v < 0 {
			v = -v
		}
		abs[i] = v
		if v > t.MaxNs {
			t.MaxNs = v
		}
	}
	// insertion sort (N is small for UI traces)
	for i := 1; i < len(abs); i++ {
		j := i
		for j > 0 && abs[j] < abs[j-1] {
			abs[j], abs[j-1] = abs[j-1], abs[j]
			j--
		}
	}
	t.P50Ns = abs[len(abs)*50/100]
	t.P99Ns = abs[len(abs)*99/100]
}

func kindName(k OpKind) string {
	switch k {
	case OpKeyDown:
		return "key.down"
	case OpKeyUp:
		return "key.up"
	case OpMouseRel:
		return "mouse.rel"
	case OpMouseAbs:
		return "mouse.abs"
	case OpMouseBtn:
		return "mouse.btn"
	case OpMouseWheel:
		return "mouse.wheel"
	case OpWait, OpWaitRand:
		return "wait"
	case OpMarker:
		return "marker"
	case OpHalt:
		return "halt"
	default:
		return "op"
	}
}
