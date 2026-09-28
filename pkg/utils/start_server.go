package utils

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/logger"
)

// shutdownTimeout bounds graceful shutdown so containers (SIGTERM from
// Docker/K8s) never hang forever on draining connections.
const shutdownTimeout = 10 * time.Second

// StartServerWithGracefulShutdown function for starting server with a graceful shutdown.
func StartServerWithGracefulShutdown(a *fiber.App) {
	// Create channel for idle connections.
	idleConnsClosed := make(chan struct{})

	go func() {
		signals := make(chan os.Signal, 1)
		// SIGINT for local Ctrl+C, SIGTERM for Docker/K8s stop.
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals

		// Received a shutdown signal, drain with a bounded timeout.
		if err := a.ShutdownWithTimeout(shutdownTimeout); err != nil {
			// Error from closing listeners, or context timeout:
			logger.L().Error("server is not shutting down", "err", err)
		}

		close(idleConnsClosed)
	}()

	// Build Fiber connection URL.
	fiberConnURL, _ := ConnectionURLBuilder("fiber")

	// Run server.
	if err := a.Listen(fiberConnURL); err != nil {
		logger.L().Error("server is not running", "err", err)
	}

	<-idleConnsClosed
}

// StartServer func for starting a simple server.
func StartServer(a *fiber.App) {
	// Build Fiber connection URL.
	fiberConnURL, _ := ConnectionURLBuilder("fiber")

	// Run server.
	if err := a.Listen(fiberConnURL); err != nil {
		logger.L().Error("server is not running", "err", err)
	}
}
