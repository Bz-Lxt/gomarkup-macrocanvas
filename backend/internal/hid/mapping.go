package hid

import "fmt"

// Platform codes are stored as uint16. Missing mappings must error, never drop.

type Platform string

const (
	PlatEvdev   Platform = "evdev"
	PlatWinVK   Platform = "win_vk"
	PlatMacCode Platform = "mac_cg"
)

func ToPlatform(usage uint16, plat Platform) (uint16, error) {
	var m map[uint16]uint16
	switch plat {
	case PlatEvdev:
		m = usageToEvdev
	case PlatWinVK:
		m = usageToWinVK
	case PlatMacCode:
		m = usageToMac
	default:
		return 0, fmt.Errorf("unknown platform %s", plat)
	}
	c, ok := m[usage]
	if !ok {
		return 0, fmt.Errorf("no %s mapping for HID usage 0x%02X (%s)", plat, usage, UsageName(usage))
	}
	return c, nil
}

func FromPlatform(code uint16, plat Platform) (uint16, error) {
	var m map[uint16]uint16
	switch plat {
	case PlatEvdev:
		m = evdevToUsage
	case PlatWinVK:
		m = winVKToUsage
	case PlatMacCode:
		m = macToUsage
	default:
		return 0, fmt.Errorf("unknown platform %s", plat)
	}
	u, ok := m[code]
	if !ok {
		return 0, fmt.Errorf("no HID usage for %s code %d", plat, code)
	}
	return u, nil
}

var (
	usageToEvdev map[uint16]uint16
	evdevToUsage map[uint16]uint16
	usageToWinVK map[uint16]uint16
	winVKToUsage map[uint16]uint16
	usageToMac   map[uint16]uint16
	macToUsage   map[uint16]uint16
)

func init() {
	usageToEvdev, evdevToUsage = invert(evdevPairs)
	usageToWinVK, winVKToUsage = invert(winPairs)
	usageToMac, macToUsage = invert(macPairs)
}

func invert(pairs [][2]uint16) (fwd, rev map[uint16]uint16) {
	fwd = make(map[uint16]uint16, len(pairs))
	rev = make(map[uint16]uint16, len(pairs))
	for _, p := range pairs {
		fwd[p[0]] = p[1]
		rev[p[1]] = p[0]
	}
	return
}

// evdev keycodes (linux/input-event-codes.h). First=HID usage, second=KEY_*.
var evdevPairs = [][2]uint16{
	{KeyA, 30}, {KeyB, 48}, {KeyC, 46}, {KeyD, 32}, {KeyE, 18}, {KeyF, 33},
	{KeyG, 34}, {KeyH, 35}, {KeyI, 23}, {KeyJ, 36}, {KeyK, 37}, {KeyL, 38},
	{KeyM, 50}, {KeyN, 49}, {KeyO, 24}, {KeyP, 25}, {KeyQ, 16}, {KeyR, 19},
	{KeyS, 31}, {KeyT, 20}, {KeyU, 22}, {KeyV, 47}, {KeyW, 17}, {KeyX, 45},
	{KeyY, 21}, {KeyZ, 44},
	{Key1, 2}, {Key2, 3}, {Key3, 4}, {Key4, 5}, {Key5, 6},
	{Key6, 7}, {Key7, 8}, {Key8, 9}, {Key9, 10}, {Key0, 11},
	{KeyEnter, 28}, {KeyEscape, 1}, {KeyBackspace, 14}, {KeyTab, 15}, {KeySpace, 57},
	{KeyMinus, 12}, {KeyEqual, 13}, {KeyLeftBrace, 26}, {KeyRightBrace, 27},
	{KeyBackslash, 43}, {KeySemicolon, 39}, {KeyApostrophe, 40}, {KeyGrave, 41},
	{KeyComma, 51}, {KeyDot, 52}, {KeySlash, 53}, {KeyCapsLock, 58},
	{KeyF1, 59}, {KeyF2, 60}, {KeyF3, 61}, {KeyF4, 62}, {KeyF5, 63}, {KeyF6, 64},
	{KeyF7, 65}, {KeyF8, 66}, {KeyF9, 67}, {KeyF10, 68}, {KeyF11, 87}, {KeyF12, 88},
	{KeyF13, 183}, {KeyF14, 184}, {KeyF15, 185}, {KeyF16, 186},
	{KeyF17, 187}, {KeyF18, 188}, {KeyF19, 189}, {KeyF20, 190},
	{KeyF21, 191}, {KeyF22, 192}, {KeyF23, 193}, {KeyF24, 194},
	{KeyPrintScreen, 99}, {KeyScrollLock, 70}, {KeyPause, 119},
	{KeyInsert, 110}, {KeyHome, 102}, {KeyPageUp, 104}, {KeyDelete, 111},
	{KeyEnd, 107}, {KeyPageDown, 109}, {KeyRight, 106}, {KeyLeft, 105},
	{KeyDown, 108}, {KeyUp, 103}, {KeyNumLock, 69},
	{KeyPadSlash, 98}, {KeyPadAsterisk, 55}, {KeyPadMinus, 74}, {KeyPadPlus, 78},
	{KeyPadEnter, 96}, {KeyPad1, 79}, {KeyPad2, 80}, {KeyPad3, 81}, {KeyPad4, 75},
	{KeyPad5, 76}, {KeyPad6, 77}, {KeyPad7, 71}, {KeyPad8, 72}, {KeyPad9, 73},
	{KeyPad0, 82}, {KeyPadDot, 83}, {KeyApp, 127},
	{KeyLeftCtrl, 29}, {KeyLeftShift, 42}, {KeyLeftAlt, 56}, {KeyLeftGUI, 125},
	{KeyRightCtrl, 97}, {KeyRightShift, 54}, {KeyRightAlt, 100}, {KeyRightGUI, 126},
}

