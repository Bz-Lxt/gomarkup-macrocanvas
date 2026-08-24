package hid

import (
	"fmt"
	"strings"
)

var usageToName = map[uint16]string{
	KeyA: "A", KeyB: "B", KeyC: "C", KeyD: "D", KeyE: "E", KeyF: "F",
	KeyG: "G", KeyH: "H", KeyI: "I", KeyJ: "J", KeyK: "K", KeyL: "L",
	KeyM: "M", KeyN: "N", KeyO: "O", KeyP: "P", KeyQ: "Q", KeyR: "R",
	KeyS: "S", KeyT: "T", KeyU: "U", KeyV: "V", KeyW: "W", KeyX: "X",
	KeyY: "Y", KeyZ: "Z",
	Key1: "1", Key2: "2", Key3: "3", Key4: "4", Key5: "5",
	Key6: "6", Key7: "7", Key8: "8", Key9: "9", Key0: "0",
	KeyEnter: "Enter", KeyEscape: "Escape", KeyBackspace: "Backspace",
	KeyTab: "Tab", KeySpace: "Space", KeyMinus: "Minus", KeyEqual: "Equal",
	KeyLeftBrace: "LeftBrace", KeyRightBrace: "RightBrace", KeyBackslash: "Backslash",
	KeySemicolon: "Semicolon", KeyApostrophe: "Apostrophe", KeyGrave: "Grave",
	KeyComma: "Comma", KeyDot: "Dot", KeySlash: "Slash", KeyCapsLock: "CapsLock",
	KeyF1: "F1", KeyF2: "F2", KeyF3: "F3", KeyF4: "F4", KeyF5: "F5", KeyF6: "F6",
	KeyF7: "F7", KeyF8: "F8", KeyF9: "F9", KeyF10: "F10", KeyF11: "F11", KeyF12: "F12",
	KeyF13: "F13", KeyF14: "F14", KeyF15: "F15", KeyF16: "F16",
	KeyF17: "F17", KeyF18: "F18", KeyF19: "F19", KeyF20: "F20",
	KeyF21: "F21", KeyF22: "F22", KeyF23: "F23", KeyF24: "F24",
	KeyPrintScreen: "PrintScreen", KeyScrollLock: "ScrollLock", KeyPause: "Pause",
	KeyInsert: "Insert", KeyHome: "Home", KeyPageUp: "PageUp",
	KeyDelete: "Delete", KeyEnd: "End", KeyPageDown: "PageDown",
	KeyRight: "Right", KeyLeft: "Left", KeyDown: "Down", KeyUp: "Up",
	KeyNumLock: "NumLock", KeyPadSlash: "PadSlash", KeyPadAsterisk: "PadAsterisk",
	KeyPadMinus: "PadMinus", KeyPadPlus: "PadPlus", KeyPadEnter: "PadEnter",
	KeyPad1: "Pad1", KeyPad2: "Pad2", KeyPad3: "Pad3", KeyPad4: "Pad4",
	KeyPad5: "Pad5", KeyPad6: "Pad6", KeyPad7: "Pad7", KeyPad8: "Pad8",
	KeyPad9: "Pad9", KeyPad0: "Pad0", KeyPadDot: "PadDot", KeyApp: "App",
	KeyLeftCtrl: "LeftCtrl", KeyLeftShift: "LeftShift", KeyLeftAlt: "LeftAlt",
	KeyLeftGUI: "LeftGUI", KeyRightCtrl: "RightCtrl", KeyRightShift: "RightShift",
	KeyRightAlt: "RightAlt", KeyRightGUI: "RightGUI",
}

var nameToUsage map[string]uint16

func init() {
	nameToUsage = make(map[string]uint16, len(usageToName)+8)
	for u, n := range usageToName {
		nameToUsage[strings.ToLower(n)] = u
	}
	nameToUsage["ctrl"] = KeyLeftCtrl
	nameToUsage["shift"] = KeyLeftShift
	nameToUsage["alt"] = KeyLeftAlt
	nameToUsage["gui"] = KeyLeftGUI
	nameToUsage["win"] = KeyLeftGUI
	nameToUsage["cmd"] = KeyLeftGUI
	nameToUsage["esc"] = KeyEscape
	nameToUsage["return"] = KeyEnter
}

func UsageName(usage uint16) string {
	if n, ok := usageToName[usage]; ok {
		return n
	}
	return fmt.Sprintf("Usage_0x%02X", usage)
}

func ParseUsage(name string) (uint16, error) {
	k := strings.ToLower(strings.TrimSpace(name))
	if u, ok := nameToUsage[k]; ok {
		return u, nil
	}
	return 0, fmt.Errorf("unknown key %q", name)
}

// ParseCombo parses "LeftCtrl+Shift+A" into ordered usages (modifiers first).
func ParseCombo(expr string) ([]uint16, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty combo")
	}
	parts := strings.Split(expr, "+")
	out := make([]uint16, 0, len(parts))
	seen := map[uint16]bool{}
	for _, p := range parts {
		u, err := ParseUsage(p)
		if err != nil {
			return nil, err
		}
		if seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out, nil
}

func FormatCombo(usages []uint16) string {
	names := make([]string, 0, len(usages))
	for _, u := range usages {
		names = append(names, UsageName(u))
	}
	return strings.Join(names, "+")
}

func ButtonName(usage uint16) string {
	switch usage {
	case Btn1:
		return "Left"
	case Btn2:
		return "Right"
	case Btn3:
		return "Middle"
	default:
		return fmt.Sprintf("Button%d", usage)
	}
}

func ParseButton(name string) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "left", "btn1", "1":
		return Btn1, nil
	case "right", "btn2", "2":
		return Btn2, nil
	case "middle", "btn3", "3":
		return Btn3, nil
	default:
		return 0, fmt.Errorf("unknown button %q", name)
	}
}

func KnownKeyboardUsages() []uint16 {
	out := make([]uint16, 0, len(usageToName))
	for u := range usageToName {
		out = append(out, u)
	}
	return out
}
