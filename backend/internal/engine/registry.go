package engine

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/clock"
)

type RunRecord struct {
	Result
	MacroName string `json:"macro_name"`
	QueuedAt  string `json:"queued_at"`
	Recovered bool   `json:"recovered"`
}

type Registry struct {
	mu      sync.Mutex
	active  map[string]context.CancelFunc
	history []RunRecord
	byID    map[string]*RunRecord
}

func NewRegistry() *Registry {
	return &Registry{active: map[string]context.CancelFunc{}, byID: map[string]*RunRecord{}}
}

func (r *Registry) Begin(id string, cancel context.CancelFunc, rec RunRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[id] = cancel
	rec.Status = "running"
	rec.QueuedAt = clock.Format(clock.Now())
	cp := rec
	r.history = append(r.history, cp)
	r.byID[id] = &r.history[len(r.history)-1]
}

func (r *Registry) Finish(res Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, res.RunID)
	if rec, ok := r.byID[res.RunID]; ok {
		rec.Result = res
		rec.EndedAt = clock.Format(clock.Now())
	}
}

func (r *Registry) Cancel(id string) bool {
	r.mu.Lock()
	c, ok := r.active[id]
	r.mu.Unlock()
	if ok && c != nil {
		c()
	}
	return ok
}

func (r *Registry) CancelAll() {
	r.mu.Lock()
	cs := make([]context.CancelFunc, 0, len(r.active))
	for _, c := range r.active {
		cs = append(cs, c)
	}
	r.mu.Unlock()
	for _, c := range cs {
		if c != nil {
			c()
		}
	}
}

func (r *Registry) Get(id string) (RunRecord, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok {
		return RunRecord{}, false
	}
	return *rec, true
}

func (r *Registry) List() []RunRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RunRecord, len(r.history))
	copy(out, r.history)
	return out
}

func (r *Registry) Incomplete() []RunRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []RunRecord
	for _, h := range r.history {
		if h.Status == "running" || h.Status == "queued" {
			out = append(out, h)
		}
	}
	return out
}

func NewRunID() string {
	return clock.Now().Format("20060102150405") + "-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
}
