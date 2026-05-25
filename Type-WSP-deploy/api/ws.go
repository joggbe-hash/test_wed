package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader 負責將 HTTP 連線升級為 WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWebSocket 處理 GET /api/ws/
// 驗證 session cookie → 升級為 WebSocket → 訂閱 Redis 通知頻道
// 當 Worker 處理完圖片後，透過 Redis Pub/Sub 發送信號，此處轉發給前端
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 透過 session cookie 驗證使用者身份
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

	// 將 HTTP 連線升級為 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升級失敗: %v", err)
		return
	}
	defer conn.Close()

	// 訂閱該使用者專屬的 Redis 通知頻道
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	channel := fmt.Sprintf("notify:user:%d", user.ID)
	sub := rdb.Subscribe(ctx, channel)
	defer sub.Close()

	// 背景 goroutine：持續讀取 WebSocket 以偵測客戶端斷線
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	// 主迴圈：將 Redis Pub/Sub 的訊息即時轉發到 WebSocket
	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
