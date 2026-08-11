package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime"
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
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return fmt.Errorf("Content-Type must be application/json")
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON value")
		}
		return fmt.Errorf("decode trailing request data: %w", err)
	}
	return nil
}

func newMux() *http.ServeMux {
	return newMuxWithDependencies(LoadSessionAndRefresh, checkReadiness)
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

	mux.HandleFunc("POST /api/auth/send-code", requireBrowserRequest(handleSendCode))
	mux.HandleFunc("POST /api/auth/register", requireBrowserRequest(handleRegister))
	mux.HandleFunc("POST /api/auth/login", requireBrowserRequest(handleLogin))
	mux.HandleFunc("POST /api/auth/login/verify", requireBrowserRequest(handleLoginVerification))
	mux.HandleFunc("GET /api/auth/session", requireAuthWithLoader(handleCurrentSession, loadSession))
	mux.HandleFunc("POST /api/auth/logout", requireBrowserRequest(handleLogout))

	mux.HandleFunc("GET /api/feed", requireAuthWithLoader(handleFeed, loadSession))
	mux.HandleFunc("GET /api/posts/me", requireAuthWithLoader(handlePersonalPosts, loadSession))
	mux.HandleFunc("POST /api/posts", requireBrowserRequest(requireAuthWithLoader(handleCreatePost, loadSession)))
	mux.HandleFunc("DELETE /api/posts/{id}", requireBrowserRequest(requireAuthWithLoader(handleDeletePost, loadSession)))
	mux.HandleFunc("GET /api/schedule", requireAuthWithLoader(handleGetSchedule, loadSession))
	mux.HandleFunc("PUT /api/schedule", requireBrowserRequest(requireAuthWithLoader(handlePutSchedule, loadSession)))
	mux.HandleFunc("GET /api/inspirations", requireAuthWithLoader(handleListInspirations, loadSession))
	mux.HandleFunc("POST /api/inspirations", requireBrowserRequest(requireAuthWithLoader(handleCreateInspiration, loadSession)))
	mux.HandleFunc("PATCH /api/inspirations/{id}", requireBrowserRequest(requireAuthWithLoader(handleUpdateInspiration, loadSession)))
	mux.HandleFunc("DELETE /api/inspirations/{id}", requireBrowserRequest(requireAuthWithLoader(handleDeleteInspiration, loadSession)))
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
