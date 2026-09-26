package configs

// Package configs is the single source of truth for backend configuration.
// All values come from the environment (see .env.example) with dev-friendly
// defaults. Call Load once at startup (main.go); elsewhere use Get, which
// returns defaults-applied values without validation so tests keep working.
// Parsing is plain stdlib so zero-config dev stays intact.

import (
	"fmt"
	"net/netip"
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
	Auth      AuthConfig
	SPA       SPAConfig

	Midtrans    MidtransConfig
	NOWPayments NOWPaymentsConfig

	Relay RelayConfig
	Log   LogConfig

	Idempotency IdempotencyConfig

	Outbox OutboxConfig

	// Cache holds aggregate-cache settings. The cache is dual-backend by
	// design: Redis when REDIS_HOST is set, process-local memory otherwise.
	// Behaviour (keys, TTL, invalidation) is identical either way; only
	// sharing across replicas differs.
	Cache CacheConfig

	Captcha CaptchaConfig

	Debug DebugConfig

	Storage StorageConfig

	Metrics MetricsConfig
}

// ServerConfig holds HTTP listen settings.
type ServerConfig struct {
	Host           string   // SERVER_HOST
	Port           string   // SERVER_PORT
	ReadTimeoutSec int      // SERVER_READ_TIMEOUT
	TrustedProxies []string // TRUSTED_PROXIES, comma-separated IPs/CIDRs
	ProxyHeader    string   // PROXY_HEADER (default X-Forwarded-For)
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

// AuthConfig holds public auth toggles.
type AuthConfig struct {
	AllowRegistration bool // ALLOW_REGISTRATION
}

// SPAConfig holds the optional embedded-frontend settings.
type SPAConfig struct {
	Dir string // SERVE_SPA_DIR: empty = API-only (default)
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

// CacheConfig holds the dashboard/report aggregate cache settings.
type CacheConfig struct {
	// AggTTLSeconds bounds staleness of cached aggregates
	// (CACHE_AGG_TTL_SECONDS). Writes invalidate explicitly, so this only
	// matters for multi-replica deployments without Redis.
	AggTTLSeconds int
}

// CaptchaConfig holds bot protection settings.
type CaptchaConfig struct {
	TurnstileSecret string // TURNSTILE_SECRET: empty = verification skipped
}

// DebugConfig holds the localhost-only diagnostics listener.
type DebugConfig struct {
	Port string // DEBUG_PORT: empty = disabled
}

// StorageConfig holds file storage settings (logos, receipts).
type StorageConfig struct {
	Backend   string // STORAGE_BACKEND: "local" or "s3"
	Dir       string // STORAGE_DIR for the local backend
	PublicURL string // STORAGE_PUBLIC_URL: public origin for stored files

	S3Endpoint  string // S3_ENDPOINT
	S3Bucket    string // S3_BUCKET
	S3Region    string // S3_REGION
	S3AccessKey string // S3_ACCESS_KEY
	S3SecretKey string // S3_SECRET_KEY
	S3UseSSL    bool   // S3_USE_SSL
}

// MidtransConfig holds Midtrans relay credentials.
type MidtransConfig struct {
	ServerKey string // MIDTRANS_SERVER_KEY
	ClientKey string // MIDTRANS_CLIENT_KEY
	IsProd    bool   // MIDTRANS_IS_PROD
	SnapBase  string // MIDTRANS_SNAP_BASE_URL (test override)
	CoreBase  string // MIDTRANS_CORE_BASE_URL (test override)
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
			TrustedProxies: envList("TRUSTED_PROXIES"),
			ProxyHeader:    envOr("PROXY_HEADER", "X-Forwarded-For"),
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
		Auth: AuthConfig{
			AllowRegistration: envBool("ALLOW_REGISTRATION", true),
		},
		SPA: SPAConfig{
			Dir: strings.TrimSpace(os.Getenv("SERVE_SPA_DIR")),
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
			CoreBase:  strings.TrimSpace(os.Getenv("MIDTRANS_CORE_BASE_URL")),
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
		Cache: CacheConfig{
			AggTTLSeconds: envInt("CACHE_AGG_TTL_SECONDS", 60),
		},
		Captcha: CaptchaConfig{
			TurnstileSecret: strings.TrimSpace(os.Getenv("TURNSTILE_SECRET")),
		},
		Debug: DebugConfig{
			Port: strings.TrimSpace(os.Getenv("DEBUG_PORT")),
		},
		Storage: StorageConfig{
			Backend:     envOr("STORAGE_BACKEND", "local"),
			Dir:         envOr("STORAGE_DIR", "./data/uploads"),
			PublicURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("STORAGE_PUBLIC_URL")), "/"),
			S3Endpoint:  strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
			S3Bucket:    strings.TrimSpace(os.Getenv("S3_BUCKET")),
			S3Region:    envOr("S3_REGION", "us-east-1"),
			S3AccessKey: strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
			S3SecretKey: os.Getenv("S3_SECRET_KEY"),
			S3UseSSL:    envBool("S3_USE_SSL", true),
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
	for _, p := range c.Server.TrustedProxies {
		if _, err := netip.ParsePrefix(p); err != nil {
			if _, err := netip.ParseAddr(p); err != nil {
				return fmt.Errorf("invalid TRUSTED_PROXIES entry %q: must be an IP or CIDR", p)
			}
		}
	}
	// Note: ProxyHeader needs no validation — envOr substitutes the
	// X-Forwarded-For default for blank values, so it is never empty.
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
		"CACHE_AGG_TTL_SECONDS": c.Cache.AggTTLSeconds,
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
	if c.Debug.Port != "" {
		if port, err := strconv.Atoi(c.Debug.Port); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid DEBUG_PORT %q: must be 1-65535", c.Debug.Port)
		}
	}
	if c.Storage.Backend != "local" && c.Storage.Backend != "s3" {
		return fmt.Errorf("invalid STORAGE_BACKEND %q: must be \"local\" or \"s3\"", c.Storage.Backend)
	}
	if c.Storage.Backend == "s3" {
		if c.Storage.S3Endpoint == "" || c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3_ENDPOINT and S3_BUCKET are required when STORAGE_BACKEND=s3")
		}
		if c.Storage.S3AccessKey == "" || c.Storage.S3SecretKey == "" {
			return fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY are required when STORAGE_BACKEND=s3")
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
