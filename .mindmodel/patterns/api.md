# API Patterns

## Rules
- Envelope responses: success via utils.OK (direct data fields), errors via utils.Fail (error.message + error.details)
- Versioning: /api/v1 before /api in route registration (order matters!)
- Legacy /api routes carry Deprecation and Sunset headers
- List endpoints: ?page (default 1), ?per_page (default 20, max 100), always return meta object
- Dates: YYYY-MM-DD strings across all endpoints
- Money: decimal strings on wire format
- Route order within prefix: public → gateway → private
- HTTP status: 200 OK, 201 Created, 204 No Content (logout, most DELETEs), 422 business rule violations

## Examples

### Success Response (utils.OK)
```go
// pkg/utils/response.go:7-11
func OK(c fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

// Usage in controller:
return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
```

### Error Response (utils.Fail)
```go
// pkg/utils/response.go:13-24
func Fail(c fiber.Ctx, status int, message string, details any) error {
	errObject := fiber.Map{"message": message}
	if details != nil {
		errObject["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errObject})
}

// Usage:
return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
```

### Route Registration Order (Critical!)
```go
// pkg/routes/versioning.go:21-42
const (
	APILegacyPrefix = "/api"
	APIV1Prefix     = "/api/v1"
	SunsetDate = "Mon, 01 Mar 2027 00:00:00 GMT"
)

func RegisterAPI(a *fiber.App, prefix string) {
	PublicRoutesAt(a, prefix)
	GatewayRoutesAt(a, prefix)
	PrivateRoutesAt(a, prefix)
}

func DeprecationHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		p := c.Path()
		if strings.HasPrefix(p, APILegacyPrefix+"/") && !strings.HasPrefix(p, APIV1Prefix+"/") {
			c.Set("Deprecation", "true")
			c.Set("Sunset", SunsetDate)
		}
		return c.Next()
	}
}
```

### Swagger Annotation Example
```go
// app/controllers/invoice_controller.go:17-26
// GetInvoice returns one invoice with items and payments.
// @Description Get invoice by ID.
// @Summary get invoice by ID
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id} [get]
```

## Anti-patterns

### ❌ Wrong Route Registration Order
```go
// BAD: /api before /api/v1 (shadowing!)
RegisterAPI(app, APILegacyPrefix)
RegisterAPI(app, APIV1Prefix)

// GOOD: /api/v1 first
RegisterAPI(app, APIV1Prefix)
RegisterAPI(app, APILegacyPrefix)
```

### ❌ Inconsistent Envelope
```go
// BAD: direct JSON response
return c.JSON(fiber.Map{"data": invoice})

// GOOD: via utils.OK
return utils.OK(c, 200, fiber.Map{"invoice": invoice})
```

### ❌ Private Before Gateway
```go
// BAD: private routes shadow gateway routes
func RegisterAPI(a *fiber.App, prefix string) {
	PrivateRoutesAt(a, prefix)
	GatewayRoutesAt(a, prefix)
}

// GOOD: gateway before private
func RegisterAPI(a *fiber.App, prefix string) {
	PublicRoutesAt(a, prefix)
	GatewayRoutesAt(a, prefix)
	PrivateRoutesAt(a, prefix)
}
```
