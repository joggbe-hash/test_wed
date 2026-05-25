package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// M 是 JSON 回應的簡寫型別
type M map[string]any

// writeJSON 將資料序列化為 JSON 並寫入 HTTP 回應
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// readJSON 從 HTTP 請求 body 反序列化 JSON
func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func main() {
	// ===== 初始化設定與外部服務連線 =====
	cfg := LoadConfig()

	InitDB(cfg)
	defer CloseDB()

	InitRedis(cfg)
	InitMinio(cfg)
	InitSession(cfg)

	// ===== 設定路由 =====
	mux := http.NewServeMux()

	// 健康檢查（供 Docker 和負載均衡器使用）
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, M{"status": "ok"})
	})

	// 認證相關路由（不需登入）
	mux.HandleFunc("POST /api/auth/send-code", handleSendCode)
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("POST /api/auth/logout", handleLogout)

	// 貼文相關路由（需要登入，由 requireAuth 中介層保護）
	mux.HandleFunc("GET /api/feed", requireAuth(handleFeed))
	mux.HandleFunc("POST /api/posts", requireAuth(handleCreatePost))

	// WebSocket 即時通知（Worker 處理完圖片後推送給前端）
	mux.HandleFunc("GET /api/ws/", handleWebSocket)

	// 圖片代理（公開存取，不需登入）
	mux.HandleFunc("GET /api/images/{key...}", handleGetImage)

	// ===== 啟動 HTTP 伺服器 =====
	log.Println("API server 啟動於 :5000")
	if err := http.ListenAndServe(":5000", mux); err != nil {
		log.Fatalf("伺服器啟動失敗: %v", err)
	}
}
