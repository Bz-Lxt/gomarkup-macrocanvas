package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/macrocanvas/macrocanvas/internal/capture"
	"github.com/macrocanvas/macrocanvas/internal/clock"
	"github.com/macrocanvas/macrocanvas/internal/config"
	"github.com/macrocanvas/macrocanvas/internal/device"
	"github.com/macrocanvas/macrocanvas/internal/engine"
	"github.com/macrocanvas/macrocanvas/internal/hid"
	"github.com/macrocanvas/macrocanvas/internal/logger"
	"github.com/macrocanvas/macrocanvas/internal/macro"
	"github.com/macrocanvas/macrocanvas/internal/storage"
	"github.com/macrocanvas/macrocanvas/internal/timing"
	"github.com/macrocanvas/macrocanvas/internal/trigger"
)

type Server struct {
	cfg     config.Config
	stack   *device.Stack
	db      *storage.DB
	hub     *capture.Hub
	ring    *capture.Ring
	pacer   *timing.Pacer
	cal     timing.Calibration
	reg     *engine.Registry
	safety  *engine.Safety
	trig    *trigger.Manager
	global  bool
	authCap bool
	mu      sync.Mutex
}

func New(cfg config.Config, stack *device.Stack, db *storage.DB, hub *capture.Hub, ring *capture.Ring, pacer *timing.Pacer, cal timing.Calibration) *Server {
	s := &Server{
		cfg: cfg, stack: stack, db: db, hub: hub, ring: ring, pacer: pacer, cal: cal,
		reg: engine.NewRegistry(), safety: engine.NewSafety(cfg.MaxMacroIters, cfg.MaxWallClockMs, 8000),
		global: true, authCap: false,
	}
	s.trig = trigger.NewManager(func(id string) { _, _ = s.runMacro(id, "hotkey") })
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/status", s.auth(s.status))
	mux.HandleFunc("GET /api/v1/macros", s.auth(s.listMacros))
	mux.HandleFunc("POST /api/v1/macros", s.auth(s.createMacro))
	mux.HandleFunc("GET /api/v1/macros/{id}", s.auth(s.getMacro))
	mux.HandleFunc("PUT /api/v1/macros/{id}", s.auth(s.putMacro))
	mux.HandleFunc("DELETE /api/v1/macros/{id}", s.auth(s.delMacro))
	mux.HandleFunc("POST /api/v1/macros/{id}/validate", s.auth(s.validateMacro))
	mux.HandleFunc("POST /api/v1/macros/{id}/deploy", s.auth(s.deployMacro))
	mux.HandleFunc("POST /api/v1/macros/{id}/run", s.auth(s.runHandler))
	mux.HandleFunc("POST /api/v1/macros/{id}/stop", s.auth(s.stopOne))
	mux.HandleFunc("POST /api/v1/emergency-stop", s.auth(s.emergency))
	mux.HandleFunc("GET /api/v1/runs", s.auth(s.listRuns))
	mux.HandleFunc("GET /api/v1/runs/{id}", s.auth(s.getRun))
	mux.HandleFunc("GET /api/v1/runs/{id}/trace", s.auth(s.getTrace))
	mux.HandleFunc("POST /api/v1/benchmark", s.auth(s.benchmark))
	mux.HandleFunc("GET /api/v1/events", s.auth(s.events))
	mux.HandleFunc("POST /api/v1/events/clear", s.auth(s.clearEvents))
	mux.HandleFunc("GET /api/v1/settings", s.auth(s.getSettings))
	mux.HandleFunc("PUT /api/v1/settings", s.auth(s.putSettings))
	mux.HandleFunc("POST /api/v1/capture/authorize", s.auth(s.authorizeCapture))
	mux.HandleFunc("GET /api/v1/hid/keys", s.auth(s.listKeys))
	mux.HandleFunc("GET /ws/events", s.wsEvents)
	mux.HandleFunc("GET /ws/exec", s.wsExec)
	return s.wrap(mux)
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: 200}
		if r.URL.Path != "/health" && !originAllowed(r, s.cfg.CORSExtra, s.cfg.PublicHost) && r.Header.Get("Origin") != "" {
			if strings.HasPrefix(r.URL.Path, "/ws/") {
				http.Error(sw, "origin denied", http.StatusForbidden)
				return
			}
			writeErr(sw, "forbid", CodeForbidden, "origin denied")
			return
		}
		next.ServeHTTP(sw, r)
	})
}

