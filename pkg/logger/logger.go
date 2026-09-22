package logger

// Package logger is the single structured-logging entrypoint for the backend.
//
// Backend: log/slog from the stdlib (no new dependency). Format follows the
// stage: human-readable text in dev, single-line JSON in prod for aggregators
// (Loki/Docker). Level comes from LOG_LEVEL (debug|info|warn|error).
//
// Usage: logger.L().Info("...", "key", value). Request-scoped correlation
// uses the X-Request-ID header via WithRequestID where available; the Fiber
// access log already carries rid= for every request.

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu      sync.RWMutex
	current *slog.Logger
)

// Init configures the shared logger. Stage "prod" selects JSON output,
// anything else selects text. Unknown levels fall back to info.
func Init(stage, level string) {
	var l slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: l}
	var handler slog.Handler
	if stage == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	mu.Lock()
	current = slog.New(handler)
	mu.Unlock()
}

// L returns the shared logger, lazily initialized with dev/text/info
// defaults when Init was not called (tests, CLIs).
func L() *slog.Logger {
	mu.RLock()
	l := current
	mu.RUnlock()
	if l != nil {
		return l
	}
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		current = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return current
}

// WithRequestID returns a logger carrying the request id for correlation.
// Empty ids return the shared logger unchanged.
func WithRequestID(requestID string) *slog.Logger {
	if strings.TrimSpace(requestID) == "" {
		return L()
	}
	return L().With("request_id", requestID)
}
