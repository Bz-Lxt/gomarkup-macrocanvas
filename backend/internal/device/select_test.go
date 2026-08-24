package device

import "testing"

func TestSelectTier(t *testing.T) {
	cases := []struct {
		mode string
		ok   bool
		want Tier
		fail bool
	}{
		{"mock", true, TierUserspaceLoop, false},
		{"mock", false, TierUserspaceLoop, false},
		{"real", true, TierKernelVirtual, false},
		{"real", false, "", true},
		{"auto", true, TierKernelVirtual, false},
		{"auto", false, TierUserspaceLoop, false},
	}
	for _, c := range cases {
		got, why := SelectTier(c.mode, c.ok)
		if c.fail && got != "" {
			t.Fatalf("%s kernel=%v want fail got %s (%s)", c.mode, c.ok, got, why)
		}
		if !c.fail && got != c.want {
			t.Fatalf("%s kernel=%v got %s want %s", c.mode, c.ok, got, c.want)
		}
	}
}

func TestSourceFromTier(t *testing.T) {
	if SourceFromTier(TierKernelVirtual) != "kernel_virtual" {
		t.Fatal(SourceFromTier(TierKernelVirtual))
	}
	if SourceFromTier(TierUserspaceLoop) != "simulated" {
		t.Fatal(SourceFromTier(TierUserspaceLoop))
	}
	if SourceFromTier(TierHostPhysical) != "physical" {
		t.Fatal(SourceFromTier(TierHostPhysical))
	}
}