func (s *Server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" {
			tok = r.Header.Get("X-Auth-Token")
		}
		if tok == "" {
			tok = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(tok), []byte(s.cfg.AuthToken)) != 1 {
			writeErr(w, "unauth", CodeUnauthorized, "invalid token")
			return
		}
		h(w, r)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.stack.Status()
	writeOK(w, map[string]any{
		"status": "ok", "service": "macrocanvas", "tier": st.ActiveTier,
		"time": clock.Format(clock.Now()),
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, "bad", CodeValidation, "invalid json")
		return
	}
	if body.Username != "geek" || body.Password != "phosphor" {
		writeErr(w, "unauth", CodeUnauthorized, "bad credentials")
		return
	}
	writeOK(w, map[string]any{"token": s.cfg.AuthToken, "username": "geek"})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	st := s.stack.Status()
	st.Calibrated = true
	st.SleepP99Ns = s.cal.SleepP99.Nanoseconds()
	st.MarginNs = s.cal.Margin.Nanoseconds()
	st.CaptureAuth = s.hub.Authorized()
	st.MaskPrint = s.hub.Mask()
	rtOK, rtWhy := timing.RTAvailable()
	writeOK(w, map[string]any{
		"device": st, "rt_available": rtOK, "rt_reason": rtWhy,
		"global_enabled": s.global, "dropped_events": s.ring.Dropped(),
		"disclaimer": "时序承诺按环境分档（E1/E2/E3），禁止解读为无条件微秒级保证。",
	})
}

func (s *Server) listMacros(w http.ResponseWriter, r *http.Request) {
	ms, err := s.db.ListMacros()
	if err != nil {
		writeErr(w, "int", CodeInternal, err.Error())
		return
	}
	if ms == nil {
		ms = []macro.Macro{}
	}
	writeOK(w, ms)
}

func (s *Server) createMacro(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	m, errs := macro.DecodeAndValidate(raw)
	if len(errs) > 0 {
		writeErr(w, "bad", CodeImport, errs[0].Error())
		return
	}
	if m.ID == "" {
		m.ID = storage.NewMacroID()
	}
	if err := s.db.UpsertMacro(m); err != nil {
		writeErr(w, "int", CodeInternal, err.Error())
		return
	}
	s.db.Audit("macro.create", m.ID)
	_ = s.reloadTriggers()
	writeOK(w, m)
}

func (s *Server) getMacro(w http.ResponseWriter, r *http.Request) {
	m, err := s.db.GetMacro(r.PathValue("id"))
	if err != nil {
		writeErr(w, "notfound", CodeNotFound, "macro not found")
		return
	}
	writeOK(w, m)
}

func (s *Server) putMacro(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	raw, _ := io.ReadAll(r.Body)
	m, errs := macro.DecodeAndValidate(raw)
	if len(errs) > 0 {
		writeErr(w, "bad", CodeImport, errs[0].Error())
		return
	}
	m.ID = id
	if err := s.db.UpsertMacro(m); err != nil {
		writeErr(w, "int", CodeInternal, err.Error())
		return
	}
	s.db.Audit("macro.update", id)
	_ = s.reloadTriggers()
	writeOK(w, m)
}

func (s *Server) delMacro(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeleteMacro(id); err != nil {
		if errors.Is(err, storage.ErrMacroNotFound) {
			writeErr(w, "notfound", CodeNotFound, "macro not found")
			return
		}
		writeErr(w, "int", CodeInternal, err.Error())
		return
	}
	s.db.Audit("macro.delete", id)
	_ = s.reloadTriggers()
	writeOK(w, map[string]any{"deleted": true})
}

