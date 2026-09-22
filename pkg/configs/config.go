package configs

// Package configs is the single source of truth for backend configuration.
//
// All values come from the environment (see .env.example) with dev-friendly
// defaults. Call Load once at startup (main.go) and fail fast on validation
// errors; everywhere else use Get, which returns the same defaults-applied
// values without validation so tests and CLIs keep working with partial env.
//
// No new dependency: parsing is plain stdlib so zero-config dev stays intact.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the full backend configuration.
type Config struct {
	Stage   string // STAGE_STATUS: "dev" or "prod"
	AppName string // APP_NAME

	Server ServerConfig
	CORS   CORSConfig
	JWT    JWTConfig
	DB     DBConfig
	Redis  RedisConfig

	RateLimit RateLimitConfig
	AI        AIConfig
	Mail      MailConfig

	Midtrans    MidtransConfig
	NOWPayments NOWPaymentsConfig

	Relay RelayConfig
	Log   LogConfig

	Idempotency IdempotencyConfig

	Outbox OutboxConfig

	Metrics MetricsConfig
}

// ServerConfig holds HTTP listen settings.
type ServerConfig struct {
	Host           string // SERVER_HOST
	Port           string // SERVER_PORT
	ReadTimeoutSec int    // SERVER_READ_TIMEOUT
}

// CORSConfig holds allowed origins for credentialed requests.
type CORSConfig struct {
	Origins []string // CORS_ORIGINS, comma-separated
}

// JWTConfig holds token signing and lifetime settings.
type JWTConfig struct {
	Secret        string // JWT_SECRET_KEY
	AccessMinutes int    // JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT
	RefreshKey    string // JWT_REFRESH_KEY
	RefreshHours  int    // JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT
}

// DBConfig holds database selection and pool settings.
type DBConfig struct {
	DSN            string // SQL_DSN: empty = SQLite, postgres://... = PostgreSQL
	SQLitePath     string // SQLITE_PATH
	MaxConn        int    // DB_MAX_CONNECTIONS
	MaxIdle        int    // DB_MAX_IDLE_CONNECTIONS
	MaxLifetimeSec int    // DB_MAX_LIFETIME_CONNECTIONS
}

// RedisConfig holds optional Redis settings.
type RedisConfig struct {
	Host     string // REDIS_HOST: empty = in-memory sessions
	Port     string // REDIS_PORT
	Password string // REDIS_PASSWORD
	DBNumber int    // REDIS_DB_NUMBER
}

// Enabled reports whether Redis-backed stores should be used.
func (r RedisConfig) Enabled() bool { return strings.TrimSpace(r.Host) != "" }

// Addr returns host:port for the Redis client.
func (r RedisConfig) Addr() string { return r.Host + ":" + r.Port }

// RateLimitConfig holds requests/minute budgets (per IP, except gateway).
type RateLimitConfig struct {
	General int // RATE_LIMIT_GENERAL
	Auth    int // RATE_LIMIT_AUTH
	Public  int // RATE_LIMIT_PUBLIC
	Webhook int // RATE_LIMIT_WEBHOOK
	Gateway int // RATE_LIMIT_GATEWAY
}

// AIConfig holds AI provider and timeout settings.
type AIConfig struct {
	TimeoutSec  int    // AI_TIMEOUT_SECONDS
	GeminiKey   string // GEMINI_API_KEY
	GeminiModel string // GEMINI_MODEL
}

// MailConfig holds SMTP settings.
type MailConfig struct {
	SMTPHost     string // SMTP_HOST
	SMTPPort     string // SMTP_PORT
	SMTPUser     string // SMTP_USER
	SMTPPass     string // SMTP_PASS
	SMTPFrom     string // SMTP_FROM
	AppPublicURL string // APP_PUBLIC_URL (links inside emails)
}

// RelayConfig holds the public origin of this API for callbacks.
type RelayConfig struct {
	PublicURL string // INVOICEMAN_PUBLIC_URL
}

// IdempotencyConfig holds replay protection for mutating payment routes.
type IdempotencyConfig struct {
	TTLHours int // IDEMPOTENCY_TTL_HOURS
}

// OutboxConfig holds the background worker settings.
type OutboxConfig struct {
	PollSeconds int // OUTBOX_POLL_SECONDS
	Batch       int // OUTBOX_BATCH
}

// MidtransConfig holds Midtrans relay credentials.
type MidtransConfig struct {
	ServerKey string // MIDTRANS_SERVER_KEY
	ClientKey string // MIDTRANS_CLIENT_KEY
	IsProd    bool   // MIDTRANS_IS_PROD
	SnapBase  string // MIDTRANS_SNAP_BASE_URL (test override)
}

