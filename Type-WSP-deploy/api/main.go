package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
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

func newMux() *http.ServeMux {
	return newMuxWithSessionLoader(LoadSession)
}

func newMuxWithSessionLoader(loadSession sessionLoader) *http.ServeMux {
	mux := http.NewServeMux()

	// Docker healthcheck 與人工檢查都走這個端點。
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, M{"status": "ok"})
	})

	// Auth 端點：送驗證碼、註冊、登入、恢復 session、登出。
	mux.HandleFunc("POST /api/auth/send-code", handleSendCode)
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/session", requireAuthWithLoader(handleCurrentSession, loadSession))
	mux.HandleFunc("POST /api/auth/logout", handleLogout)

	// Feed/Post 端點需要登入，requireAuth 會把 session user 放進 request context。
	mux.HandleFunc("GET /api/feed", requireAuthWithLoader(handleFeed, loadSession))
	mux.HandleFunc("POST /api/posts", requireAuthWithLoader(handleCreatePost, loadSession))
	mux.HandleFunc("DELETE /api/posts/{id}", requireAuthWithLoader(handleDeletePost, loadSession))

	// Worker 圖片處理完成後透過 Redis Pub/Sub 通知，API 再轉成 WebSocket 推給前端。
	mux.HandleFunc("GET /api/ws/", handleWebSocket)

	// 圖片不直接公開 MinIO，由 API 驗證 session 與 key prefix 後代讀。
	mux.HandleFunc("GET /api/images/{key...}", requireAuthWithLoader(handleGetImage, loadSession))

	return mux
}

func main() {
	// 啟動順序：先讀環境設定，再初始化 DB、Redis、MinIO、Session。
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load API config failed: %v", err)
	}

	InitDB(cfg)
	defer CloseDB()

	migrationCtx, cancelMigrations := context.WithTimeout(context.Background(), 60*time.Second)
	if err := RunMigrations(migrationCtx); err != nil {
		cancelMigrations()
		log.Fatalf("database migration failed: %v", err)
	}
	cancelMigrations()

	InitRedis(cfg)
	InitMinio(cfg)
	InitSession(cfg)
	InitAuth(cfg)

	mux := newMux()

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("API server listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
