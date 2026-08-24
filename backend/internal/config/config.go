package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr        string
	DataDir           string
	DeviceMode        string // auto | real | mock
	HIDGadgetEnabled  bool
	AuthToken         string
	BindLoopbackOnly  bool
	LogLevel          string
	PublicHost        string
	CORSExtra         []string
	MaxMacroIters     int
	MaxWallClockMs    int
	EmergencyHotkey   string
	CaptureMaskPrint  bool
	CaptureAuthorized bool
}

func Load() Config {
	c := Config{
		ListenAddr:       env("LISTEN_ADDR", ":8080"),
		DataDir:          env("DATA_DIR", "./data"),
		DeviceMode:       strings.ToLower(env("DEVICE_MODE", "auto")),
		HIDGadgetEnabled: envBool("HID_GADGET_ENABLED", false),
		AuthToken:        env("AUTH_TOKEN", "mc-dev-31821"),
		BindLoopbackOnly: envBool("BIND_LOOPBACK_ONLY", false),
		LogLevel:         env("LOG_LEVEL", "info"),
		PublicHost:       env("PUBLIC_HOST", "localhost:31821"),
		CORSExtra:        splitCSV(env("CORS_EXTRA", "")),
		MaxMacroIters:    envInt("MAX_MACRO_ITERS", 10000),
		MaxWallClockMs:   envInt("MAX_WALL_CLOCK_MS", 120000),
		EmergencyHotkey:  env("EMERGENCY_HOTKEY", "LeftCtrl+LeftShift+Escape"),
		CaptureMaskPrint: envBool("CAPTURE_MASK_PRINTABLE", true),
	}
	switch c.DeviceMode {
	case "auto", "real", "mock":
	default:
		c.DeviceMode = "auto"
	}
	return c
}

func (c Config) SimulatorEnabled() bool {
	return c.DeviceMode == "mock" || c.DeviceMode == "auto"
}

func (c Config) DeviceIngestEnabled() bool {
	return c.DeviceMode == "real" || c.DeviceMode == "auto"
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
