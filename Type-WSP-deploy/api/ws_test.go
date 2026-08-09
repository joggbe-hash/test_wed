package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

func TestConsumeWebSocketMessagesAllowsSmallMessageAndRejectsOversizedMessage(t *testing.T) {
	received := make(chan string, 1)
	readResult := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			readResult <- err
			return
		}
		defer conn.Close()
		readResult <- consumeWebSocketMessages(conn, func(message []byte) {
			received <- string(message)
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial test WebSocket: %v", err)
	}
	defer client.Close()

	if err := client.WriteMessage(websocket.TextMessage, []byte("ok")); err != nil {
		t.Fatalf("write legitimate message: %v", err)
	}
	select {
	case message := <-received:
		if message != "ok" {
			t.Fatalf("received message = %q", message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("legitimate message was not accepted")
	}

	oversized := bytes.Repeat([]byte("x"), maxClientWebSocketMessageBytes+1)
	if err := client.WriteMessage(websocket.BinaryMessage, oversized); err != nil {
		t.Fatalf("write oversized message: %v", err)
	}
	select {
	case err := <-readResult:
		if err == nil || !strings.Contains(err.Error(), "read limit") {
			t.Fatalf("expected WebSocket read-limit error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("oversized message was not rejected")
	}
}

func TestWebSocketSubscribesToSessionRevocation(t *testing.T) {
	channels := webSocketChannels(7, "session-id")
	if len(channels) != 2 {
		t.Fatalf("channel count = %d, want 2", len(channels))
	}
	revocationChannel := sessionRevocationChannel("session-id")
	if channels[1] != revocationChannel {
		t.Fatalf("revocation channel = %q, want %q", channels[1], revocationChannel)
	}
	if !shouldCloseWebSocketForMessage(revocationChannel, revocationChannel) {
		t.Fatal("session revocation message did not close the WebSocket")
	}
	if shouldCloseWebSocketForMessage(channels[0], revocationChannel) {
		t.Fatal("ordinary user notification closed the WebSocket")
	}
}
