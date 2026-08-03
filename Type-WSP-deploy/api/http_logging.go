package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (writer *loggingResponseWriter) WriteHeader(status int) {
	if writer.status != 0 {
		return
	}
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *loggingResponseWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	written, err := writer.ResponseWriter.Write(body)
	writer.bytes += written
	return written, err
}

func (writer *loggingResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := writer.ResponseWriter.(io.ReaderFrom); ok {
		written, err := readerFrom.ReadFrom(reader)
		writer.bytes += int(written)
		return written, err
	}
	return io.Copy(struct{ io.Writer }{writer}, reader)
}

func (writer *loggingResponseWriter) Flush() {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (writer *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := writer.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	connection, buffer, err := hijacker.Hijack()
	if err == nil && writer.status == 0 {
		writer.status = http.StatusSwitchingProtocols
	}
	return connection, buffer, err
}

func (writer *loggingResponseWriter) Push(target string, options *http.PushOptions) error {
	if pusher, ok := writer.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, options)
	}
	return http.ErrNotSupported
}

func (writer *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := uuid.NewString()
		w.Header().Set(requestIDHeader, requestID)
		r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID))
		writer := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		if writer.status == 0 {
			writer.status = http.StatusOK
		}
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		logger.InfoContext(r.Context(), "http_request",
			slog.String("service", "api"),
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("route", route),
			slog.Int("status", writer.status),
			slog.Int("response_bytes", writer.bytes),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
	})
}
