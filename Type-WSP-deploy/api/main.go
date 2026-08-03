package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type M map[string]any

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, value any) error {
	return json.NewDecoder(r.Body).Decode(value)
}

func newMux() *http.ServeMux {
	return newMuxWithDependencies(LoadSession, checkReadiness)
}

func newMuxWithSessionLoader(loadSession sessionLoader) *http.ServeMux {
	return newMuxWithDependencies(loadSession, checkReadiness)
}

func newMuxWithDependencies(loadSession sessionLoader, ready readinessChecker) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handleLiveness)
	mux.HandleFunc("GET /health/ready", readinessHandler(ready))
	// Compatibility alias for clients using the original endpoint.
	mux.HandleFunc("GET /api/health", handleLiveness)

	mux.HandleFunc("POST /api/auth/send-code", handleSendCode)
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/session", requireAuthWithLoader(handleCurrentSession, loadSession))
	mux.HandleFunc("POST /api/auth/logout", handleLogout)

	mux.HandleFunc("GET /api/feed", requireAuthWithLoader(handleFeed, loadSession))
	mux.HandleFunc("POST /api/posts", requireAuthWithLoader(handleCreatePost, loadSession))
	mux.HandleFunc("DELETE /api/posts/{id}", requireAuthWithLoader(handleDeletePost, loadSession))
	mux.HandleFunc("GET /api/ws/", handleWebSocket)
	mux.HandleFunc("GET /api/images/{key...}", requireAuthWithLoader(handleGetImage, loadSession))
	return mux
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           requestLogger(logger, newMux()),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	slog.Info("api_server_starting", "service", "api", "listen_addr", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
