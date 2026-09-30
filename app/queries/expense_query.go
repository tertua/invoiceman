package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// ExpenseQueries provides expense persistence.
type ExpenseQueries struct {
	*gorm.DB
}

// ListExpenses returns one page of expenses owned by an org, optionally filtered by category.
func (q *ExpenseQueries) ListExpenses(orgID uuid.UUID, category string, limit, offset int) ([]models.Expense, error) {
	expenses := []models.Expense{}
	tx := q.Where("org_id = ?", orgID)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Order("expense_date DESC").Order("created_at DESC").Limit(limit).Offset(offset).Find(&expenses).Error; err != nil {
		return expenses, err
	}
	return expenses, nil
}

// CountExpenses returns the total expenses matching the category filter.
func (q *ExpenseQueries) CountExpenses(orgID uuid.UUID, category string) (int64, error) {
	var total int64
	tx := q.Model(&models.Expense{}).Where("org_id = ?", orgID)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// ExpenseTotals holds global aggregates over the category filter.
type ExpenseTotals struct {
	Total     decimal.Decimal
	ThisMonth decimal.Decimal
}

// GetExpenseTotals returns all-time and current-month sums; month bounds are computed in Go for portability.
func (q *ExpenseQueries) GetExpenseTotals(orgID uuid.UUID, category string) (ExpenseTotals, error) {
	totals := ExpenseTotals{}
	tx := q.Model(&models.Expense{}).Where("org_id = ?", orgID)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Select("COALESCE(SUM(amount), 0) AS total").Scan(&totals).Error; err != nil {
		return totals, err
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonth := monthStart.AddDate(0, 1, 0)
	monthTx := q.Model(&models.Expense{}).Where("org_id = ? AND expense_date >= ? AND expense_date < ?", orgID, monthStart, nextMonth)
	if category != "" && category != "all" {
		monthTx = monthTx.Where("category = ?", category)
	}
	var thisMonth decimal.Decimal
	if err := monthTx.Select("COALESCE(SUM(amount), 0)").Scan(&thisMonth).Error; err != nil {
		return totals, err
	}
	totals.ThisMonth = thisMonth
	return totals, nil
}

// ExpenseCategories returns the distinct categories of an org for the filter.
func (q *ExpenseQueries) ExpenseCategories(orgID uuid.UUID) ([]string, error) {
	categories := []string{}
	if err := q.Model(&models.Expense{}).Where("org_id = ?", orgID).
		Distinct().Pluck("category", &categories).Error; err != nil {
		return categories, err
	}
	return categories, nil
}

// GetExpense returns one expense of an org; the receipt path resolves by id plus org scope.
func (q *ExpenseQueries) GetExpense(orgID, id uuid.UUID) (models.Expense, error) {
	expense := models.Expense{}
	if err := q.Where("id = ? AND org_id = ?", id, orgID).First(&expense).Error; err != nil {
		return expense, notFound(err)
	}
	return expense, nil
}

// CreateExpense persists an expense; an unset OrgID rides as the creator UserID (rides-as), UserID stays audit.
func (q *ExpenseQueries) CreateExpense(expense *models.Expense) error {
	return rideOrg(q.DB, expense, &expense.OrgID, expense.UserID)
}

// UpdateExpense updates an expense owned by an org.
func (q *ExpenseQueries) UpdateExpense(expense *models.Expense) error {
	return q.Model(&models.Expense{}).Where("id = ? AND org_id = ?", expense.ID, expense.OrgID).
		Updates(map[string]any{
			"updated_at":   time.Now(),
			"vendor":       expense.Vendor,
			"category":     expense.Category,
			"expense_date": expense.ExpenseDate,
			"amount":       expense.Amount,
			"currency":     expense.Currency,
			"notes":        expense.Notes,
			"receipt_url":  expense.ReceiptURL,
		}).Error
}

// DeleteExpense deletes an expense owned by an org.
func (q *ExpenseQueries) DeleteExpense(orgID, id uuid.UUID) error {
	return q.Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Expense{}).Error
}
