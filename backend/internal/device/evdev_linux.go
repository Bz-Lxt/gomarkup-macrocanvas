//go:build linux

package device

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/macrocanvas/macrocanvas/internal/clock"
	"github.com/macrocanvas/macrocanvas/internal/hid"
)

// EvdevSource reads host /dev/input/event* (T-A). Optional EVIOCGRAB.
type EvdevSource struct {
	paths  []string
	grab   bool
	fds    []int
	seq    atomic.Uint64
	cancel context.CancelFunc
}

func OpenEvdev(grab bool) (*EvdevSource, error) {
	nodes, _ := filepath.Glob("/dev/input/event*")
	if len(nodes) == 0 {
		return nil, errNotAvailable
	}
	return &EvdevSource{paths: nodes, grab: grab}, nil
}

func (s *EvdevSource) Name() string { return "evdev-host" }

func (s *EvdevSource) Start(ctx context.Context, out chan<- Envelope) error {
	ctx, s.cancel = context.WithCancel(ctx)
	const eviocgrab = 0x40044590
	for _, p := range s.paths {
		fd, err := syscall.Open(p, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		if s.grab {
			_ = ioctl(uintptr(fd), eviocgrab, 1)
		}
		s.fds = append(s.fds, fd)
		go s.loop(ctx, fd, p, out)
	}
	if len(s.fds) == 0 {
		return errNotAvailable
	}
	return nil
}

func (s *EvdevSource) loop(ctx context.Context, fd int, node string, out chan<- Envelope) {
	buf := make([]byte, 24*32)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := syscall.Read(fd, buf)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				time.Sleep(500 * time.Microsecond)
				continue
			}
			return
		}
		ingress := clock.Now()
		for off := 0; off+24 <= n; off += 24 {
			e := *(*inputEvent)(unsafe.Pointer(&buf[off]))
			if e.Type == evSyn {
				continue
			}
			env, ok := fromEvdev(e, ingress, &s.seq, node)
			if !ok {
				continue
			}
			env.Source = hid.SourcePhysical
			select {
			case out <- env:
			default:
			}
		}
	}
}

func (s *EvdevSource) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	for _, fd := range s.fds {
		_ = syscall.Close(fd)
	}
	s.fds = nil
	return nil
}

func ListEventNodes() []string {
	n, _ := filepath.Glob("/dev/input/event*")
	if n == nil {
		return []string{}
	}
	return n
}

func HasUinput() bool {
	_, err := os.Stat("/dev/uinput")
	return err == nil
}
