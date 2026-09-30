package configs

// Package configs is the single source of truth for backend configuration.
// All values come from the environment (see .env.example) with dev-friendly
// defaults. Call Load once at startup (main.go); elsewhere use Get, which
// returns defaults-applied values without validation so tests keep working.
// Parsing is plain stdlib so zero-config dev stays intact.

import (
	"errors"
	"os"
	"strings"
	"time"
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
	WebUI     WebUIConfig

	Midtrans    MidtransConfig
	NOWPayments NOWPaymentsConfig

	Relay RelayConfig
	Log   LogConfig

	Idempotency IdempotencyConfig

	Outbox OutboxConfig

	// Gateway holds the provider-call timeouts, retry policy and polling
	// cadence that used to be hardcoded constants. Defaults reproduce the old
	// constants exactly, so unsetting them changes nothing at runtime.
	Gateway GatewayConfig

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

// WebUIConfig holds the optional embedded-frontend settings.
type WebUIConfig struct {
	Dir string // SERVE_WEBUI (legacy SERVE_SPA_DIR): empty = API-only (default)
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

// RelayConfig holds this API's public origin for provider webhooks.
type RelayConfig struct {
	PublicURL string // TUPAY_PUBLIC_URL (fallback INVOICEMAN_PUBLIC_URL)
}

// IdempotencyConfig holds replay protection for mutating payment routes.
type IdempotencyConfig struct {
	TTLHours int // IDEMPOTENCY_TTL_HOURS
}

// OutboxConfig holds the background worker settings.
type OutboxConfig struct {
	PollSeconds       int // OUTBOX_POLL_SECONDS
	Batch             int // OUTBOX_BATCH
	MaxAttempts       int // OUTBOX_MAX_ATTEMPTS
	MaxBackoffMinutes int // OUTBOX_MAX_BACKOFF_MINUTES
}

// GatewayConfig holds provider-call timeouts, retry policy and polling cadence.
// Each field has an accessor that converts to time.Duration so call sites stay
// import-light (they read configs, not time).
type GatewayConfig struct {
	ReconcileMinutes         int // GATEWAY_RECONCILE_MINUTES
	APITimeoutSec            int // GATEWAY_API_TIMEOUT_SECONDS
	PaymentTimeoutSec        int // GATEWAY_PAYMENT_TIMEOUT_SECONDS
	QRFetchTimeoutSec        int // QR_FETCH_TIMEOUT_SECONDS
	WebhookForwardTimeoutSec int // WEBHOOK_FORWARD_TIMEOUT_SECONDS
	MaxAttempts              int // GATEWAY_MAX_ATTEMPTS
	RetryBaseSec             int // GATEWAY_RETRY_BASE_SECONDS
	RetryMaxSec              int // GATEWAY_RETRY_MAX_SECONDS
	WebhookRetryMinutes      int // WEBHOOK_RETRY_MINUTES
}

// APITimeout is the per-request timeout for gateway API operations.
func (g GatewayConfig) APITimeout() time.Duration {
	return time.Duration(g.APITimeoutSec) * time.Second
}

// PaymentTimeout bounds payment-creation calls that may take longer.
func (g GatewayConfig) PaymentTimeout() time.Duration {
	return time.Duration(g.PaymentTimeoutSec) * time.Second
}

// QRFetchTimeout bounds QR image fetches.
func (g GatewayConfig) QRFetchTimeout() time.Duration {
	return time.Duration(g.QRFetchTimeoutSec) * time.Second
}

// WebhookForwardTimeout bounds forwarding webhooks downstream.
func (g GatewayConfig) WebhookForwardTimeout() time.Duration {
	return time.Duration(g.WebhookForwardTimeoutSec) * time.Second
}

// RetryBase is the linear backoff base before jitter.
func (g GatewayConfig) RetryBase() time.Duration {
	return time.Duration(g.RetryBaseSec) * time.Second
}

// RetryMax is the backoff ceiling and the Retry-After cap (one knob).
func (g GatewayConfig) RetryMax() time.Duration {
	return time.Duration(g.RetryMaxSec) * time.Second
}

// ReconcileAge spaces provider status polls per stale transaction.
func (g GatewayConfig) ReconcileAge() time.Duration {
	return time.Duration(g.ReconcileMinutes) * time.Minute
}

// WebhookRetry is the manual retry delay for relay deliveries.
func (g GatewayConfig) WebhookRetry() time.Duration {
	return time.Duration(g.WebhookRetryMinutes) * time.Minute
}

// MaxBackoff is the outbox exponential-backoff ceiling.
func (o OutboxConfig) MaxBackoff() time.Duration {
	return time.Duration(o.MaxBackoffMinutes) * time.Minute
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
	TimeoutSec      int    // CAPTCHA_TIMEOUT_SECONDS
}

// Timeout bounds Cloudflare Turnstile verification.
func (c CaptchaConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutSec) * time.Second
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
	// intEnv collects every bad integer env value instead of swallowing it:
	// a set-but-invalid value (0, negative, non-numeric) is a startup error
	// naming the key and the raw value (D2).
	var issues []string
	intEnv := func(name string, fallback int) int {
		v, err := envInt(name, fallback)
		if err != nil {
			issues = append(issues, err.Error())
			return fallback
		}
		return v
	}
	cfg := Config{
		Stage:   envOr("STAGE_STATUS", "dev"),
		AppName: envOr("APP_NAME", "TuPay"),
		Server: ServerConfig{
			Host:           envOr("SERVER_HOST", "0.0.0.0"),
			Port:           envOr("SERVER_PORT", "5000"),
			ReadTimeoutSec: intEnv("SERVER_READ_TIMEOUT", 60),
			TrustedProxies: envList("TRUSTED_PROXIES"),
			ProxyHeader:    envOr("PROXY_HEADER", "X-Forwarded-For"),
		},
		CORS: CORSConfig{Origins: envList("CORS_ORIGINS")},
		JWT: JWTConfig{
			Secret:        strings.TrimSpace(os.Getenv("JWT_SECRET_KEY")),
			AccessMinutes: intEnv("JWT_SECRET_KEY_EXPIRE_MINUTES_COUNT", 15),
			RefreshKey:    strings.TrimSpace(os.Getenv("JWT_REFRESH_KEY")),
			RefreshHours:  intEnv("JWT_REFRESH_KEY_EXPIRE_HOURS_COUNT", 720),
		},
		DB: DBConfig{
			DSN:            strings.TrimSpace(os.Getenv("SQL_DSN")),
			SQLitePath:     envOr("SQLITE_PATH", "./data/db/tupay.db"),
			MaxConn:        intEnv("DB_MAX_CONNECTIONS", 100),
			MaxIdle:        intEnv("DB_MAX_IDLE_CONNECTIONS", 10),
			MaxLifetimeSec: intEnv("DB_MAX_LIFETIME_CONNECTIONS", 2),
		},
		Redis: RedisConfig{
			Host:     strings.TrimSpace(os.Getenv("REDIS_HOST")),
			Port:     envOr("REDIS_PORT", "6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DBNumber: envIntAllowZero("REDIS_DB_NUMBER", 0, &issues),
		},
		RateLimit: RateLimitConfig{
			General: intEnv("RATE_LIMIT_GENERAL", 100),
			Auth:    intEnv("RATE_LIMIT_AUTH", 10),
			Public:  intEnv("RATE_LIMIT_PUBLIC", 30),
			Webhook: intEnv("RATE_LIMIT_WEBHOOK", 60),
			Gateway: intEnv("RATE_LIMIT_GATEWAY", 120),
		},
		AI: AIConfig{
			TimeoutSec:  intEnv("AI_TIMEOUT_SECONDS", 30),
			GeminiKey:   strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
			GeminiModel: envOr("GEMINI_MODEL", "gemini-2.0-flash"),
		},
		Auth: AuthConfig{
			AllowRegistration: envBool("ALLOW_REGISTRATION", true),
		},
		WebUI: WebUIConfig{
			Dir: strings.TrimSpace(envOr("SERVE_WEBUI", os.Getenv("SERVE_SPA_DIR"))),
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
			PublicURL: envOr("TUPAY_PUBLIC_URL", envOr("INVOICEMAN_PUBLIC_URL", "http://localhost:5000")),
		},
		Idempotency: IdempotencyConfig{
			TTLHours: intEnv("IDEMPOTENCY_TTL_HOURS", 24),
		},
		Outbox: OutboxConfig{
			PollSeconds:       intEnv("OUTBOX_POLL_SECONDS", 10),
			Batch:             intEnv("OUTBOX_BATCH", 20),
			MaxAttempts:       intEnv("OUTBOX_MAX_ATTEMPTS", 10),
			MaxBackoffMinutes: intEnv("OUTBOX_MAX_BACKOFF_MINUTES", 120),
		},
		Gateway: GatewayConfig{
			ReconcileMinutes:         intEnv("GATEWAY_RECONCILE_MINUTES", 15),
			APITimeoutSec:            intEnv("GATEWAY_API_TIMEOUT_SECONDS", 15),
			PaymentTimeoutSec:        intEnv("GATEWAY_PAYMENT_TIMEOUT_SECONDS", 25),
			QRFetchTimeoutSec:        intEnv("QR_FETCH_TIMEOUT_SECONDS", 10),
			WebhookForwardTimeoutSec: intEnv("WEBHOOK_FORWARD_TIMEOUT_SECONDS", 10),
			MaxAttempts:              intEnv("GATEWAY_MAX_ATTEMPTS", 3),
			RetryBaseSec:             intEnv("GATEWAY_RETRY_BASE_SECONDS", 1),
			RetryMaxSec:              intEnv("GATEWAY_RETRY_MAX_SECONDS", 30),
			WebhookRetryMinutes:      intEnv("WEBHOOK_RETRY_MINUTES", 5),
		},
		Cache: CacheConfig{
			AggTTLSeconds: intEnv("CACHE_AGG_TTL_SECONDS", 60),
		},
		Captcha: CaptchaConfig{
			TurnstileSecret: strings.TrimSpace(os.Getenv("TURNSTILE_SECRET")),
			TimeoutSec:      intEnv("CAPTCHA_TIMEOUT_SECONDS", 3),
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
	if len(issues) > 0 {
		return cfg, errors.New(strings.Join(issues, "; "))
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

// DSN returns the database DSN (empty = SQLite).
func (c Config) DSN() string { return c.DB.DSN }

// ListenAddr returns host:port for the Fiber listener.
func (c Config) ListenAddr() string { return c.Server.Host + ":" + c.Server.Port }
