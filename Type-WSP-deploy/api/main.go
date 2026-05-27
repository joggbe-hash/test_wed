package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// M 是 API 回應常用的 JSON map，讓 handler 可以快速組出簡單 response。
type M map[string]any

// writeJSON 統一設定 JSON header、HTTP status，並把資料序列化回前端。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// readJSON 把 request body 解析到指定 struct，供登入、註冊、純文字發文使用。
func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func main() {
	// 啟動順序：先讀環境設定，再初始化 DB、Redis、MinIO、Session。
	cfg := LoadConfig()

	InitDB(cfg)
	defer CloseDB()

	InitRedis(cfg)
	InitMinio(cfg)
	InitSession(cfg)

	mux := http.NewServeMux()

	// Docker healthcheck 與人工檢查都走這個端點。
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, M{"status": "ok"})
	})

	// Auth 端點：送驗證碼、註冊、登入、登出。
	mux.HandleFunc("POST /api/auth/send-code", handleSendCode)
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("POST /api/auth/logout", handleLogout)

	// Feed/Post 端點需要登入，requireAuth 會把 session user 放進 request context。
	mux.HandleFunc("GET /api/feed", requireAuth(handleFeed))
	mux.HandleFunc("POST /api/posts", requireAuth(handleCreatePost))
	mux.HandleFunc("DELETE /api/posts/{id}", requireAuth(handleDeletePost))

	// Worker 圖片處理完成後透過 Redis Pub/Sub 通知，API 再轉成 WebSocket 推給前端。
	mux.HandleFunc("GET /api/ws/", handleWebSocket)

	// 圖片不直接公開 MinIO，由 API 代讀並回傳，方便之後補權限檢查。
	mux.HandleFunc("GET /api/images/{key...}", handleGetImage)

	log.Println("API server listening on :5000")
	if err := http.ListenAndServe(":5000", mux); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
