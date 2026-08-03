package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type readinessChecker func(context.Context) error

func checkReadiness(ctx context.Context) error {
	var failures []error
	if userPool == nil || systemPool == nil {
		failures = append(failures, errors.New("database pools are not initialized"))
	} else {
		if err := userPool.Ping(ctx); err != nil {
			failures = append(failures, fmt.Errorf("user database: %w", err))
		}
		if err := systemPool.Ping(ctx); err != nil {
			failures = append(failures, fmt.Errorf("system database: %w", err))
		}
	}
	if rdb == nil {
		failures = append(failures, errors.New("redis client is not initialized"))
	} else if err := rdb.Ping(ctx).Err(); err != nil {
		failures = append(failures, fmt.Errorf("redis: %w", err))
	}
	return errors.Join(failures...)
}

func handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, M{"status": "live"})
}

func readinessHandler(check readinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, M{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, M{"status": "ready"})
	}
}