func (s *Server) validateMacro(w http.ResponseWriter, r *http.Request) {
	m, err := s.db.GetMacro(r.PathValue("id"))
	if err != nil {
		writeErr(w, "notfound", CodeNotFound, "macro not found")
		return
	}
	issues := engine.ValidateGraph(m)
	prog, cerrs := engine.Compile(m)
	ok := len(issues) == 0 && len(cerrs) == 0
	var unpaired []engine.Issue
	ops := 0
	if prog != nil {
		unpaired = engine.CheckUnpairedKeys(prog)
		ops = len(prog.Ops)
	}
	writeOK(w, map[string]any{"ok": ok, "issues": issues, "compile": cerrs, "unpaired": unpaired, "opcodes": ops})
}

func (s *Server) deployMacro(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := s.db.GetMacro(id)
	if err != nil {
		writeErr(w, "notfound", CodeNotFound, "macro not found")
		return
	}
	issues := engine.ValidateGraph(m)
	if len(issues) > 0 {
		writeErr(w, "bad", CodeValidation, issues[0].Message)
		return
	}
	m.Deployed = true
	m.Enabled = true
	if err := s.db.UpsertMacro(m); err != nil {
		writeErr(w, "int", CodeInternal, err.Error())
		return
	}
	if err := s.reloadTriggers(); err != nil {
		writeErr(w, "conflict", CodeConflict, err.Error())
		return
	}
	s.db.Audit("macro.deploy", id)
	writeOK(w, m)
}

func (s *Server) runHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := s.runMacro(id, "http")
	if err != nil {
		writeErr(w, "bad", CodeValidation, err.Error())
		return
	}
	writeOK(w, res)
}

func (s *Server) runMacro(id, via string) (engine.Result, error) {
	s.mu.Lock()
	en := s.global
	s.mu.Unlock()
	if !en {
		return engine.Result{Status: "stopped", Reason: "global_disabled"}, nil
	}
	m, err := s.db.GetMacro(id)
	if err != nil {
		return engine.Result{}, err
	}
	if issues := engine.ValidateGraph(m); len(issues) > 0 {
		return engine.Result{}, issues[0]
	}
	prog, cerrs := engine.Compile(m)
	if len(cerrs) > 0 {
		return engine.Result{}, cerrs[0]
	}
	runID := engine.NewRunID()
	_ = s.db.SaveRun(runID, id, "queued", map[string]string{"via": via})
	ctx, cancel := context.WithCancel(context.Background())
	s.reg.Begin(runID, cancel, engine.RunRecord{Result: engine.Result{RunID: runID, MacroID: id, Status: "queued"}, MacroName: m.Name})
	_ = s.db.SaveRun(runID, id, "running", map[string]string{"via": via})
	s.safety.Reset()
	ex := engine.NewExecutor(s.stack, s.pacer, s.safety)
	res := ex.Run(ctx, runID, id, prog)
	res.StartedAt = clock.Format(clock.Now())
	res.EndedAt = clock.Format(clock.Now())
	s.reg.Finish(res)
	_ = s.db.SaveRun(runID, id, res.Status, res)
	s.db.Audit("macro.run", id+" "+res.Status+" via "+via)
	cancel()
	return res, nil
}

func (s *Server) stopOne(w http.ResponseWriter, r *http.Request) {
	s.safety.EmergencyStop()
	writeOK(w, map[string]any{"stopped": true})
}

func (s *Server) emergency(w http.ResponseWriter, r *http.Request) {
	s.safety.EmergencyStop()
	s.reg.CancelAll()
	s.db.Audit("emergency-stop", "ui")
	writeOK(w, map[string]any{"stopped": true})
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.reg.List())
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.reg.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, "notfound", CodeNotFound, "run not found")
		return
	}
	writeOK(w, rec)
}

func (s *Server) getTrace(w http.ResponseWriter, r *http.Request) {
	rec, ok := s.reg.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, "notfound", CodeNotFound, "run not found")
		return
	}
	writeOK(w, rec.Trace)
}

