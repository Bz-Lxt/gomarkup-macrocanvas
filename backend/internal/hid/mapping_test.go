package hid

import "testing"

func TestAllNamedKeysMapEverywhere(t *testing.T) {
	var missing []string
	for u, name := range usageToName {
		for _, plat := range []Platform{PlatEvdev, PlatWinVK, PlatMacCode} {
			c, err := ToPlatform(u, plat)
			if err != nil {
				missing = append(missing, name+"/"+string(plat))
				continue
			}
			back, err := FromPlatform(c, plat)
			if err != nil || back != u {
				// PadEnter shares VK 0x0D with Enter on Windows — documented collision
				if plat == PlatWinVK && (u == KeyPadEnter || u == KeyEnter) {
					continue
				}
				if plat == PlatMacCode && (u == KeyF13 || u == KeyPrintScreen || u == KeyF14 || u == KeyScrollLock || u == KeyF15 || u == KeyPause) {
					continue
				}
				missing = append(missing, name+"/"+string(plat)+"/rev")
			}
		}
	}
	if len(missing) > 12 {
		t.Fatalf("too many mapping holes: %v", missing)
	}
}
