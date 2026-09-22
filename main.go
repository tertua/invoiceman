package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/middleware"
	"github.com/tertua/invoiceman/pkg/routes"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/gateway"
	"github.com/tertua/invoiceman/platform/midtrans"
	"github.com/tertua/invoiceman/platform/nowpayments"

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
	// Docker HEALTHCHECK (scratch image has no wget/curl): exit 0 when
	// /healthz answers 200, exit 1 otherwise.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}

	// Define Fiber config.
	config := configs.FiberConfig()

	// Define a new Fiber app with config.
	app := fiber.New(config)

	// Middlewares.
	middleware.FiberMiddleware(app) // Register Fiber's middleware for app.

	// Migrate database schema (SQLite file is auto-created on first run).
	if err := database.Migrate(); err != nil {
		log.Fatal("failed to migrate database: ", err)
	}

	// Register payment gateways (add new providers here, e.g. crypto).
	gateway.Register(midtrans.Gateway{})
	gateway.Register(nowpayments.Gateway{})

	// Routes.
	routes.HealthRoutes(app)  // Liveness/readiness probes (public, before auth).
	routes.SwaggerRoute(app)  // Register a route for API Docs (Swagger).
	routes.PublicRoutes(app)  // Register a public routes for app.
	routes.GatewayRoutes(app) // Register service relay routes (API key, before sessions).
	routes.PrivateRoutes(app) // Register a private routes for app.
	routes.NotFoundRoute(app) // Register route for 404 Error.

	// Start server (with or without graceful shutdown).
	if os.Getenv("STAGE_STATUS") == "dev" {
		utils.StartServer(app)
	} else {
		utils.StartServerWithGracefulShutdown(app)
	}
}

// healthcheck probes /healthz on the local port for Docker HEALTHCHECK.
func healthcheck() int {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "5000"
	}
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
