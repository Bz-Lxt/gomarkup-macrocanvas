//go:build linux

package device

import (
	"os"

	"github.com/macrocanvas/macrocanvas/internal/hid"
)

type GadgetSink struct {
	path string
	f    *os.File
	kbd  hid.KeyboardState
	mse  hid.MouseState
}

func OpenGadget(path string) (*GadgetSink, error) {
	if path == "" {
		path = "/dev/hidg0"
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return nil, errNoGadget
	}
	return &GadgetSink{path: path, f: f}, nil
}

func (g *GadgetSink) Name() string { return "hidg:" + g.path }

func (g *GadgetSink) Inject(ev hid.Event) error {
	switch ev.Kind {
	case hid.KindKey:
		if ev.Value != 0 {
			if err := g.kbd.Press(ev.Usage); err != nil {
				return err
			}
		} else {
			g.kbd.Release(ev.Usage)
		}
		r := g.kbd.Encode()
		pkt := append([]byte{0x01}, r[:]...)
		_, err := g.f.Write(pkt)
		return err
	case hid.KindButton:
		if err := g.mse.SetButton(ev.Usage, ev.Value != 0); err != nil {
			return err
		}
		r := g.mse.Encode()
		pkt := append([]byte{0x02}, r[:]...)
		_, err := g.f.Write(pkt)
		g.mse.X, g.mse.Y, g.mse.Wheel = 0, 0, 0
		return err
	case hid.KindPointer:
		switch ev.Usage {
		case hid.GDX:
			g.mse.X = clamp8(ev.Value)
		case hid.GDY:
			g.mse.Y = clamp8(ev.Value)
		case hid.GDWheel:
			g.mse.Wheel = clamp8(ev.Value)
		}
		r := g.mse.Encode()
		pkt := append([]byte{0x02}, r[:]...)
		_, err := g.f.Write(pkt)
		g.mse.X, g.mse.Y, g.mse.Wheel = 0, 0, 0
		return err
	}
	return nil
}

func (g *GadgetSink) Close() error { return g.f.Close() }

func clamp8(v int32) int8 {
	if v > 127 {
		return 127
	}
	if v < -127 {
		return -127
	}
	return int8(v)
}

func HasGadget() bool {
	_, err := os.Stat("/dev/hidg0")
	return err == nil
}
