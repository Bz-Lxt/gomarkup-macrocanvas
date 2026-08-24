package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/macrocanvas/macrocanvas/internal/api"
	"github.com/macrocanvas/macrocanvas/internal/capture"
	"github.com/macrocanvas/macrocanvas/internal/config"
	"github.com/macrocanvas/macrocanvas/internal/device"
	"github.com/macrocanvas/macrocanvas/internal/logger"
	"github.com/macrocanvas/macrocanvas/internal/macro"
	"github.com/macrocanvas/macrocanvas/internal/storage"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	log := logger.Log()

	db, err := storage.Open(cfg.DataDir)
	if err != nil {
		log.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if _, err := db.GetMacro("sample-p10"); err != nil {
		s := macro.P10Sample()
		if err := db.UpsertMacro(s); err != nil {
			log.Error("seed sample", "err", err)
		}
	}

	cal := timing.CalibrateSleep(80)
	pacer := timing.NewPacer(cal)
	log.Info("pacer calibrated", "margin_ns", cal.Margin.Nanoseconds(), "sleep_p99_ns", cal.SleepP99.Nanoseconds())

	stack := device.NewStack(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := stack.Open(ctx); err != nil {
		log.Error("device stack", "err", err)
		os.Exit(1)
	}
	defer stack.Close()

	ring := capture.NewRing(32768)
	hub := capture.NewHub(ring, cfg.CaptureMaskPrint)
	srvAPI := api.New(cfg, stack, db, hub, ring, pacer, cal)
	srvAPI.Recover()

	go srvAPI.Pump(ctx)
	go hub.FlushLoop(ctx.Done())

	addr := cfg.ListenAddr
	if cfg.BindLoopbackOnly && addr == ":8080" {
		addr = "127.0.0.1:8080"
	}
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srvAPI.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() {
		log.Info("listen", "addr", addr, "tier", stack.ActiveTier(), "mode", cfg.DeviceMode)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	shCtx, shCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer shCancel()
	_ = httpSrv.Shutdown(shCtx)
	cancel()
}
