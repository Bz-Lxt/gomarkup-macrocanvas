//go:build !linux

package device

import (
	"context"

	"github.com/macrocanvas/macrocanvas/internal/hid"
)

type UInput struct{}

func OpenUInput() (*UInput, error) { return nil, errUnsupported }

func (u *UInput) Name() string                              { return "uinput-stub" }
func (u *UInput) Node() string                              { return "" }
func (u *UInput) Inject(hid.Event) error                    { return errUnsupported }
func (u *UInput) Close() error                              { return nil }
func (u *UInput) Start(context.Context, chan<- Envelope) error { return errUnsupported }
func (u *UInput) Stop() error                               { return nil }
