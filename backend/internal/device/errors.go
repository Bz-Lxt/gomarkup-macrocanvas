package device

import "errors"

var (
	errClosed       = errors.New("device closed")
	errNoUinput     = errors.New("uinput unavailable")
	errNoGadget     = errors.New("hid gadget unavailable")
	errUnsupported  = errors.New("device backend not supported on this platform")
	errNotAvailable = errors.New("requested device tier not available")
)
