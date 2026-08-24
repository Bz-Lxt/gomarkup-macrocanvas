package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macrocanvas/macrocanvas/internal/api"
	"github.com/macrocanvas/macrocanvas/internal/config"
	"github.com/macrocanvas/macrocanvas/internal/storage"
	"github.com/macrocanvas/macrocanvas/internal/timing"
)

func TestDeleteMissingMacroReturnsNotFound(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	const token = "delete-test-token"
	srv := api.New(config.Config{
		AuthToken:       token,
		MaxMacroIters:   100,
		MaxWallClockMs: 1_000,
	}, nil, db, nil, nil, nil, timing.Calibration{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/macros/missing-macro", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing macro status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	var envelope api.Envelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.OK {
		t.Fatalf("DELETE missing macro returned a successful envelope: %s", rec.Body.String())
	}
	if envelope.Error == nil || envelope.Error.Code != api.CodeNotFound {
		t.Fatalf("DELETE missing macro error = %+v, want code %s", envelope.Error, api.CodeNotFound)
	}
}
