package capture

import (
	"testing"

	"github.com/macrocanvas/macrocanvas/internal/device"
)

func TestRingWrap(t *testing.T) {
	r := NewRing(8)
	for i := 0; i < 20; i++ {
		r.Push(device.Envelope{Seq: uint64(i + 1)})
	}
	tail := r.Tail(5)
	if len(tail) != 5 {
		t.Fatalf("len=%d", len(tail))
	}
	if tail[4].Seq != 20 {
		t.Fatalf("last=%d", tail[4].Seq)
	}
}

func TestRingEmpty(t *testing.T) {
	r := NewRing(4)
	if n := len(r.Tail(10)); n != 0 {
		t.Fatal(n)
	}
}