// NOWPaymentsConfig holds crypto relay credentials.
type NOWPaymentsConfig struct {
	APIKey    string // NOWPAYMENTS_API_KEY
	IPNSecret string // NOWPAYMENTS_IPN_SECRET
	Sandbox   bool   // NOWPAYMENTS_SANDBOX
	BaseURL   string // NOWPAYMENTS_BASE_URL (test override)
}

// LogConfig holds structured logging settings.
type LogConfig struct {
	Level string // LOG_LEVEL: debug|info|warn|error
}

// MetricsConfig holds observability endpoint settings.
type MetricsConfig struct {
	Enabled bool // METRICS_ENABLED
}

// IsDev reports dev stage (plain startup, text logs, simple server).
func (c Config) IsDev() bool { return c.Stage == "dev" }

// Load reads the environment, applies defaults and validates.
// It always returns the defaults-applied Config; err is non-nil when
// validation fails (main.go treats that as fatal).
func Load() (Config, error) {
	cfg := Config{
		Stage:   envOr("STAGE_STATUS", "dev"),
		AppName: envOr("APP_NAME", "Invoiceman"),
		Server: ServerConfig{
			Host:           envOr("SERVER_HOST", "0.0.0.0"),
			Port:           envOr("SERVER_PORT", "5000"),
			ReadTimeoutSec: envInt("SERVER_READ_TIMEOUT", 60),
		},
		CORS: CORSConfig{Origins: envList("CORS_ORIGINS")},
		JWT: JWTConfig{
			Secret:        strings.TrimSpace(os.Getenv("JWT_SECRET_KEY")),
			AccessMinutes: envInt("JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT", 15),
			RefreshKey:    strings.TrimSpace(os.Getenv("JWT_REFRESH_KEY")),
			RefreshHours:  envInt("JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT", 720),
		},
		DB: DBConfig{
			DSN:            strings.TrimSpace(os.Getenv("SQL_DSN")),
			SQLitePath:     envOr("SQLITE_PATH", "./data/invoiceman.db"),
			MaxConn:        envInt("DB_MAX_CONNECTIONS", 100),
			MaxIdle:        envInt("DB_MAX_IDLE_CONNECTIONS", 10),
			MaxLifetimeSec: envInt("DB_MAX_LIFETIME_CONNECTIONS", 2),
		},
		Redis: RedisConfig{
			Host:     strings.TrimSpace(os.Getenv("REDIS_HOST")),
			Port:     envOr("REDIS_PORT", "6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DBNumber: envInt("REDIS_DB_NUMBER", 0),
		},
		RateLimit: RateLimitConfig{
			General: envInt("RATE_LIMIT_GENERAL", 100),
			Auth:    envInt("RATE_LIMIT_AUTH", 10),
			Public:  envInt("RATE_LIMIT_PUBLIC", 30),
			Webhook: envInt("RATE_LIMIT_WEBHOOK", 60),
			Gateway: envInt("RATE_LIMIT_GATEWAY", 120),
		},
		AI: AIConfig{
			TimeoutSec:  envInt("AI_TIMEOUT_SECONDS", 30),
			GeminiKey:   strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
			GeminiModel: envOr("GEMINI_MODEL", "gemini-2.0-flash"),
		},
		Mail: MailConfig{
			SMTPHost:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			SMTPPort:     envOr("SMTP_PORT", "587"),
			SMTPUser:     strings.TrimSpace(os.Getenv("SMTP_USER")),
			SMTPPass:     os.Getenv("SMTP_PASS"),
			SMTPFrom:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
			AppPublicURL: envOr("APP_PUBLIC_URL", "http://localhost:5173"),
		},
		Midtrans: MidtransConfig{
			ServerKey: strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY")),
			ClientKey: strings.TrimSpace(os.Getenv("MIDTRANS_CLIENT_KEY")),
			IsProd:    envBool("MIDTRANS_IS_PROD", false),
			SnapBase:  strings.TrimSpace(os.Getenv("MIDTRANS_SNAP_BASE_URL")),
		},
		NOWPayments: NOWPaymentsConfig{
			APIKey:    strings.TrimSpace(os.Getenv("NOWPAYMENTS_API_KEY")),
			IPNSecret: strings.TrimSpace(os.Getenv("NOWPAYMENTS_IPN_SECRET")),
			Sandbox:   envBool("NOWPAYMENTS_SANDBOX", false),
			BaseURL:   strings.TrimSpace(os.Getenv("NOWPAYMENTS_BASE_URL")),
		},
		Relay: RelayConfig{
			PublicURL: envOr("INVOICEMAN_PUBLIC_URL", "http://localhost:5000"),
		},
		Idempotency: IdempotencyConfig{
			TTLHours: envInt("IDEMPOTENCY_TTL_HOURS", 24),
		},
		Outbox: OutboxConfig{
			PollSeconds: envInt("OUTBOX_POLL_SECONDS", 10),
			Batch:       envInt("OUTBOX_BATCH", 20),
		},
		Log: LogConfig{Level: strings.ToLower(envOr("LOG_LEVEL", "info"))},
		Metrics: MetricsConfig{
			Enabled: envBool("METRICS_ENABLED", true),
		},
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Get returns the current environment configuration with defaults applied.
// Validation errors are ignored here (fail-fast happens in main via Load);
// use Load directly when the error matters.
func Get() Config {
	cfg, _ := Load()
	return cfg
}

// Validate fail-fasts on configuration that would break or insecurely run
// the server. Dev stays permissive; prod requires real secrets.
func (c Config) Validate() error {
	if c.Stage != "dev" && c.Stage != "prod" {
		return fmt.Errorf("invalid STAGE_STATUS %q: must be \"dev\" or \"prod\"", c.Stage)
	}
	if c.Stage == "prod" {
		if c.JWT.Secret == "" || c.JWT.Secret == "secret" {
			return fmt.Errorf("JWT_SECRET_KEY must be set to a non-default value in prod")
		}
		if c.JWT.RefreshKey == "" || c.JWT.RefreshKey == "refresh" {
			return fmt.Errorf("JWT_REFRESH_KEY must be set to a non-default value in prod")
		}
	}
	if c.DSN() != "" && !strings.HasPrefix(c.DSN(), "postgres://") && !strings.HasPrefix(c.DSN(), "postgresql://") {
		return fmt.Errorf("unsupported SQL_DSN: leave it empty for SQLite or use a postgres://... DSN")
	}
	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid SERVER_PORT %q: must be 1-65535", c.Server.Port)
	}
	if c.Server.ReadTimeoutSec <= 0 {
		return fmt.Errorf("invalid SERVER_READ_TIMEOUT %q: must be > 0", os.Getenv("SERVER_READ_TIMEOUT"))
	}
	if c.JWT.AccessMinutes <= 0 || c.JWT.RefreshHours <= 0 {
		return fmt.Errorf("invalid JWT lifetimes: access minutes and refresh hours must be > 0")
	}
	for name, v := range map[string]int{
		"DB_MAX_CONNECTIONS": c.DB.MaxConn, "DB_MAX_IDLE_CONNECTIONS": c.DB.MaxIdle,
		"DB_MAX_LIFETIME_CONNECTIONS": c.DB.MaxLifetimeSec,
		"RATE_LIMIT_GENERAL":          c.RateLimit.General, "RATE_LIMIT_AUTH": c.RateLimit.Auth,
		"RATE_LIMIT_PUBLIC": c.RateLimit.Public, "RATE_LIMIT_WEBHOOK": c.RateLimit.Webhook,
		"RATE_LIMIT_GATEWAY": c.RateLimit.Gateway, "AI_TIMEOUT_SECONDS": c.AI.TimeoutSec,
		"IDEMPOTENCY_TTL_HOURS": c.Idempotency.TTLHours,
		"OUTBOX_POLL_SECONDS":   c.Outbox.PollSeconds, "OUTBOX_BATCH": c.Outbox.Batch,
	} {
		if v <= 0 {
			return fmt.Errorf("invalid %s: must be > 0", name)
		}
	}
	if c.Redis.Enabled() {
		if port, err := strconv.Atoi(c.Redis.Port); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid REDIS_PORT %q: must be 1-65535", c.Redis.Port)
		}
		if c.Redis.DBNumber < 0 {
			return fmt.Errorf("invalid REDIS_DB_NUMBER: must be >= 0")
		}
	}
	if c.Mail.SMTPHost != "" {
		if port, err := strconv.Atoi(c.Mail.SMTPPort); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid SMTP_PORT %q: must be 1-65535", c.Mail.SMTPPort)
		}
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q: must be debug|info|warn|error", c.Log.Level)
	}
	return nil
}

// DSN returns the database DSN (empty = SQLite).
func (c Config) DSN() string { return c.DB.DSN }

// ListenAddr returns host:port for the Fiber listener.
func (c Config) ListenAddr() string { return c.Server.Host + ":" + c.Server.Port }

func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func envInt(name string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); err == nil && v > 0 {
		return v
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	return strings.EqualFold(raw, "true")
}

func envList(name string) []string {
	var out []string
	for _, o := range strings.Split(strings.TrimSpace(os.Getenv(name)), ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}
