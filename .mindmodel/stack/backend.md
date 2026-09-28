# Backend Stack

## Rules
- Go 1.27.1 pinned toolchain (never upgrade without updating go.mod and CI)
- Fiber v3 web framework for HTTP handling
- GORM for models and queries; dual-dialect support (SQLite + PostgreSQL)
- Redis optional (in-memory fallback when REDIS_HOST empty)
- JWT sessions: access token (15m) + refresh cookie (7d), strict single-session (sid bound)
- Godotenv autoloads .env file at startup
- Decimal arithmetic via shopspring/decimal for all money values

## Examples

### Main Entry Point with Gateway Registration
```go
// main.go:81-104
func run() int {
	cfg, err := configs.Load()
	if err != nil {
		logger.Init("dev", "error")
		logger.L().Error("invalid configuration", "err", err)
		return 1
	}

	logger.Init(cfg.Stage, cfg.Log.Level)

	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		return healthcheck(cfg.Server.Port)
	}

	if err := database.OpenConnection(); err != nil {
		logger.L().Error("database connection failed", "err", err)
		return 1
	}
	defer database.CloseConnection()

	// Register payment gateways
	gateway.Register(midtrans.New())
	gateway.Register(nowpayments.New())

	worker := outbox.NewWorker()
```

### Money Type (Decimal, Never Float)
```go
// app/models/money.go:1-17
package models

import "github.com/shopspring/decimal"

type Money = decimal.Decimal

var ZeroMoney = decimal.Zero

func MoneyFromMinor(minor int64) Money {
	return decimal.NewFromInt(minor)
}

func DecimalFromFloat(value float64) Money {
	return decimal.NewFromFloat(value)
}
```

### Gateway Interface Pattern
```go
// platform/gateway/gateway.go:37-50
type Gateway interface {
	Name() string
	CreateTransaction(ctx context.Context, req *CreateTxRequest) (*CreateTxResponse, error)
	ParseAndVerify(raw []byte) (*NotificationResult, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Gateway{}
)

func Register(g Gateway) {
	mu.Lock()
	defer mu.Unlock()
	registry[g.Name()] = g
}
```

## Anti-patterns

### ❌ Float64 for Money
```go
// BAD: precision loss in financial calculations
amount := 100.01 * 1.15 // might be 115.01149999
```

### ❌ Direct Provider SDK in Controllers
```go
// BAD: controller imports midtrans SDK directly
import "github.com/midtrans/midtrans-go"
// Gateway abstraction must be used instead
```
