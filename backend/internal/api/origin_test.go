package api

import (
	"net/http"
	"testing"
)

func TestOriginSameHost(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://localhost:31821/ws/events", nil)
	r.Host = "localhost:31821"
	r.Header.Set("Origin", "http://localhost:31821")
	if !originAllowed(r, nil, "localhost:31821") {
		t.Fatal("same origin denied")
	}
}

func TestOriginDenied(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://backend/ws/events", nil)
	r.Host = "backend:8080"
	r.Header.Set("Origin", "http://evil.example")
	if originAllowed(r, nil, "localhost:31821") {
		t.Fatal("evil origin allowed")
	}
}

func TestOriginEmptyNonBrowser(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://backend/ws/events", nil)
	r.Host = "backend:8080"
	if !originAllowed(r, nil, "") {
		t.Fatal("empty origin should pass (non-browser)")
	}
}
