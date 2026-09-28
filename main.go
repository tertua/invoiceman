package main

import (
	"context"
	"errors"
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

	_ "net/http/pprof" // #nosec G108 -- localhost-only diagnostics (see startDebugListener)
)

// @title Invoiceman API
// @version 1.0
// @description Invoiceman API. Two prefixes serve the same routes: /api/v1
// @description (current, used by the MPA) and /api (legacy, deprecated —
// @description responses carry a Sunset header). Unless tagged otherwise,
// @description endpoints speak JSON with this envelope. Success: the data
// @description keys directly, e.g. {"expense": {...}} or
// @description {"invoices": [...], "meta": {"page": 1, "per_page": 20, "total": 42}}.
// @description List endpoints accept ?page (default 1) and ?per_page
// @description (default 20, max 100) and always return a meta object.
// @description Errors: {"error": {"message": "...", "details": {...}}} with a
// @description stable English message (translated client-side via i18n api.*
// @description keys). Logout answers 204 with no body; most DELETEs do too,
// @description except DELETE /payments/:id which voids and answers 200.
// @description
// @description Auth — three independent schemes, never mixed:
// @description 1. SessionCookie (most endpoints): login/register set
// @description HttpOnly access_token + refresh_token cookies. Send cookies
// @description with every request (fetch: credentials:include). An expired
// @description access token is refreshed transparently from the refresh
// @description cookie. A Bearer access token in the Authorization header
// @description works as a fallback where cookies are unavailable.
// @description 2. CSRF double-submit (session-cookie POST/PATCH/DELETE only):
// @description login/register also set a readable csrf_token cookie — echo
// @description it back as the X-CSRF-Token header or the mutation is
// @description rejected with 403. Public GETs and API-key calls are exempt.
// @description 3. ApiKeyAuth (service relay /gateway/* only): pass the
// @description project API key as the Authorization header; cookie sessions
// @description are never accepted there.
// @description Mutations that create money movement accept an Idempotency-Key
// @description header (one UUID per intent); replays return the original
// @description result instead of duplicating. Login/register/forgot-password
// @description are behind Cloudflare Turnstile when a secret is configured
// @description (captcha token required) and rate-limited; AI endpoints answer
// @description 501 without GEMINI_API_KEY.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @license.name GNU GPLv3
// @license.url https://www.gnu.org/licenses/gpl-3.0.html
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey SessionCookie
// @in cookie
// @name access_token
func main() {
	os.Exit(run())
}

// run holds the startup sequence so deferred cleanup (worker.Stop) always
// runs; os.Exit stays in main where no defers are pending.
func run() int {
	// Load and validate configuration first: fail fast on bad env
	// (wrong DSN, default secrets in prod, invalid ports) before binding.
	cfg, err := configs.Load()
	if err != nil {
		logger.Init("dev", "error")
		logger.L().Error("invalid configuration", "err", err)
		return 1
	}

	// Structured logging: text in dev, JSON in prod.
	logger.Init(cfg.Stage, cfg.Log.Level)

	// Docker HEALTHCHECK (scratch image has no wget/curl): exit 0 when
	// /healthz answers 200, exit 1 otherwise.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		return healthcheck(cfg.Server.Port)
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
	middleware.FiberMiddleware(app)      // Register Fiber's middleware for app.
	app.Use(routes.DeprecationHeaders()) // Mark legacy /api responses.

	// Migrate database schema (SQLite file is auto-created on first run).
	if err := database.Migrate(); err != nil {
		logger.L().Error("failed to migrate database", "err", err)
		return 1
	}

	// Register payment gateways (add new providers here, e.g. crypto).
	gateway.Register(midtrans.Gateway{})
	gateway.Register(nowpayments.Gateway{})

	// Background worker: async mail + webhook retries + idempotency purge.
	// Stopped (with drain) after the HTTP server shuts down.
	worker := outbox.New()
	worker.Start(context.Background())
	defer worker.Stop()

	// Localhost-only diagnostics (pprof). Off unless DEBUG_PORT is set;
	// never exposed publicly.
	startDebugListener(cfg.Debug.Port)

	// Routes.
	routes.HealthRoutes(app)  // Liveness/readiness probes (public, before auth).
	routes.MetricsRoutes(app) // Prometheus-compatible scrape endpoint.
	if cfg.Storage.Backend == "local" {
		// Public logos only (receipts stay behind the auth proxy).
		if err := routes.MountUploads(app, cfg.Storage.Dir); err != nil {
			logger.L().Error("failed to mount uploads", "err", err)
			return 1
		}
	}
	routes.SwaggerRoute(app)                        // Register a route for API Docs (Swagger).
	routes.RegisterAPI(app, routes.APIV1Prefix)     // Current prefix first (see versioning.go ordering).
	routes.RegisterAPI(app, routes.APILegacyPrefix) // Legacy prefix (deprecation headers).
	routes.MountWebUI(app, cfg.WebUI.Dir)           // Optional embedded web UI (empty = API-only).
	routes.NotFoundRoute(app)                       // Register route for 404 Error.

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
	return 0
}

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
