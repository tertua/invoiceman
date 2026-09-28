# Architecture Layers

## Rules
- Controllers in app/controllers handle HTTP, validate input, call business logic
- Queries in app/queries encapsulate all database reads/writes (never inline in controllers)
- Models in app/models define structs, validation tags, constants
- Platform layer (platform/*) provides gateway, database, cache, mail, outbox abstractions
- Utils in pkg/utils provide shared helpers (response envelopes, date parsing, session context)
- Raw SQL only in platform/database, everywhere else uses GORM
- Controllers follow pattern: CurrentUserID → OpenDBConnection → business logic → utils.OK/Fail

## Examples

### Controller Pattern
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

### Query Layer Separation
```go
// app/queries/invoice_query.go:13-18
type InvoiceQueries struct {
	*gorm.DB
}

var invoiceSortColumns = map[string]string{
	"issue_date": "invoices.issue_date",
```

### Response Envelope Helpers
```go
// pkg/utils/response.go:7-24
func OK(c fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

func Fail(c fiber.Ctx, status int, message string, details any) error {
	errObject := fiber.Map{"message": message}
	if details != nil {
		errObject["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errObject})
}

func ValidationFailed(c fiber.Ctx, err error) error {
	return Fail(c, fiber.StatusBadRequest, "validation failed", ValidatorErrors(err))
}
```

## Anti-patterns

### ❌ Inline SQL in Controllers
```go
// BAD: controller does raw query
db.Raw("SELECT * FROM invoices WHERE user_id = ?", userID).Scan(&invoices)

// GOOD: call query layer
queries.ListInvoices(userID, status, search, sort, order, limit, offset)
```

### ❌ Business Logic in Query Layer
```go
// BAD: query validates invoice rules
func (q *InvoiceQueries) CreateInvoice(input InvoiceInput) error {
	if input.Status == "sent" && input.ClientID == nil {
		return errors.New("client required")
	}
}

// GOOD: controller validates, query persists
validateInvoice(input.Status, clientID) // in controller
queries.CreateInvoice(invoice) // in query layer
```
