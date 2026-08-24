package device

// SelectTier is the pure policy used by tests and documented in README §7.
// mode: auto|real|mock. kernelOK means /dev/uinput usable.
func SelectTier(mode string, kernelOK bool) (Tier, string) {
	switch mode {
	case "mock":
		return TierUserspaceLoop, "DEVICE_MODE=mock"
	case "real":
		if kernelOK {
			return TierKernelVirtual, "DEVICE_MODE=real"
		}
		return "", "DEVICE_MODE=real but uinput unavailable"
	default:
		if kernelOK {
			return TierKernelVirtual, "auto: kernel"
		}
		return TierUserspaceLoop, "auto: fallback T-C"
	}
}
