package main

import (
	"net"
	"net/http"
	"strings"
)

const browserRequestHeader = "X-Type-WSP-Request"

// requireBrowserRequest rejects browser-simple cross-site requests. The frontend
// supplies this non-simple header, so a cross-origin browser must pass CORS
// preflight before it can reach a session-changing handler.
func requireBrowserRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(browserRequestHeader) != "1" {
			writeJSON(w, http.StatusForbidden, M{"error": "untrusted request origin"})
			return
		}
		next(w, r)
	}
}

func requestClientIdentity(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(forwarded) != nil {
		return forwarded
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if direct := strings.TrimSpace(r.RemoteAddr); net.ParseIP(direct) != nil {
		return direct
	}
	return "unknown"
}
