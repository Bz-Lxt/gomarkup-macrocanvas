package trigger

import (
	"sort"
	"strings"
	"sync"

	"github.com/macrocanvas/macrocanvas/internal/device"
	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/macro"
)

type Binding struct {
	MacroID string
	Combo   []uint16
	Key     string
}

type Manager struct {
	mu       sync.Mutex
	bindings []Binding
	down     map[uint16]bool
	fire     func(macroID string)
	enabled  bool
}

func NewManager(fire func(string)) *Manager {
	return &Manager{down: map[uint16]bool{}, fire: fire, enabled: true}
}

func (m *Manager) SetEnabled(v bool) {
	m.mu.Lock()
	m.enabled = v
	m.mu.Unlock()
}

func (m *Manager) Replace(macros []macro.Macro) error {
	var next []Binding
	seen := map[string]string{}
	for _, mac := range macros {
		if !mac.Enabled || !mac.Deployed || mac.Trigger.Kind != "hotkey" {
			continue
		}
		us, err := hid.ParseCombo(mac.Trigger.Hotkey)
		if err != nil {
			return err
		}
		key := comboKey(us)
		if other, ok := seen[key]; ok {
			return errConflict(other, mac.ID, mac.Trigger.Hotkey)
		}
		seen[key] = mac.ID
		next = append(next, Binding{MacroID: mac.ID, Combo: us, Key: key})
	}
	m.mu.Lock()
	m.bindings = next
	m.mu.Unlock()
	return nil
}

func (m *Manager) OnEvent(e device.Envelope) {
	if e.Kind != "key" {
		return
	}
	// never fire on injected/simulated — prevents self-trigger (C6)
	if e.Source == hid.SourceInjected || e.Source == hid.SourceSimulated {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled {
		return
	}
	if e.Value != 0 {
		m.down[e.Usage] = true
	} else {
		delete(m.down, e.Usage)
		return
	}
	for _, b := range m.bindings {
		if match(m.down, b.Combo) && m.fire != nil {
			go m.fire(b.MacroID)
		}
	}
}

func match(down map[uint16]bool, combo []uint16) bool {
	if len(combo) == 0 {
		return false
	}
	for _, u := range combo {
		if !down[u] {
			return false
		}
	}
	return true
}

func comboKey(us []uint16) string {
	cp := append([]uint16(nil), us...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	var b strings.Builder
	for i, u := range cp {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(hid.UsageName(u))
	}
	return b.String()
}

type conflictError struct{ a, b, key string }

func errConflict(a, b, key string) error { return conflictError{a, b, key} }
func (e conflictError) Error() string {
	return "hotkey conflict " + e.key + " between " + e.a + " and " + e.b
}
