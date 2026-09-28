# Type Definitions

## Rules
- Money: decimal.Decimal (never float64)
- UUID: uuid.UUID from google/uuid
- Time: *time.Time for optional dates, time.Time for required
- GORM tags: gorm:"type:uuid;primaryKey" for IDs, gorm:"type:decimal(19,4)" for money
- JSON tags: json:"field_name" matching database snake_case
- Validation tags: validate:"required,uuid" for struct validation
- Frontend: TypeScript not used, JSDoc comments for complex types

## Examples

### Money Type Alias
```go
// app/models/money.go:3-7
package models

import "github.com/shopspring/decimal"

type Money = decimal.Decimal

var ZeroMoney = decimal.Zero
```

### Model with Type Tags
```go
// app/models/invoice_model.go:28-42
type Invoice struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id" validate:"required,uuid"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
	UserID    uuid.UUID  `gorm:"type:uuid" db:"user_id" json:"user_id" validate:"required,uuid"`
	InvoiceGatewayIdentity
	ClientID      *uuid.UUID `gorm:"type:uuid" db:"client_id" json:"client_id"`
	InvoiceNumber string     `db:"invoice_number" json:"invoice_number" validate:"required,lte=50"`
	Status        string     `db:"status" json:"status" validate:"required,oneof=draft sent paid"`
	IssueDate     *time.Time `db:"issue_date" json:"issue_date"`
	DueDate       *time.Time `db:"due_date" json:"due_date"`
	Currency      string     `db:"currency" json:"currency" validate:"required,lte=3"`
	TaxRate       float64    `db:"tax_rate" json:"tax_rate" validate:"gte=0"`
	Discount      Money      `gorm:"type:decimal(19,4)" db:"discount" json:"discount"`
	Subtotal      Money      `gorm:"type:decimal(19,4)" db:"subtotal" json:"subtotal"`
```

### Query Type
```go
// app/queries/invoice_query.go:13-16
type InvoiceQueries struct {
	*gorm.DB
}

var invoiceSortColumns = map[string]string{
```

## Anti-patterns

### ❌ Float64 for Money
```go
// BAD: precision loss
type Invoice struct {
	Total float64 `json:"total"`
}

// GOOD: decimal type
type Invoice struct {
	Total Money `gorm:"type:decimal(19,4)" json:"total"`
}
```

### ❌ String for IDs
```go
// BAD: string IDs
type Invoice struct {
	ID string `json:"id"`
}

// GOOD: UUID type
type Invoice struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
}
```
