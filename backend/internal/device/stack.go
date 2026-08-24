package device

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/macrocanvas/macrocanvas/internal/config"
	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/logger"
)

// Stack owns the active Source/Sink pair and records probe results.
type Stack struct {
	mu     sync.RWMutex
	cfg    config.Config
	tier   Tier
	src    Source
	sink   Sink
	loop   *Loopback
	uinput *UInput
	gadget Sink
	probes []ProbeResult
	bus    chan Envelope
}

func NewStack(cfg config.Config) *Stack {
	return &Stack{cfg: cfg, loop: NewLoopback(), bus: make(chan Envelope, 4096)}
}

func (s *Stack) Bus() <-chan Envelope { return s.bus }

func (s *Stack) Open(ctx context.Context) error {
	s.probes = probeAll(s.cfg)
	mode := s.cfg.DeviceMode
	var err error
	switch mode {
	case "mock":
		err = s.useLoop()
	case "real":
		err = s.useKernelOrFail()
	default:
		if e := s.useKernelOrFail(); e != nil {
			logger.Log().Warn("kernel tier unavailable, falling back to T-C", "err", e)
			err = s.useLoop()
		}
	}
	if err != nil {
		return err
	}
	if s.cfg.HIDGadgetEnabled && HasGadget() {
		if g, e := OpenGadget("/dev/hidg0"); e == nil {
			s.gadget = g
			s.sink = multiSink{s.sink, g}
		}
	}
	return s.src.Start(ctx, s.bus)
}

func (s *Stack) useLoop() error {
	s.tier = TierUserspaceLoop
	s.src = s.loop
	s.sink = taggedSink{inner: s.loop, src: hid.SourceInjected}
	return nil
}

func (s *Stack) useKernelOrFail() error {
	ui, err := OpenUInput()
	if err != nil {
		return err
	}
	s.uinput = ui
	s.tier = TierKernelVirtual
	s.src = ui
	s.sink = taggedSink{inner: ui, src: hid.SourceInjected}
	// T-A: if extra host event nodes exist besides our virtual one, also listen.
	if nodes := ListEventNodes(); len(nodes) > 1 {
		if ev, e := OpenEvdev(false); e == nil {
			s.src = multiSource{ui, ev}
			s.tier = TierHostPhysical
		}
	}
	return nil
}

func (s *Stack) Inject(ev hid.Event) error {
	s.mu.RLock()
	sink := s.sink
	s.mu.RUnlock()
	if sink == nil {
		return errClosed
	}
	ev.Source = hid.SourceInjected
	return sink.Inject(ev)
}

func (s *Stack) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.src != nil {
		_ = s.src.Stop()
	}
	if s.uinput != nil {
		_ = s.uinput.Close()
	}
	if s.gadget != nil {
		_ = s.gadget.Close()
	}
	return s.loop.Close()
}

func (s *Stack) ActiveTier() Tier {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tier
}

func (s *Stack) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	name := map[Tier]string{
		TierHostPhysical:  "宿主物理档",
		TierKernelVirtual: "内核虚拟档",
		TierUserspaceLoop: "用户态回环档",
		TierNativeAgent:   "宿主原生 Agent",
	}[s.tier]
	devs := ListEventNodes()
	if devs == nil {
		devs = []string{}
	}
	if s.uinput != nil {
		if n := s.uinput.Node(); n != "" && !contains(devs, n) {
			devs = append(devs, n)
		}
	}
	return Status{
		ActiveTier:   s.tier,
		ActiveName:   name,
		Mode:         s.cfg.DeviceMode,
		SourceFrom:   SourceFromTier(s.tier),
		Probes:       s.probes,
		Devices:      devs,
		GadgetOn:     s.gadget != nil,
		CaptureAuth:  s.cfg.CaptureAuthorized,
		MaskPrint:    s.cfg.CaptureMaskPrint,
		EmergencyKey: s.cfg.EmergencyHotkey,
	}
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func probeAll(cfg config.Config) []ProbeResult {
	out := []ProbeResult{
		{Tier: TierHostPhysical, Name: "宿主物理档", Available: len(ListEventNodes()) > 0 && HasUinput(), Reason: hostReason()},
		{Tier: TierKernelVirtual, Name: "内核虚拟档", Available: HasUinput(), UinputPath: "/dev/uinput", Privileged: isPrivileged(), Reason: uinputReason()},
		{Tier: TierUserspaceLoop, Name: "用户态回环档", Available: true, Reason: "always"},
		{Tier: TierNativeAgent, Name: "宿主原生 Agent", Available: false, Reason: "V2: mc-agent 未连接"},
	}
	out[1].RTBudgetOK = rtBudgetOK()
	if HasGadget() {
		out[1].GadgetPath = "/dev/hidg0"
	}
	_ = cfg
	return out
}

func hostReason() string {
	if !HasUinput() {
		return "无 /dev/uinput"
	}
	if len(ListEventNodes()) == 0 {
		return "无宿主 event 节点（容器内常见）"
	}
	return "event 节点可见"
}

func uinputReason() string {
	if HasUinput() {
		return "/dev/uinput 可访问"
	}
	return "/dev/uinput 不可用，将降级 T-C"
}

func isPrivileged() bool {
	f, err := os.Open("/dev/uinput")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func rtBudgetOK() bool {
	if b, err := os.ReadFile("/sys/fs/cgroup/cpu.rt_runtime_us"); err == nil {
		return string(b) != "0\n" && string(b) != "0"
	}
	// cgroup v2 has no RT interface → unsafe
	if _, err := os.Stat("/sys/fs/cgroup/cpu.max"); err == nil {
		return false
	}
	return false
}

type taggedSink struct {
	inner Sink
	src   hid.Source
}

func (t taggedSink) Name() string { return t.inner.Name() }
func (t taggedSink) Close() error { return t.inner.Close() }
func (t taggedSink) Inject(ev hid.Event) error {
	ev.Source = t.src
	return t.inner.Inject(ev)
}

type multiSink []Sink

func (m multiSink) Name() string { return "multi-sink" }
func (m multiSink) Close() error {
	var first error
	for _, s := range m {
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
func (m multiSink) Inject(ev hid.Event) error {
	var first error
	for _, s := range m {
		if err := s.Inject(ev); err != nil && first == nil {
			first = err
		}
	}
	return first
}

type multiSource []Source

func (m multiSource) Name() string { return "multi-source" }
func (m multiSource) Stop() error {
	for _, s := range m {
		_ = s.Stop()
	}
	return nil
}
func (m multiSource) Start(ctx context.Context, out chan<- Envelope) error {
	var n int
	var last error
	for _, s := range m {
		if err := s.Start(ctx, out); err != nil {
			last = err
			continue
		}
		n++
	}
	if n == 0 {
		if last != nil {
			return last
		}
		return fmt.Errorf("no source started")
	}
	return nil
}
