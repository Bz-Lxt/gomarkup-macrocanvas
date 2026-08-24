package hid

import "fmt"

const (
	BootKeyboardLen = 8
	BootMouseLen    = 4
)

// KeyboardState is the 6KRO boot-protocol keyboard.
type KeyboardState struct {
	Modifier byte
	Keys     [6]byte
}

func (s KeyboardState) Encode() [8]byte {
	var r [8]byte
	r[0] = s.Modifier
	copy(r[2:], s.Keys[:])
	return r
}

func DecodeKeyboard(b []byte) (KeyboardState, error) {
	if len(b) < BootKeyboardLen {
		return KeyboardState{}, fmt.Errorf("keyboard report wants %d bytes, got %d", BootKeyboardLen, len(b))
	}
	var s KeyboardState
	s.Modifier = b[0]
	copy(s.Keys[:], b[2:8])
	return s, nil
}

func (s *KeyboardState) Press(usage uint16) error {
	if usage == 0 {
		return fmt.Errorf("usage 0 is reserved")
	}
	if bit := ModifierBit(usage); bit != 0 {
		s.Modifier |= bit
		return nil
	}
	if usage > 0xFF {
		return fmt.Errorf("usage 0x%X exceeds boot report", usage)
	}
	code := byte(usage)
	for _, k := range s.Keys {
		if k == code {
			return nil
		}
	}
	for i, k := range s.Keys {
		if k == 0 {
			s.Keys[i] = code
			return nil
		}
	}
	return fmt.Errorf("6-key rollover exceeded")
}

func (s *KeyboardState) Release(usage uint16) {
	if bit := ModifierBit(usage); bit != 0 {
		s.Modifier &^= bit
		return
	}
	code := byte(usage)
	for i, k := range s.Keys {
		if k == code {
			s.Keys[i] = 0
		}
	}
	// compact
	n := 0
	var compact [6]byte
	for _, k := range s.Keys {
		if k != 0 {
			compact[n] = k
			n++
		}
	}
	s.Keys = compact
}

func (s KeyboardState) DownUsages() []uint16 {
	out := make([]uint16, 0, 8)
	for _, u := range []uint16{KeyLeftCtrl, KeyLeftShift, KeyLeftAlt, KeyLeftGUI, KeyRightCtrl, KeyRightShift, KeyRightAlt, KeyRightGUI} {
		if s.Modifier&ModifierBit(u) != 0 {
			out = append(out, u)
		}
	}
	for _, k := range s.Keys {
		if k != 0 {
			out = append(out, uint16(k))
		}
	}
	return out
}

type MouseState struct {
	Buttons byte
	X       int8
	Y       int8
	Wheel   int8
}

func (s MouseState) Encode() [4]byte {
	return [4]byte{s.Buttons, byte(s.X), byte(s.Y), byte(s.Wheel)}
}

func DecodeMouse(b []byte) (MouseState, error) {
	if len(b) < 3 {
		return MouseState{}, fmt.Errorf("mouse report wants at least 3 bytes, got %d", len(b))
	}
	s := MouseState{Buttons: b[0], X: int8(b[1]), Y: int8(b[2])}
	if len(b) >= 4 {
		s.Wheel = int8(b[3])
	}
	return s, nil
}

func (s *MouseState) SetButton(usage uint16, down bool) error {
	if usage < 1 || usage > 5 {
		return fmt.Errorf("button usage %d out of range", usage)
	}
	bit := byte(1 << (usage - 1))
	if down {
		s.Buttons |= bit
	} else {
		s.Buttons &^= bit
	}
	return nil
}
