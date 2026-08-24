package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu sync.RWMutex
	L  *slog.Logger
)

func Init(level string) {
	lv := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	mu.Lock()
	L = slog.New(h)
	mu.Unlock()
}

func Log() *slog.Logger {
	mu.RLock()
	l := L
	mu.RUnlock()
	if l == nil {
		Init("info")
		return L
	}
	return l
}

func SetOutput(w io.Writer, level slog.Level) {
	mu.Lock()
	L = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
	mu.Unlock()
}
