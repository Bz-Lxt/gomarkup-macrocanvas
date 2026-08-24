package device

import (
	"context"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/hid"
)

type Tier string

const (
	TierHostPhysical   Tier = "T-A"
	TierKernelVirtual  Tier = "T-B"
	TierUserspaceLoop  Tier = "T-C"
	TierNativeAgent    Tier = "T-N"
)

type ProbeResult struct {
	Tier         Tier   `json:"tier"`
	Name         string `json:"name"`
	Available    bool   `json:"available"`
	Reason       string `json:"reason"`
	UinputPath   string `json:"uinput_path"`
	EventNode    string `json:"event_node"`
	GadgetPath   string `json:"gadget_path"`
	RTBudgetOK   bool   `json:"rt_budget_ok"`
	Privileged   bool   `json:"privileged"`
}

type Status struct {
	ActiveTier   Tier          `json:"active_tier"`
	ActiveName   string        `json:"active_name"`
	Mode         string        `json:"mode"`
	SourceFrom   hid.Source    `json:"source_from_tier"`
	Probes       []ProbeResult `json:"probes"`
	Devices      []string      `json:"devices"`
	GadgetOn     bool          `json:"gadget_on"`
	Calibrated   bool          `json:"calibrated"`
	SleepP99Ns   int64         `json:"sleep_p99_ns"`
	MarginNs     int64         `json:"margin_ns"`
	CaptureAuth  bool          `json:"capture_authorized"`
	MaskPrint    bool          `json:"mask_printable"`
	EmergencyKey string        `json:"emergency_hotkey"`
}

func SourceFromTier(t Tier) hid.Source {
	switch t {
	case TierHostPhysical, TierNativeAgent:
		return hid.SourcePhysical
	case TierKernelVirtual:
		return hid.SourceKernelVirtual
	default:
		return hid.SourceSimulated
	}
}

type Envelope struct {
	Seq          uint64     `json:"seq"`
	IngressMono  int64      `json:"ingress_mono_ns"`
	KernelStamp  int64      `json:"kernel_stamp_ns"`
	KernelSkewNs int64      `json:"kernel_skew_ns"`
	BeijingTime  string     `json:"beijing_time"`
	Page         uint16     `json:"page"`
	Usage        uint16     `json:"usage"`
	Value        int32      `json:"value"`
	Kind         string     `json:"kind"`
	Name         string     `json:"name"`
	Source       hid.Source `json:"source"`
	Masked       bool       `json:"masked"`
	Device       string     `json:"device"`
}

type Source interface {
	Start(ctx context.Context, out chan<- Envelope) error
	Stop() error
	Name() string
}

type Sink interface {
	Inject(ev hid.Event) error
	Close() error
	Name() string
}

type Clock interface {
	Mono() time.Duration
	Now() time.Time
}
