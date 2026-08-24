package capture

import (
	"sync"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/device"
	"github.com/macrocanvas/macrocanvas/internal/hid"
)

type Filter struct {
	Kinds   map[string]bool
	Sources map[hid.Source]bool
	Paused  bool
}

type Client struct {
	ch     chan []device.Envelope
	filter Filter
}

type Hub struct {
	ring    *Ring
	mu      sync.Mutex
	clients map[*Client]struct{}
	mask    bool
	auth    bool
	batch   []device.Envelope
}

func NewHub(ring *Ring, mask bool) *Hub {
	return &Hub{ring: ring, clients: map[*Client]struct{}{}, mask: mask}
}

func (h *Hub) SetMask(v bool) { h.mu.Lock(); h.mask = v; h.mu.Unlock() }
func (h *Hub) SetAuth(v bool) { h.mu.Lock(); h.auth = v; h.mu.Unlock() }
func (h *Hub) Authorized() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.auth
}

func (h *Hub) Ingest(e device.Envelope) {
	h.mu.Lock()
	if h.mask && e.Kind == "key" && isPrintable(e.Usage) {
		e.Name = "•"
		e.Masked = true
	}
	if !h.auth {
		h.mu.Unlock()
		return
	}
	h.batch = append(h.batch, e)
	h.mu.Unlock()
	h.ring.Push(e)
}

func (h *Hub) FlushLoop(stop <-chan struct{}) {
	t := time.NewTicker(16 * time.Millisecond) // ≤60 fps
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			h.flush()
		}
	}
}

func (h *Hub) flush() {
	h.mu.Lock()
	if len(h.batch) == 0 {
		h.mu.Unlock()
		return
	}
	batch := h.batch
	h.batch = nil
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		if c.filter.Paused {
			continue
		}
		select {
		case c.ch <- batch:
		default:
			h.ring.Drop()
		}
	}
}

func (h *Hub) Subscribe() *Client {
	c := &Client{ch: make(chan []device.Envelope, 8), filter: Filter{}}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) Unsubscribe(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (c *Client) C() <-chan []device.Envelope { return c.ch }

func isPrintable(u uint16) bool {
	return (u >= hid.KeyA && u <= hid.Key0) || u == hid.KeySpace
}

func (h *Hub) Mask() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.mask
}
