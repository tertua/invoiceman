package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/metrics"
	"github.com/tertua/invoiceman/pkg/middleware"
	"github.com/tertua/invoiceman/pkg/routes"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
	"github.com/tertua/invoiceman/platform/nowpayments"
	"github.com/tertua/invoiceman/platform/outbox"

	"github.com/gofiber/fiber/v3"

	_ "github.com/tertua/invoiceman/docs" // load API Docs files (Swagger)

	_ "github.com/joho/godotenv/autoload" // load .env file automatically
)

// @title API
// @version 1.0
// @description This is an auto-generated API Docs.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// Load and validate configuration first: fail fast on bad env
	// (wrong DSN, default secrets in prod, invalid ports) before binding.
	cfg, err := configs.Load()
	if err != nil {
		logger.Init("dev", "error")
		logger.L().Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	// Structured logging: text in dev, JSON in prod.
	logger.Init(cfg.Stage, cfg.Log.Level)

	// Docker HEALTHCHECK (scratch image has no wget/curl): exit 0 when
	// /healthz answers 200, exit 1 otherwise.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck(cfg.Server.Port))
	}

	// Metrics registry with build version and DB pool gauges.
	reg := metrics.Init(appVersion())
	reg.SetDBStats(func() (open, idle, inUse int, backend string, ok bool) {
		stats, ok := database.Stats()
		if !ok {
			return 0, 0, 0, "", false
		}
		return stats.Open, stats.Idle, stats.InUse, stats.Backend, true
	})

	// Define Fiber config.
	fiberConfig := configs.FiberConfig()

	// Define a new Fiber app with config.
	app := fiber.New(fiberConfig)

	// Middlewares.
	middleware.FiberMiddleware(app) // Register Fiber's middleware for app.

	// Migrate database schema (SQLite file is auto-created on first run).
	if err := database.Migrate(); err != nil {
		logger.L().Error("failed to migrate database", "err", err)
		os.Exit(1)
	}

	// Register payment gateways (add new providers here, e.g. crypto).
	gateway.Register(midtrans.Gateway{})
	gateway.Register(nowpayments.Gateway{})

	// Background worker: async mail + webhook retries + idempotency purge.
	// Stopped (with drain) after the HTTP server shuts down.
	worker := outbox.New()
	worker.Start(context.Background())
	defer worker.Stop()

	// Routes.
	routes.HealthRoutes(app)  // Liveness/readiness probes (public, before auth).
	routes.MetricsRoutes(app) // Prometheus-compatible scrape endpoint.
	routes.SwaggerRoute(app)  // Register a route for API Docs (Swagger).
	routes.PublicRoutes(app)  // Register a public routes for app.
	routes.GatewayRoutes(app) // Register service relay routes (API key, before sessions).
	routes.PrivateRoutes(app) // Register a private routes for app.
	routes.NotFoundRoute(app) // Register route for 404 Error.

	// One-line startup summary (secrets never logged).
	logger.L().Info("starting server",
		"version", appVersion(),
		"stage", cfg.Stage,
		"addr", cfg.ListenAddr(),
		"db", database.Backend(),
		"redis", cfg.Redis.Enabled(),
		"metrics", cfg.Metrics.Enabled,
	)

	// Start server (with or without graceful shutdown).
	if cfg.IsDev() {
		utils.StartServer(app)
	} else {
		utils.StartServerWithGracefulShutdown(app)
	}
}

// healthcheck probes /healthz on the local port for Docker HEALTHCHECK.
func healthcheck(port string) int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
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