// Windows virtual-key codes.
var winPairs = [][2]uint16{
	{KeyA, 0x41}, {KeyB, 0x42}, {KeyC, 0x43}, {KeyD, 0x44}, {KeyE, 0x45}, {KeyF, 0x46},
	{KeyG, 0x47}, {KeyH, 0x48}, {KeyI, 0x49}, {KeyJ, 0x4A}, {KeyK, 0x4B}, {KeyL, 0x4C},
	{KeyM, 0x4D}, {KeyN, 0x4E}, {KeyO, 0x4F}, {KeyP, 0x50}, {KeyQ, 0x51}, {KeyR, 0x52},
	{KeyS, 0x53}, {KeyT, 0x54}, {KeyU, 0x55}, {KeyV, 0x56}, {KeyW, 0x57}, {KeyX, 0x58},
	{KeyY, 0x59}, {KeyZ, 0x5A},
	{Key1, 0x31}, {Key2, 0x32}, {Key3, 0x33}, {Key4, 0x34}, {Key5, 0x35},
	{Key6, 0x36}, {Key7, 0x37}, {Key8, 0x38}, {Key9, 0x39}, {Key0, 0x30},
	{KeyEnter, 0x0D}, {KeyEscape, 0x1B}, {KeyBackspace, 0x08}, {KeyTab, 0x09}, {KeySpace, 0x20},
	{KeyMinus, 0xBD}, {KeyEqual, 0xBB}, {KeyLeftBrace, 0xDB}, {KeyRightBrace, 0xDD},
	{KeyBackslash, 0xDC}, {KeySemicolon, 0xBA}, {KeyApostrophe, 0xDE}, {KeyGrave, 0xC0},
	{KeyComma, 0xBC}, {KeyDot, 0xBE}, {KeySlash, 0xBF}, {KeyCapsLock, 0x14},
	{KeyF1, 0x70}, {KeyF2, 0x71}, {KeyF3, 0x72}, {KeyF4, 0x73}, {KeyF5, 0x74}, {KeyF6, 0x75},
	{KeyF7, 0x76}, {KeyF8, 0x77}, {KeyF9, 0x78}, {KeyF10, 0x79}, {KeyF11, 0x7A}, {KeyF12, 0x7B},
	{KeyF13, 0x7C}, {KeyF14, 0x7D}, {KeyF15, 0x7E}, {KeyF16, 0x7F},
	{KeyF17, 0x80}, {KeyF18, 0x81}, {KeyF19, 0x82}, {KeyF20, 0x83},
	{KeyF21, 0x84}, {KeyF22, 0x85}, {KeyF23, 0x86}, {KeyF24, 0x87},
	{KeyPrintScreen, 0x2C}, {KeyScrollLock, 0x91}, {KeyPause, 0x13},
	{KeyInsert, 0x2D}, {KeyHome, 0x24}, {KeyPageUp, 0x21}, {KeyDelete, 0x2E},
	{KeyEnd, 0x23}, {KeyPageDown, 0x22}, {KeyRight, 0x27}, {KeyLeft, 0x25},
	{KeyDown, 0x28}, {KeyUp, 0x26}, {KeyNumLock, 0x90},
	{KeyPadSlash, 0x6F}, {KeyPadAsterisk, 0x6A}, {KeyPadMinus, 0x6D}, {KeyPadPlus, 0x6B},
	{KeyPadEnter, 0x0D}, {KeyPad1, 0x61}, {KeyPad2, 0x62}, {KeyPad3, 0x63}, {KeyPad4, 0x64},
	{KeyPad5, 0x65}, {KeyPad6, 0x66}, {KeyPad7, 0x67}, {KeyPad8, 0x68}, {KeyPad9, 0x69},
	{KeyPad0, 0x60}, {KeyPadDot, 0x6E}, {KeyApp, 0x5D},
	{KeyLeftCtrl, 0xA2}, {KeyLeftShift, 0xA0}, {KeyLeftAlt, 0xA4}, {KeyLeftGUI, 0x5B},
	{KeyRightCtrl, 0xA3}, {KeyRightShift, 0xA1}, {KeyRightAlt, 0xA5}, {KeyRightGUI, 0x5C},
}

