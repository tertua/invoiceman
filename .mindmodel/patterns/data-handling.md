# Data Handling

## Rules
- Money type: decimal.Decimal (shopspring/decimal), never float64
- API wire format: money as decimal strings ("100000.50")
- Dates: YYYY-MM-DD strings (utils.FormatDate/ParseDate)
- Single date format across backend and frontend (DateLayout = "2006-01-02")
- Currency codes: 3-letter ISO (IDR, USD, EUR) stored with every invoice
- Optional dates use *time.Time (nil serializes as empty string)
- UUID primary keys for all entities

## Examples

### Money Type Definition
```go
// app/models/money.go:3-17
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

### Date Formatting
```go
// pkg/utils/date.go:9-36
const DateLayout = "2006-01-02"

func ParseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(DateLayout, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func FormatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(DateLayout)
}

func FormatTime(value time.Time) string {
	return value.Format(DateLayout)
}
```

### Model with Money and Dates
```go
// app/models/invoice_model.go:37-50
type Invoice struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" db:"id" json:"id"`
	IssueDate     *time.Time `db:"issue_date" json:"issue_date"`
	DueDate       *time.Time `db:"due_date" json:"due_date"`
	Currency      string     `db:"currency" json:"currency" validate:"required,lte=3"`
	TaxRate       float64    `db:"tax_rate" json:"tax_rate" validate:"gte=0"`
	Discount      Money      `gorm:"type:decimal(19,4)" db:"discount" json:"discount"`
	Subtotal      Money      `gorm:"type:decimal(19,4)" db:"subtotal" json:"subtotal"`
	TaxAmount     Money      `gorm:"type:decimal(19,4)" db:"tax_amount" json:"tax_amount"`
	Total         Money      `gorm:"type:decimal(19,4)" db:"total" json:"total"`
}
```

## Anti-patterns

### ❌ Float64 for Money
```go
// BAD: precision loss
var amount float64 = 100.01
tax := amount * 0.15 // might lose precision

// GOOD: decimal arithmetic
amount := decimal.NewFromFloat(100.01)
tax := amount.Mul(decimal.NewFromFloat(0.15))
```

### ❌ Inconsistent Date Formats
```go
// BAD: multiple formats across codebase
time.Parse("2006-01-02", date)
time.Parse("02/01/2006", date)

// GOOD: single format via utils
utils.ParseDate(date) // always YYYY-MM-DD
```

### ❌ Currency Defaults in AI
```go
// BAD: bare numbers → AI defaults to USD
prompt := "Calculate total: " + amount

// GOOD: currency directive
prompt := currencyDirective("IDR") + "Calculate total: " + amount
```
