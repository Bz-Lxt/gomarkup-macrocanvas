package capture

import (
	"sync"
	"sync/atomic"

	"github.com/macrocanvas/macrocanvas/internal/device"
)

type Ring struct {
	buf    []device.Envelope
	head   uint64
	mask   uint64
	dropped atomic.Uint64
	mu     sync.Mutex
}

func NewRing(n int) *Ring {
	p := 1
	for p < n {
		p <<= 1
	}
	return &Ring{buf: make([]device.Envelope, p), mask: uint64(p - 1)}
}

func (r *Ring) Push(e device.Envelope) {
	r.mu.Lock()
	r.buf[r.head&r.mask] = e
	r.head++
	r.mu.Unlock()
}

func (r *Ring) Drop() { r.dropped.Add(1) }

func (r *Ring) Dropped() uint64 { return r.dropped.Load() }

func (r *Ring) Tail(n int) []device.Envelope {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 {
		n = 256
	}
	h := r.head
	if h == 0 {
		return []device.Envelope{}
	}
	capn := uint64(len(r.buf))
	avail := h
	if avail > capn {
		avail = capn
	}
	if uint64(n) > avail {
		n = int(avail)
	}
	out := make([]device.Envelope, n)
	start := h - uint64(n)
	for i := 0; i < n; i++ {
		out[i] = r.buf[(start+uint64(i))&r.mask]
	}
	return out
}

func (r *Ring) Clear() {
	r.mu.Lock()
	r.head = 0
	r.mu.Unlock()
	r.dropped.Store(0)
}
