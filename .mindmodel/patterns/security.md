# Security Patterns

## Rules
- Session auth: JWT access token (15m) + refresh cookie (7d), strict single-session (sid bound)
- CSRF double-submit: csrf_token cookie → X-CSRF-Token header (mutations only)
- API key auth: Authorization header for gateway relay (/gateway/*), never mixed with sessions
- Idempotency-Key header: money-moving mutations, replay protection with payload hash
- Gateway webhooks: signature verification via ParseAndVerify (HMAC, IP allowlist, API fetch)
- Public payment links: scoped idempotency (pay:<token>)
- Middleware order matters: auth → CSRF → idempotency
- Session cookies: HttpOnly, Secure in prod, SameSite=Lax

## Examples

### Session Auth with Transparent Refresh
```go
// pkg/middleware/auth_middleware.go:17-36
func AuthRequired() fiber.Handler {
	return func(c fiber.Ctx) error {
		if userID, sid, ok := validAccessToken(accessTokenString(c)); ok && sessionMatches(c, userID, sid) {
			c.Locals(utils.SessionUserIDKey, userID)
			return c.Next()
		}

		userID, ok := refreshSession(c)
		if !ok {
			return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
		}

		c.Locals(utils.SessionUserIDKey, userID)
		return c.Next()
	}
}

func accessTokenString(c fiber.Ctx) string {
	if token := c.Cookies(utils.AccessCookieName); token != "" {
		return token
	}
```

### CSRF Double-Submit Protection
```go
// pkg/middleware/csrf.go:17-52
const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

func NewCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func SetCSRFCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: false,
		Secure:   !configs.Get().IsDev(),
		SameSite: "Lax",
	})
}
```

### Idempotency Middleware with Scope
```go
// pkg/middleware/idempotency.go:29-65
type IdempotencyScope func(c fiber.Ctx) (string, error)

func SessionIdempotencyScope(c fiber.Ctx) (string, error) {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return "", err
	}
	return "user:" + userID.String(), nil
}

func GatewayIdempotencyScope(c fiber.Ctx) (string, error) {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return "", err
	}
	return "project:" + project.Slug, nil
}

func Idempotency(scope IdempotencyScope) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawKey := strings.TrimSpace(c.Get(idempotencyHeader))
		if rawKey == "" {
			return c.Next()
		}
		if len(rawKey) > idempotencyMaxKey {
			return utils.Fail(c, fiber.StatusBadRequest, "idempotency key is too long", nil)
		}
```

### Gateway Auth (API Key)
```go
// pkg/middleware/gateway_middleware.go:11-34
func GatewayAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		key := relay.ExtractKey(c.Get("Authorization"), c.Get("X-Api-Key"))
		if key == "" {
			return utils.Fail(c, fiber.StatusUnauthorized, "missing api key", nil)
		}
		db, err := database.OpenDBConnection()
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		}
		project, err := db.GetProjectByKeyHash(relay.HashKey(key))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return utils.Fail(c, fiber.StatusUnauthorized, "invalid api key", nil)
			}
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to load project", nil)
		}
		if !project.IsActive {
			return utils.Fail(c, fiber.StatusForbidden, "project is disabled", nil)
		}
```

## Anti-patterns

### ❌ Mixing Auth Schemes
```go
// BAD: accepting both session cookie and API key on same route
if sessionUser := utils.CurrentUserID(c); sessionUser != nil {
	// user context
} else if project := utils.CurrentServiceProject(c); project != nil {
	// project context
}

// GOOD: separate route groups with dedicated auth
```

### ❌ Skipping Signature Verification
```go
// BAD: trusting webhook payload without verification
func HandleWebhook(c fiber.Ctx) error {
	var payload WebhookPayload
	c.Bind().Body(&payload)
	// process without verifying sender
}

// GOOD: ParseAndVerify first
notif, err := gw.ParseAndVerify(raw)
```
