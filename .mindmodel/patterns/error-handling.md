# Error Handling

## Rules
- Controller errors via utils.Fail (stable English message, optional details)
- Query errors propagate to controller, wrapped with context
- Gateway errors map to standard errors (ErrNotConfigured, ErrInvalidSignature, ErrRateLimited)
- Business rule violations (422) separated from malformed input (400)
- AI errors logged server-side, generic message to client (no provider details leak)
- Frontend apiClient reads error.message and error.details from envelope
- No panic in production code; return errors up the stack

## Examples

### Controller Error Pattern
```go
// app/controllers/invoice_controller.go:26-50
func GetInvoice(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	detail, err := invoiceDetail(*db, userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}
```

### Business Rule Errors (422 vs 400)
```go
// app/controllers/invoice_rules.go:11-31
var (
	ErrClientRequiredToSend = errors.New("client is required to send an invoice")
	ErrPendingPayment       = errors.New("invoice has a pending payment")
	ErrInvoicePaid          = errors.New("invoice is already paid")
)

func invoiceRuleStatus(err error) int {
	if errors.Is(err, ErrClientRequiredToSend) || errors.Is(err, ErrPendingPayment) || errors.Is(err, ErrInvoicePaid) {
		return fiber.StatusUnprocessableEntity
	}
	return fiber.StatusBadRequest
}

func failInvoiceRule(c fiber.Ctx, err error) error {
	return utils.Fail(c, invoiceRuleStatus(err), err.Error(), nil)
}

func validateInvoice(status string, clientID *uuid.UUID) error {
	if status == models.InvoiceStatusSent && clientID == nil {
		return ErrClientRequiredToSend
	}
	return nil
}
```

### AI Error Handling (No Provider Leakage)
```go
// app/controllers/ai_controller.go:62-73
func aiError(c fiber.Ctx, err error) error {
	if errors.Is(err, ai.ErrNotConfigured) {
		return utils.Fail(c, fiber.StatusNotImplemented, "AI provider is not configured", nil)
	}
	// Logged server-side only; the client keeps generic messages so no
	// provider details leak to the browser.
	logger.L().Warn("AI provider request failed", "err", err)
	if errors.Is(err, ai.ErrRateLimited) {
		return utils.Fail(c, fiber.StatusTooManyRequests, "AI rate limit reached, please try again shortly", nil)
	}
	return utils.Fail(c, fiber.StatusBadGateway, "AI provider request failed", nil)
}
```

## Anti-patterns

### ❌ Exposing Internal Details
```go
// BAD: leak database path to client
return utils.Fail(c, 500, err.Error(), nil) // "cannot open /data/tupay.db"

// GOOD: stable generic message
return utils.Fail(c, 500, "database connection error", nil)
```

### ❌ Panic in Production Code
```go
// BAD: panic on error
if err != nil {
	panic(err)
}

// GOOD: return error
if err != nil {
	return utils.Fail(c, 500, "operation failed", nil)
}
```
