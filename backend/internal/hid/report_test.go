package hid

import "testing"

func TestKeyboardRoundTrip(t *testing.T) {
	var s KeyboardState
	if err := s.Press(KeyA); err != nil {
		t.Fatal(err)
	}
	if err := s.Press(KeyLeftShift); err != nil {
		t.Fatal(err)
	}
	enc := s.Encode()
	if enc[0] != 0x02 {
		t.Fatalf("modifier = %x", enc[0])
	}
	dec, err := DecodeKeyboard(enc[:])
	if err != nil {
		t.Fatal(err)
	}
	downs := dec.DownUsages()
	if len(downs) != 2 {
		t.Fatalf("downs=%v", downs)
	}
	s.Release(KeyA)
	s.Release(KeyLeftShift)
	if s.Modifier != 0 || s.Keys != [6]byte{} {
		t.Fatalf("not released: %+v", s)
	}
}

func TestRollover(t *testing.T) {
	var s KeyboardState
	for i := 0; i < 6; i++ {
		if err := s.Press(KeyA + uint16(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Press(KeyG); err == nil {
		t.Fatal("expected 6KRO error")
	}
}

func TestMouseButtons(t *testing.T) {
	var s MouseState
	if err := s.SetButton(Btn1, true); err != nil {
		t.Fatal(err)
	}
	enc := s.Encode()
	if enc[0]&0x01 == 0 {
		t.Fatal("left button missing")
	}
	dec, err := DecodeMouse(enc[:])
	if err != nil {
		t.Fatal(err)
	}
	if dec.Buttons != 1 {
		t.Fatalf("buttons=%d", dec.Buttons)
	}
}

func TestDescriptorBalanced(t *testing.T) {
	d := CompositeReportDescriptor()
	if !DescriptorValid(d) {
		t.Fatal("descriptor not balanced")
	}
}

func TestComboParse(t *testing.T) {
	us, err := ParseCombo("LeftCtrl+Shift+A")
	if err != nil {
		t.Fatal(err)
	}
	if len(us) != 3 || us[2] != KeyA {
		t.Fatalf("%v", us)
	}
	if _, err := ParseUsage("NotAKey"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMappingsBidirectional(t *testing.T) {
	for _, plat := range []Platform{PlatEvdev, PlatWinVK, PlatMacCode} {
		c, err := ToPlatform(KeyA, plat)
		if err != nil {
			t.Fatalf("%s: %v", plat, err)
		}
		u, err := FromPlatform(c, plat)
		if err != nil || u != KeyA {
			t.Fatalf("%s reverse %v %v", plat, u, err)
		}
	}
	if _, err := ToPlatform(0x00, PlatEvdev); err == nil {
		t.Fatal("missing mapping must error")
	}
}
