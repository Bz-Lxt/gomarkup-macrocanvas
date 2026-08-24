//go:build linux

package device

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/macrocanvas/macrocanvas/internal/clock"
	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/logger"
)

const (
	uiDevCreate  = 0x5501
	uiDevDestroy = 0x5502
	uiDevSetup   = 0x405C5503
	uiSetEvBit   = 0x40045564
	uiSetKeyBit  = 0x40045565
	uiSetRelBit  = 0x40045566
	uiGetSysname = 0x8040552C
	evSyn        = 0x00
	evKey        = 0x01
	evRel        = 0x02
	relX         = 0x00
	relY         = 0x01
	relWheel     = 0x08
	synReport    = 0
	busUSB       = 0x03
)

type inputID struct {
	BusType, Vendor, Product, Version uint16
}

type uinputSetup struct {
	ID           inputID
	Name         [80]byte
	FFEffectsMax uint32
}

type inputEvent struct {
	Sec, Usec int64
	Type      uint16
	Code      uint16
	Value     int32
}

type UInput struct {
	fd       int
	eventFD  int
	node     string
	seq      atomic.Uint64
	closed   atomic.Bool
	cancel   context.CancelFunc
}

func OpenUInput() (*UInput, error) {
	if _, err := os.Stat("/dev/uinput"); err != nil {
		return nil, errNoUinput
	}
	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open uinput: %w", err)
	}
	ufd := uintptr(fd)
	for _, b := range []struct {
		req, val uintptr
	}{
		{uiSetEvBit, evKey}, {uiSetEvBit, evRel},
		{uiSetRelBit, relX}, {uiSetRelBit, relY}, {uiSetRelBit, relWheel},
	} {
		if err := ioctl(ufd, b.req, b.val); err != nil {
			syscall.Close(fd)
			return nil, err
		}
	}
	for _, u := range hid.KnownKeyboardUsages() {
		code, err := hid.ToPlatform(u, hid.PlatEvdev)
		if err != nil {
			continue
		}
		if err := ioctl(ufd, uiSetKeyBit, uintptr(code)); err != nil {
			syscall.Close(fd)
			return nil, err
		}
	}
	for _, btn := range []uintptr{0x110, 0x111, 0x112, 0x113, 0x114} { // BTN_LEFT..
		if err := ioctl(ufd, uiSetKeyBit, btn); err != nil {
			syscall.Close(fd)
			return nil, err
		}
	}
	setup := uinputSetup{ID: inputID{BusType: busUSB, Vendor: 0x1d6b, Product: 0x0104, Version: 1}}
	copy(setup.Name[:], "MacroCanvas HID Composite")
	if err := ioctl(ufd, uiDevSetup, uintptr(unsafe.Pointer(&setup))); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("UI_DEV_SETUP: %w", err)
	}
	if err := ioctl(ufd, uiDevCreate, 0); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("UI_DEV_CREATE: %w", err)
	}
	node, err := discoverNode(ufd)
	if err != nil {
		ioctl(ufd, uiDevDestroy, 0)
		syscall.Close(fd)
		return nil, err
	}
	rfd, err := syscall.Open(node, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		ioctl(ufd, uiDevDestroy, 0)
		syscall.Close(fd)
		return nil, fmt.Errorf("open %s: %w", node, err)
	}
	logger.Log().Info("uinput device created", "node", node)
	return &UInput{fd: fd, eventFD: rfd, node: node}, nil
}

func (u *UInput) Name() string { return "uinput:" + u.node }

func (u *UInput) Node() string { return u.node }

func (u *UInput) Inject(ev hid.Event) error {
	if u.closed.Load() {
		return errClosed
	}
	typ, code, err := toEvdev(ev)
	if err != nil {
		return err
	}
	if err := writeEvent(u.fd, typ, code, ev.Value); err != nil {
		return err
	}
	return writeEvent(u.fd, evSyn, synReport, 0)
}

func (u *UInput) Close() error {
	if u.closed.Swap(true) {
		return nil
	}
	if u.cancel != nil {
		u.cancel()
	}
	ioctl(uintptr(u.fd), uiDevDestroy, 0)
	_ = syscall.Close(u.eventFD)
	return syscall.Close(u.fd)
}

func (u *UInput) Start(ctx context.Context, out chan<- Envelope) error {
	ctx, u.cancel = context.WithCancel(ctx)
	go u.readLoop(ctx, out)
	return nil
}

func (u *UInput) Stop() error {
	if u.cancel != nil {
		u.cancel()
	}
	return nil
}

