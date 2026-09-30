package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tertua/tupay/pkg/logger"
)

// healthcheck probes /healthz on the local port for Docker HEALTHCHECK.
func healthcheck(port string) int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

// startDebugListener serves net/http/pprof on 127.0.0.1 only.
// Empty port disables it (default).
func startDebugListener(port string) {
	if port == "" {
		return
	}
	go func() {
		// Bound to loopback deliberately: profiles can leak internals.
		// ReadHeaderTimeout guards the header-read phase (Slowloris).
		srv := &http.Server{Addr: "127.0.0.1:" + port, ReadHeaderTimeout: 5 * time.Second}
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.L().Warn("debug listener stopped", "err", err)
		}
	}()
	logger.L().Info("debug listener on 127.0.0.1:" + port)
}

// appVersion reads the single-source VERSION file, falling back to "dev".
func appVersion() string {
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	if v := strings.TrimSpace(string(raw)); v != "" {
		return v
	}
	return "dev"
}
