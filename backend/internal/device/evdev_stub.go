//go:build !linux

package device

import "context"

type EvdevSource struct{}

func OpenEvdev(bool) (*EvdevSource, error)              { return nil, errUnsupported }
func (s *EvdevSource) Name() string                     { return "evdev-stub" }
func (s *EvdevSource) Start(context.Context, chan<- Envelope) error {
	return errUnsupported
}
func (s *EvdevSource) Stop() error { return nil }

func ListEventNodes() []string { return []string{} }
func HasUinput() bool          { return false }
