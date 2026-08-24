package device

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/macrocanvas/macrocanvas/internal/clock"
	"github.com/macrocanvas/macrocanvas/internal/hid"
)

// Loopback is the T-C userspace bus. Injected events are immediately
// readable as SourceSimulated (never as physical).
type Loopback struct {
	mu     sync.Mutex
	subs   []chan hid.Event
	seq    atomic.Uint64
	closed atomic.Bool
}

func NewLoopback() *Loopback { return &Loopback{} }

func (l *Loopback) Name() string { return "userspace-loopback" }

func (l *Loopback) Inject(ev hid.Event) error {
	if l.closed.Load() {
		return errClosed
	}
	ev.Source = hid.SourceInjected
	l.mu.Lock()
	subs := append([]chan hid.Event(nil), l.subs...)
	l.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
	return nil
}

func (l *Loopback) Close() error {
	l.closed.Store(true)
	return nil
}

func (l *Loopback) Start(ctx context.Context, out chan<- Envelope) error {
	ch := make(chan hid.Event, 256)
	l.mu.Lock()
	l.subs = append(l.subs, ch)
	l.mu.Unlock()
	go func() {
		defer l.unsub(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				out <- wrapEvent(ev, hid.SourceSimulated, l.Name(), &l.seq)
			}
		}
	}()
	return nil
}

func (l *Loopback) Stop() error { return nil }

func (l *Loopback) unsub(ch chan hid.Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, s := range l.subs {
		if s == ch {
			l.subs = append(l.subs[:i], l.subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func wrapEvent(ev hid.Event, src hid.Source, dev string, seq *atomic.Uint64) Envelope {
	if ev.Source != "" {
		src = ev.Source
	}
	name := hid.UsageName(ev.Usage)
	if ev.Kind == hid.KindButton {
		name = hid.ButtonName(ev.Usage)
	}
	if ev.Kind == hid.KindPointer {
		if ev.Usage == hid.GDX {
			name = "REL_X"
		} else if ev.Usage == hid.GDY {
			name = "REL_Y"
		} else {
			name = "WHEEL"
		}
	}
	kind := "key"
	switch ev.Kind {
	case hid.KindButton:
		kind = "button"
	case hid.KindPointer:
		kind = "pointer"
	case hid.KindSync:
		kind = "sync"
	}
	now := clock.Now()
	return Envelope{
		Seq:         seq.Add(1),
		IngressMono: now.UnixNano(),
		BeijingTime: clock.FormatMicro(now),
		Page:        ev.Page,
		Usage:       ev.Usage,
		Value:       ev.Value,
		Kind:        kind,
		Name:        name,
		Source:      src,
		Device:      dev,
	}
}