// macOS virtual key codes (ANSI).
var macPairs = [][2]uint16{
	{KeyA, 0x00}, {KeyB, 0x0B}, {KeyC, 0x08}, {KeyD, 0x02}, {KeyE, 0x0E}, {KeyF, 0x03},
	{KeyG, 0x05}, {KeyH, 0x04}, {KeyI, 0x22}, {KeyJ, 0x26}, {KeyK, 0x28}, {KeyL, 0x25},
	{KeyM, 0x2E}, {KeyN, 0x2D}, {KeyO, 0x1F}, {KeyP, 0x23}, {KeyQ, 0x0C}, {KeyR, 0x0F},
	{KeyS, 0x01}, {KeyT, 0x11}, {KeyU, 0x20}, {KeyV, 0x09}, {KeyW, 0x0D}, {KeyX, 0x07},
	{KeyY, 0x10}, {KeyZ, 0x06},
	{Key1, 0x12}, {Key2, 0x13}, {Key3, 0x14}, {Key4, 0x15}, {Key5, 0x17},
	{Key6, 0x16}, {Key7, 0x1A}, {Key8, 0x1C}, {Key9, 0x19}, {Key0, 0x1D},
	{KeyEnter, 0x24}, {KeyEscape, 0x35}, {KeyBackspace, 0x33}, {KeyTab, 0x30}, {KeySpace, 0x31},
	{KeyMinus, 0x1B}, {KeyEqual, 0x18}, {KeyLeftBrace, 0x21}, {KeyRightBrace, 0x1E},
	{KeyBackslash, 0x2A}, {KeySemicolon, 0x29}, {KeyApostrophe, 0x27}, {KeyGrave, 0x32},
	{KeyComma, 0x2B}, {KeyDot, 0x2F}, {KeySlash, 0x2C}, {KeyCapsLock, 0x39},
	{KeyF1, 0x7A}, {KeyF2, 0x78}, {KeyF3, 0x63}, {KeyF4, 0x76}, {KeyF5, 0x60}, {KeyF6, 0x61},
	{KeyF7, 0x62}, {KeyF8, 0x64}, {KeyF9, 0x65}, {KeyF10, 0x6D}, {KeyF11, 0x67}, {KeyF12, 0x6F},
	{KeyF13, 0x69}, {KeyF14, 0x6B}, {KeyF15, 0x71}, {KeyF16, 0x6A},
	{KeyF17, 0x40}, {KeyF18, 0x4F}, {KeyF19, 0x50}, {KeyF20, 0x5A},
	{KeyF21, 0x5B}, {KeyF22, 0x5C}, {KeyF23, 0x5D}, {KeyF24, 0x5E},
	{KeyPrintScreen, 0x69}, {KeyScrollLock, 0x6B}, {KeyPause, 0x71},
	{KeyInsert, 0x72}, {KeyHome, 0x73}, {KeyPageUp, 0x74}, {KeyDelete, 0x75},
	{KeyEnd, 0x77}, {KeyPageDown, 0x79}, {KeyRight, 0x7C}, {KeyLeft, 0x7B},
	{KeyDown, 0x7D}, {KeyUp, 0x7E}, {KeyNumLock, 0x47},
	{KeyPadSlash, 0x4B}, {KeyPadAsterisk, 0x43}, {KeyPadMinus, 0x4E}, {KeyPadPlus, 0x45},
	{KeyPadEnter, 0x4C}, {KeyPad1, 0x53}, {KeyPad2, 0x54}, {KeyPad3, 0x55}, {KeyPad4, 0x56},
	{KeyPad5, 0x57}, {KeyPad6, 0x58}, {KeyPad7, 0x59}, {KeyPad8, 0x5B}, {KeyPad9, 0x5C},
	{KeyPad0, 0x52}, {KeyPadDot, 0x41}, {KeyApp, 0x6E},
	{KeyLeftCtrl, 0x3B}, {KeyLeftShift, 0x38}, {KeyLeftAlt, 0x3A}, {KeyLeftGUI, 0x37},
	{KeyRightCtrl, 0x3E}, {KeyRightShift, 0x3C}, {KeyRightAlt, 0x3D}, {KeyRightGUI, 0x36},
}
