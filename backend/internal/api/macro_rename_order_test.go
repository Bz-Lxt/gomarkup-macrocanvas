package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macrocanvas/macrocanvas/internal/api"
	"github.com/macrocanvas/macrocanvas/internal/capture"
	"github.com/macrocanvas/macrocanvas/internal/config"
	"github.com/macrocanvas/macrocanvas/internal/device"
	"github.com/macrocanvas/macrocanvas/internal/macro"
	"github.com/macrocanvas/macrocanvas/internal/storage"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

func TestMacroRenameRefreshesListOrdering(t *testing.T) {
	cfg := config.Config{
		AuthToken:      "test-token",
		DeviceMode:     "mock",
		MaxMacroIters:  1000,
		MaxWallClockMs: 1000,
	}
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ring := capture.NewRing(16)
	hub := capture.NewHub(ring, true)
	stack := device.NewStack(cfg)
	cal := timing.Calibration{}
	handler := api.New(cfg, stack, db, hub, ring, timing.NewPacer(cal), cal).Handler()

	alpha := macro.Macro{ID: "a", Name: "Alpha", Nodes: []macro.Node{}, Edges: []macro.Edge{}}
	beta := macro.Macro{ID: "b", Name: "Beta", Nodes: []macro.Node{}, Edges: []macro.Edge{}}
	callMacroAPI(t, handler, cfg.AuthToken, http.MethodPost, "/api/v1/macros", alpha)
	callMacroAPI(t, handler, cfg.AuthToken, http.MethodPost, "/api/v1/macros", beta)

	alpha.Name = "Zulu"
	callMacroAPI(t, handler, cfg.AuthToken, http.MethodPut, "/api/v1/macros/a", alpha)

	var detail macro.Macro
	if err := json.Unmarshal(callMacroAPI(t, handler, cfg.AuthToken, http.MethodGet, "/api/v1/macros/a", nil), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Name != "Zulu" {
		t.Fatalf("detail name = %q, want Zulu", detail.Name)
	}

	var list []macro.Macro
	if err := json.Unmarshal(callMacroAPI(t, handler, cfg.AuthToken, http.MethodGet, "/api/v1/macros", nil), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list length = %d, want 2", len(list))
	}
	if list[0].Name != "Beta" || list[1].Name != "Zulu" {
		t.Fatalf("list order = [%q, %q], want [Beta, Zulu]", list[0].Name, list[1].Name)
	}
}

func callMacroAPI(t *testing.T, handler http.Handler, token, method, path string, payload any) json.RawMessage {
	t.Helper()
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s: status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	var env api.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK {
		t.Fatalf("%s %s: API error=%+v", method, path, env.Error)
	}
	return env.Data
}
