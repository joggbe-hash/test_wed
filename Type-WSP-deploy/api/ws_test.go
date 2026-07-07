package main

import (
	"net/http/httptest"
	"testing"
)

func TestSameOriginAllowsMatchingHost(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "https://example.test")

	if !sameOrigin(req) {
		t.Fatal("expected same origin request to be allowed")
	}
}

func TestSameOriginRejectsDifferentHost(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.test/api/ws/", nil)
	req.Header.Set("Origin", "https://evil.test")

	if sameOrigin(req) {
		t.Fatal("expected cross origin request to be rejected")
	}
}
