package main

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginAllowsMatchingHost(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("X-Forwarded-Proto", "https")

	if !sameOrigin(req) {
		t.Fatal("expected same origin request to be allowed")
	}
}

func TestSameOriginRejectsDifferentHost(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "https://evil.test")
	req.Header.Set("X-Forwarded-Proto", "https")

	if sameOrigin(req) {
		t.Fatal("expected cross origin request to be rejected")
	}
}

func TestSameOriginRejectsDifferentPort(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "https://example.test:444")
	req.Header.Set("X-Forwarded-Proto", "https")

	if sameOrigin(req) {
		t.Fatal("expected different origin port to be rejected")
	}
}

func TestSameOriginRejectsDifferentScheme(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("X-Forwarded-Proto", "https")

	if sameOrigin(req) {
		t.Fatal("expected different origin scheme to be rejected")
	}
}