func (u *UInput) readLoop(ctx context.Context, out chan<- Envelope) {
	buf := make([]byte, 24*32)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := syscall.Read(u.eventFD, buf)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				time.Sleep(200 * time.Microsecond)
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
			env, ok := fromEvdev(e, ingress, &u.seq, u.node)
			if ok {
				select {
				case out <- env:
				default:
				}
			}
		}
	}
}

func fromEvdev(e inputEvent, ingress time.Time, seq *atomic.Uint64, node string) (Envelope, bool) {
	kts := time.Unix(e.Sec, e.Usec*1000)
	skew := ingress.UnixNano() - kts.UnixNano()
	env := Envelope{
		Seq:          seq.Add(1),
		IngressMono:  ingress.UnixNano(),
		KernelStamp:  kts.UnixNano(),
		KernelSkewNs: skew,
		BeijingTime:  clock.FormatMicro(ingress),
		Value:        e.Value,
		Source:       hid.SourceKernelVirtual,
		Device:       node,
	}
	switch e.Type {
	case evKey:
		if e.Code >= 0x110 && e.Code <= 0x114 {
			env.Kind = "button"
			env.Page = hid.PageButton
			env.Usage = uint16(e.Code - 0x110 + 1)
			env.Name = hid.ButtonName(env.Usage)
			return env, true
		}
		u, err := hid.FromPlatform(e.Code, hid.PlatEvdev)
		if err != nil {
			return Envelope{}, false
		}
		env.Kind = "key"
		env.Page = hid.PageKeyboard
		env.Usage = u
		env.Name = hid.UsageName(u)
		return env, true
	case evRel:
		env.Kind = "pointer"
		env.Page = hid.PageGenericDesktop
		switch e.Code {
		case relX:
			env.Usage = hid.GDX
			env.Name = "REL_X"
		case relY:
			env.Usage = hid.GDY
			env.Name = "REL_Y"
		case relWheel:
			env.Usage = hid.GDWheel
			env.Name = "WHEEL"
		default:
			return Envelope{}, false
		}
		return env, true
	}
	return Envelope{}, false
}

func toEvdev(ev hid.Event) (typ, code uint16, err error) {
	switch ev.Kind {
	case hid.KindKey:
		c, e := hid.ToPlatform(ev.Usage, hid.PlatEvdev)
		return evKey, c, e
	case hid.KindButton:
		return evKey, 0x110 + ev.Usage - 1, nil
	case hid.KindPointer:
		switch ev.Usage {
		case hid.GDX:
			return evRel, relX, nil
		case hid.GDY:
			return evRel, relY, nil
		case hid.GDWheel:
			return evRel, relWheel, nil
		}
	}
	return 0, 0, fmt.Errorf("unmappable event kind=%d usage=%d", ev.Kind, ev.Usage)
}

func writeEvent(fd int, typ, code uint16, val int32) error {
	e := inputEvent{Type: typ, Code: code, Value: val}
	b := (*[24]byte)(unsafe.Pointer(&e))[:]
	_, err := syscall.Write(fd, b)
	return err
}

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

func discoverNode(ufd uintptr) (string, error) {
	before := map[string]bool{}
	matches, _ := filepath.Glob("/dev/input/event*")
	for _, p := range matches {
		before[p] = true
	}
	for i := 0; i < 20; i++ {
		cur, _ := filepath.Glob("/dev/input/event*")
		for _, p := range cur {
			if !before[p] {
				return p, nil
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	var raw [64]byte
	if err := ioctl(ufd, uiGetSysname, uintptr(unsafe.Pointer(&raw))); err != nil {
		return "", fmt.Errorf("UI_GET_SYSNAME: %w", err)
	}
	n := 0
	for n < len(raw) && raw[n] != 0 {
		n++
	}
	sysname := string(raw[:n])
	globs, _ := filepath.Glob("/sys/devices/virtual/input/" + sysname + "/event*/dev")
	if len(globs) == 0 {
		globs, _ = filepath.Glob("/sys/class/input/" + sysname + "/event*/dev")
	}
	if len(globs) == 0 {
		return "", fmt.Errorf("sysfs event node missing for %s", sysname)
	}
	devStr, err := os.ReadFile(globs[0])
	if err != nil {
		return "", err
	}
	var major, minor int
	if _, err := fmt.Sscanf(string(devStr), "%d:%d", &major, &minor); err != nil {
		return "", err
	}
	evName := filepath.Base(filepath.Dir(globs[0]))
	node := "/dev/input/" + evName
	_ = os.MkdirAll("/dev/input", 0o755)
	_ = syscall.Unlink(node)
	if err := syscall.Mknod(node, syscall.S_IFCHR|0o600, int(major<<8|minor)); err != nil {
		return "", fmt.Errorf("mknod %s: %w", node, err)
	}
	return node, nil
}
