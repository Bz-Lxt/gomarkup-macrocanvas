//go:build !linux

package device

import "github.com/macrocanvas/macrocanvas/internal/hid"

type GadgetSink struct{}

func OpenGadget(string) (*GadgetSink, error) { return nil, errNoGadget }
func (g *GadgetSink) Name() string           { return "hidg-stub" }
func (g *GadgetSink) Inject(hid.Event) error { return errNoGadget }
func (g *GadgetSink) Close() error           { return nil }
func HasGadget() bool                        { return false }
