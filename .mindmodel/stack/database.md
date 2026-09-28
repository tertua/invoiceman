# Database Stack

## Rules
- Dual-dialect: SQLite (default, in-memory for tests) and PostgreSQL (production-ready)
- GORM AutoMigrate at startup for schema upgrades
- Migrations in platform/database/migrations.go with Down rollback functions
- Down must be backend-agnostic (Migrator calls only, no raw SQL)
- Empty SQL_DSN = auto SQLite in data/ directory
- Test suite uses in-memory SQLite by default, opt-in PG/Redis via env vars

## Examples

### Migration Structure
```go
// platform/database/migrations.go:14-25
type Migration struct {
	Version     int
	Description string
	Down        func(db *gorm.DB) error
}

func MigrateDownTo(target int) (int, error) {
	db, err := openShared()
	if err != nil {
		return 0, err
	}
	row := models.SchemaMigration{}
```

### Model with UUID Primary Key
```go
// app/models/invoice_model.go:28-40
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
```

### Query with Ownership Filter
```go
// app/queries/invoice_query.go:63-76
func (q *InvoiceQueries) filteredInvoices(userID uuid.UUID, status, search string) *gorm.DB {
	paidSubquery := q.Model(&models.Payment{}).
		Select("invoice_id, SUM(amount) AS paid").
		Where("voided_at IS NULL").
		Group("invoice_id")

	tx := q.Table("invoices").
		Select(`invoices.id, invoices.invoice_number,
			COALESCE(clients.name, '') AS client_name,
			COALESCE(clients.company, '') AS client_company,
			invoices.issue_date, invoices.due_date, invoices.total,
			invoices.currency, invoices.status,
			COALESCE(pay.paid, 0) AS paid_amount`).
		Joins("LEFT JOIN clients ON clients.id = invoices.client_id").
```

## Anti-patterns

### ❌ Raw SQL in Controllers
```go
// BAD: controller executes raw query
db.Exec("SELECT * FROM invoices WHERE user_id = ?", userID)

// GOOD: query layer
queries.ListInvoices(userID, status, search, sort, order, limit, offset)
```

### ❌ Non-Agnostic Migration Down
```go
// BAD: PostgreSQL-specific syntax in Down
db.Exec("DROP INDEX CONCURRENTLY idx_name")

// GOOD: Migrator call
db.Migrator().DropIndex(&models.Invoice{}, "idx_name")
```
