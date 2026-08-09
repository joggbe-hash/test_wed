package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"typewsp/shared/contracts"
)

const maxClientWebSocketMessageBytes = 1024

const webSocketSessionCheckInterval = 30 * time.Second

// upgrader 把 HTTP 連線升級成 WebSocket；第一版先放寬 origin，正式版應改成白名單。
var upgrader = websocket.Upgrader{
	CheckOrigin: sameOrigin,
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil || originURL.Host == "" {
		return false
	}

	expectedScheme := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if commaIndex := strings.IndexByte(expectedScheme, ','); commaIndex >= 0 {
		expectedScheme = strings.TrimSpace(expectedScheme[:commaIndex])
	}
	if expectedScheme == "" {
		if r.TLS != nil {
			expectedScheme = "https"
		} else {
			expectedScheme = "http"
		}
	}
	if originURL.Scheme != expectedScheme {
		return false
	}

	originHost, err := normalizedHostPort(originURL.Host, originURL.Scheme)
	if err != nil {
		return false
	}
	requestHost, err := normalizedHostPort(r.Host, expectedScheme)
	if err != nil {
		return false
	}
	return originHost == requestHost
}

func normalizedHostPort(host, scheme string) (string, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	if h, port, err := net.SplitHostPort(host); err == nil {
		return net.JoinHostPort(h, port), nil
	}

	defaultPort := "80"
	if scheme == "https" {
		defaultPort = "443"
	} else if scheme != "http" {
		return "", fmt.Errorf("unsupported origin scheme %q", scheme)
	}
	return net.JoinHostPort(strings.Trim(host, "[]"), defaultPort), nil
}

// handleWebSocket 驗證 session 後訂閱該使用者的 Redis channel。
// Worker 完成圖片處理時 publish，API 再把通知轉給前端。
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := LoadSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade WebSocket failed: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sid, err := verifySID(cookie.Value)
	if err != nil {
		return
	}
	channels := webSocketChannels(user.ID, sid)
	revocationChannel := sessionRevocationChannel(sid)
	sub := rdb.Subscribe(ctx, channels...)
	defer sub.Close()
	subscribeCtx, cancelSubscribe := context.WithTimeout(ctx, 5*time.Second)
	_, subscribeErr := sub.Receive(subscribeCtx)
	cancelSubscribe()
	if subscribeErr != nil {
		return
	}
	// Close the race between the initial authentication check and the active
	// revocation subscription. A logout in that window deletes this session.
	if _, err := LoadSession(ctx, cookie.Value); err != nil {
		return
	}
	ticker := time.NewTicker(webSocketSessionCheckInterval)
	defer ticker.Stop()

	go func() {
		_ = consumeWebSocketMessages(conn, nil)
		cancel()
	}()

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if shouldCloseWebSocketForMessage(msg.Channel, revocationChannel) {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				return
			}
		case <-ticker.C:
			if _, err := LoadSession(ctx, cookie.Value); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func webSocketChannels(userID int, sid string) []string {
	return []string{
		contracts.NotifyUserChannel(userID),
		sessionRevocationChannel(sid),
	}
}

func shouldCloseWebSocketForMessage(channel string, revocationChannels ...string) bool {
	for _, revocationChannel := range revocationChannels {
		if channel == revocationChannel {
			return true
		}
	}
	return false
}

func consumeWebSocketMessages(conn *websocket.Conn, onMessage func([]byte)) error {
	conn.SetReadLimit(maxClientWebSocketMessageBytes)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if onMessage != nil {
			onMessage(message)
		}
	}
}
