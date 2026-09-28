# Integration Patterns

## Rules
- Gateway interface: Name(), CreateTransaction(), ParseAndVerify() methods
- Register gateways in main.go via gateway.Register(provider.New())
- Webhook verification: signature/HMAC check in ParseAndVerify, reject invalid with ErrInvalidSignature
- Outbox worker: durable job queue (mail_outbox, webhook_deliveries) with retry backoff
- Retry policy: 1m, 2m, 4m, ... capped at 2h, dead after 10 attempts
- Gateway status mapping: provider-specific → standard (pending/success/failed/expired/refunded)
- Relay pattern: downstream projects charge via Tupay, webhook forwarded with HMAC signature
- Worker runs in-process (stdlib only), safe with multiple replicas (atomic claiming per row)

## Examples

### Gateway Interface
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

### Gateway Registration in main.go
```go
// main.go:96-97
gateway.Register(midtrans.New())
gateway.Register(nowpayments.New())
```

### Webhook Handler with Verification
```go
// app/controllers/gateway_webhook_controller.go:30-54
func handleGatewayWebhook(c fiber.Ctx, gatewayName string) error {
	gw, err := gateway.Get(gatewayName)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "unknown payment gateway", nil)
	}
	raw := append([]byte(nil), c.Body()...)
	notif, err := gw.ParseAndVerify(raw)
	if err != nil {
		switch {
		case errors.Is(err, gateway.ErrNotConfigured):
			return utils.Fail(c, fiber.StatusNotImplemented, "payment gateway is not configured", nil)
		case errors.Is(err, gateway.ErrInvalidSignature):
			return utils.Fail(c, fiber.StatusUnauthorized, "invalid signature", nil)
		default:
			return utils.Fail(c, fiber.StatusBadRequest, "invalid notification", nil)
		}
	}
	status := notif.Status
```

### Outbox Worker with Retry Backoff
```go
// platform/outbox/worker.go:31-50
const (
	maxAttempts  = 10
	maxBackoff   = 2 * time.Hour
)

func Backoff(n int) time.Duration {
	d := time.Minute << (n - 1)
	if d <= 0 || d > maxBackoff {
		return maxBackoff
	}
	return d
}

func nextRetryAt(attempt int, now time.Time) *time.Time {
	if attempt >= maxAttempts {
		return nil
	}
	at := now.Add(Backoff(attempt))
	return &at
}

var SendMail = func(to, subject, textBody, htmlBody string) error {
	mailer, err := mail.NewFromEnv()
```

## Anti-patterns

### ❌ Direct Provider SDK in Controller
```go
// BAD: controller imports midtrans SDK
import "github.com/midtrans/midtrans-go"
midtrans.CreateTransaction(...)

// GOOD: via gateway abstraction
gw, _ := gateway.Get("midtrans")
gw.CreateTransaction(ctx, req)
```

### ❌ Webhook Without Verification
```go
// BAD: trust payload without signature check
func HandleWebhook(c fiber.Ctx) error {
	var payload Notification
	c.Bind().Body(&payload)
	processPayment(payload.TransactionID)
}

// GOOD: verify first
notif, err := gw.ParseAndVerify(raw)
if err != nil {
	return utils.Fail(c, 401, "invalid signature", nil)
}
```
