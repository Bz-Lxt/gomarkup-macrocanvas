package hid

import (
	"fmt"
	"strings"
)

func DumpKeyboard(s KeyboardState) string {
	var b strings.Builder
	b.WriteString("mod=")
	b.WriteString(fmt.Sprintf("%#02x", s.Modifier))
	b.WriteString(" keys=[")
	for i, k := range s.Keys {
		if i > 0 {
			b.WriteByte(' ')
		}
		if k == 0 {
			b.WriteString("00")
			continue
		}
		b.WriteString(UsageName(uint16(k)))
	}
	b.WriteByte(']')
	return b.String()
}

func DumpMouse(s MouseState) string {
	return fmt.Sprintf("btn=%#02x x=%d y=%d wh=%d", s.Buttons, s.X, s.Y, s.Wheel)
}

func DumpDescriptor(desc []byte) string {
	return fmt.Sprintf("hid-rd %d bytes balanced=%v", len(desc), DescriptorValid(desc))
}
