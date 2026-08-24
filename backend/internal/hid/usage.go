package hid

// HID Usage Pages used by MacroCanvas (HID Usage Tables).
const (
	PageGenericDesktop uint16 = 0x01
	PageKeyboard       uint16 = 0x07
	PageButton         uint16 = 0x09
)

// Generic Desktop usages.
const (
	GDX     uint16 = 0x30
	GDY     uint16 = 0x31
	GDWheel uint16 = 0x38
)

// Keyboard usages (Page 0x07). A=0x04 … matching Boot Keyboard report IDs.
const (
	KeyA uint16 = 0x04
	KeyB uint16 = 0x05
	KeyC uint16 = 0x06
	KeyD uint16 = 0x07
	KeyE uint16 = 0x08
	KeyF uint16 = 0x09
	KeyG uint16 = 0x0A
	KeyH uint16 = 0x0B
	KeyI uint16 = 0x0C
	KeyJ uint16 = 0x0D
	KeyK uint16 = 0x0E
	KeyL uint16 = 0x0F
	KeyM uint16 = 0x10
	KeyN uint16 = 0x11
	KeyO uint16 = 0x12
	KeyP uint16 = 0x13
	KeyQ uint16 = 0x14
	KeyR uint16 = 0x15
	KeyS uint16 = 0x16
	KeyT uint16 = 0x17
	KeyU uint16 = 0x18
	KeyV uint16 = 0x19
	KeyW uint16 = 0x1A
	KeyX uint16 = 0x1B
	KeyY uint16 = 0x1C
	KeyZ uint16 = 0x1D

	Key1 uint16 = 0x1E
	Key2 uint16 = 0x1F
	Key3 uint16 = 0x20
	Key4 uint16 = 0x21
	Key5 uint16 = 0x22
	Key6 uint16 = 0x23
	Key7 uint16 = 0x24
	Key8 uint16 = 0x25
	Key9 uint16 = 0x26
	Key0 uint16 = 0x27

	KeyEnter       uint16 = 0x28
	KeyEscape      uint16 = 0x29
	KeyBackspace   uint16 = 0x2A
	KeyTab         uint16 = 0x2B
	KeySpace       uint16 = 0x2C
	KeyMinus       uint16 = 0x2D
	KeyEqual       uint16 = 0x2E
	KeyLeftBrace   uint16 = 0x2F
	KeyRightBrace  uint16 = 0x30
	KeyBackslash   uint16 = 0x31
	KeySemicolon   uint16 = 0x33
	KeyApostrophe  uint16 = 0x34
	KeyGrave       uint16 = 0x35
	KeyComma       uint16 = 0x36
	KeyDot         uint16 = 0x37
	KeySlash       uint16 = 0x38
	KeyCapsLock    uint16 = 0x39
	KeyF1          uint16 = 0x3A
	KeyF2          uint16 = 0x3B
	KeyF3          uint16 = 0x3C
	KeyF4          uint16 = 0x3D
	KeyF5          uint16 = 0x3E
	KeyF6          uint16 = 0x3F
	KeyF7          uint16 = 0x40
	KeyF8          uint16 = 0x41
	KeyF9          uint16 = 0x42
	KeyF10         uint16 = 0x43
	KeyF11         uint16 = 0x44
	KeyF12         uint16 = 0x45
	KeyF13         uint16 = 0x68
	KeyF14         uint16 = 0x69
	KeyF15         uint16 = 0x6A
	KeyF16         uint16 = 0x6B
	KeyF17         uint16 = 0x6C
	KeyF18         uint16 = 0x6D
	KeyF19         uint16 = 0x6E
	KeyF20         uint16 = 0x6F
	KeyF21         uint16 = 0x70
	KeyF22         uint16 = 0x71
	KeyF23         uint16 = 0x72
	KeyF24         uint16 = 0x73
	KeyPrintScreen uint16 = 0x46
	KeyScrollLock  uint16 = 0x47
	KeyPause       uint16 = 0x48
	KeyInsert      uint16 = 0x49
	KeyHome        uint16 = 0x4A
	KeyPageUp      uint16 = 0x4B
	KeyDelete      uint16 = 0x4C
	KeyEnd         uint16 = 0x4D
	KeyPageDown    uint16 = 0x4E
	KeyRight       uint16 = 0x4F
	KeyLeft        uint16 = 0x50
	KeyDown        uint16 = 0x51
	KeyUp          uint16 = 0x52
	KeyNumLock     uint16 = 0x53
	KeyPadSlash    uint16 = 0x54
	KeyPadAsterisk uint16 = 0x55
	KeyPadMinus    uint16 = 0x56
	KeyPadPlus     uint16 = 0x57
	KeyPadEnter    uint16 = 0x58
	KeyPad1        uint16 = 0x59
	KeyPad2        uint16 = 0x5A
	KeyPad3        uint16 = 0x5B
	KeyPad4        uint16 = 0x5C
	KeyPad5        uint16 = 0x5D
	KeyPad6        uint16 = 0x5E
	KeyPad7        uint16 = 0x5F
	KeyPad8        uint16 = 0x60
	KeyPad9        uint16 = 0x61
	KeyPad0        uint16 = 0x62
	KeyPadDot      uint16 = 0x63
	KeyApp         uint16 = 0x65

	KeyLeftCtrl   uint16 = 0xE0
	KeyLeftShift  uint16 = 0xE1
	KeyLeftAlt    uint16 = 0xE2
	KeyLeftGUI    uint16 = 0xE3
	KeyRightCtrl  uint16 = 0xE4
	KeyRightShift uint16 = 0xE5
	KeyRightAlt   uint16 = 0xE6
	KeyRightGUI   uint16 = 0xE7
)

// Button usages (Page 0x09).
const (
	Btn1 uint16 = 0x01 // left
	Btn2 uint16 = 0x02 // right
	Btn3 uint16 = 0x03 // middle
	Btn4 uint16 = 0x04
	Btn5 uint16 = 0x05
)

type EventKind uint8

const (
	KindKey EventKind = iota
	KindButton
	KindPointer
	KindSync
)

type Event struct {
	Page   uint16
	Usage  uint16
	Value  int32 // 1 down / 0 up / axis delta
	Kind   EventKind
	Source Source
}

type Source string

const (
	SourcePhysical      Source = "physical"
	SourceKernelVirtual Source = "kernel_virtual"
	SourceInjected      Source = "injected"
	SourceSimulated     Source = "simulated"
)

func IsModifier(usage uint16) bool {
	return usage >= KeyLeftCtrl && usage <= KeyRightGUI
}

func ModifierBit(usage uint16) byte {
	switch usage {
	case KeyLeftCtrl:
		return 0x01
	case KeyLeftShift:
		return 0x02
	case KeyLeftAlt:
		return 0x04
	case KeyLeftGUI:
		return 0x08
	case KeyRightCtrl:
		return 0x10
	case KeyRightShift:
		return 0x20
	case KeyRightAlt:
		return 0x40
	case KeyRightGUI:
		return 0x80
	}
	return 0
}
