package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func originAllowed(r *http.Request, extra []string, publicHost string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	ohost := u.Host
	reqHost := r.Host
	if ohost == reqHost {
		return true
	}
	if publicHost != "" && ohost == publicHost {
		return true
	}
	oh := hostOnly(ohost)
	rh := hostOnly(reqHost)
	if isLocal(oh) && isLocal(rh) {
		return true
	}
	if isLocal(oh) && (rh == "frontend-user" || rh == "backend") {
		return true
	}
	for _, w := range extra {
		if strings.EqualFold(strings.TrimSpace(w), origin) || strings.EqualFold(strings.TrimSpace(w), ohost) {
			return true
		}
	}
	return false
}

func hostOnly(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	return h
}

func isLocal(h string) bool {
	h = strings.ToLower(h)
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}