func (s *Server) benchmark(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Strategy string `json:"strategy"`
		TargetUs int    `json:"target_us"`
		Samples  int    `json:"samples"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.TargetUs <= 0 {
		body.TargetUs = 1000
	}
	if body.Samples <= 0 {
		body.Samples = 80
	}
	st := timing.Balanced
	switch body.Strategy {
	case "realtime":
		st = timing.Realtime
	case "efficient":
		st = timing.Efficient
	}
	res := timing.RunBench(s.pacer, st, time.Duration(body.TargetUs)*time.Microsecond, body.Samples)
	writeOK(w, res)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{"events": s.ring.Tail(400), "dropped": s.ring.Dropped()})
}

func (s *Server) clearEvents(w http.ResponseWriter, r *http.Request) {
	s.ring.Clear()
	writeOK(w, map[string]any{"cleared": true})
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"mask_printable":     s.hub.Mask(),
		"capture_authorized": s.hub.Authorized(),
		"global_enabled":     s.global,
		"emergency_hotkey":   s.cfg.EmergencyHotkey,
		"device_mode":        s.cfg.DeviceMode,
	})
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MaskPrintable *bool `json:"mask_printable"`
		GlobalEnabled *bool `json:"global_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, "bad", CodeValidation, "invalid json")
		return
	}
	if body.MaskPrintable != nil {
		s.hub.SetMask(*body.MaskPrintable)
	}
	if body.GlobalEnabled != nil {
		s.mu.Lock()
		s.global = *body.GlobalEnabled
		s.mu.Unlock()
		s.trig.SetEnabled(*body.GlobalEnabled)
	}
	s.getSettings(w, r)
}

func (s *Server) authorizeCapture(w http.ResponseWriter, r *http.Request) {
	s.hub.SetAuth(true)
	s.db.Audit("capture.authorize", "explicit")
	writeOK(w, map[string]any{"authorized": true})
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Usage uint16 `json:"usage"`
		Name  string `json:"name"`
	}
	var keys []row
	for _, u := range hid.KnownKeyboardUsages() {
		keys = append(keys, row{u, hid.UsageName(u)})
	}
	writeOK(w, keys)
}

func (s *Server) reloadTriggers() error {
	ms, err := s.db.ListMacros()
	if err != nil {
		return err
	}
	return s.trig.Replace(ms)
}

func (s *Server) Recover() {
	ids, _ := s.db.IncompleteRuns()
	for _, id := range ids {
		_ = s.db.MarkAbandoned(id)
	}
	if len(ids) > 0 {
		logger.Log().Info("recovered incomplete runs as abandoned", "count", len(ids))
	}
	_ = s.reloadTriggers()
}

func (s *Server) Pump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-s.stack.Bus():
			if !ok {
				return
			}
			s.hub.Ingest(ev)
			s.trig.OnEvent(ev)
			if isEmergency(ev, s.cfg.EmergencyHotkey) {
				s.safety.EmergencyStop()
				s.reg.CancelAll()
			}
		}
	}
}

func isEmergency(ev device.Envelope, combo string) bool {
	if ev.Kind != "key" || ev.Value == 0 {
		return false
	}
	// cheap last-key check: Escape while we rely on trigger manager for full combo
	return ev.Name == "Escape" && strings.Contains(combo, "Escape")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // real check happens in wrap()
}

func (s *Server) wsEvents(w http.ResponseWriter, r *http.Request) {
	if !s.checkWSAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	cl := s.hub.Subscribe()
	defer s.hub.Unsubscribe(cl)
	for batch := range cl.C() {
		if err := c.WriteJSON(map[string]any{"events": batch}); err != nil {
			return
		}
	}
}

func (s *Server) wsExec(w http.ResponseWriter, r *http.Request) {
	if !s.checkWSAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			return
		}
		s.safety.Heartbeat()
	}
}

func (s *Server) checkWSAuth(r *http.Request) bool {
	tok := r.URL.Query().Get("token")
	if tok == "" {
		tok = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	return subtle.ConstantTimeCompare([]byte(tok), []byte(s.cfg.AuthToken)) == 1
}
