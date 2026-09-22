package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// ExpenseQueries provides expense persistence operations.
type ExpenseQueries struct {
	*gorm.DB
}

// ListExpenses returns one page of expenses owned by a user,
// optionally filtered by category.
func (q *ExpenseQueries) ListExpenses(userID uuid.UUID, category string, limit, offset int) ([]models.Expense, error) {
	expenses := []models.Expense{}
	tx := q.Where("user_id = ?", userID)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Order("expense_date DESC").Order("created_at DESC").Limit(limit).Offset(offset).Find(&expenses).Error; err != nil {
		return expenses, err
	}
	return expenses, nil
}

// CountExpenses returns the total expenses matching the category filter.
func (q *ExpenseQueries) CountExpenses(userID uuid.UUID, category string) (int64, error) {
	var total int64
	tx := q.Model(&models.Expense{}).Where("user_id = ?", userID)
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
	Total     float64
	ThisMonth float64
}

// GetExpenseTotals returns all-time and current-month sums. Month bounds are
// computed in Go (portable across SQLite and PostgreSQL).
func (q *ExpenseQueries) GetExpenseTotals(userID uuid.UUID, category string) (ExpenseTotals, error) {
	totals := ExpenseTotals{}
	tx := q.Model(&models.Expense{}).Where("user_id = ?", userID)
	if category != "" && category != "all" {
		tx = tx.Where("category = ?", category)
	}
	if err := tx.Select("COALESCE(SUM(amount), 0) AS total").Scan(&totals).Error; err != nil {
		return totals, err
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonth := monthStart.AddDate(0, 1, 0)
	monthTx := q.Model(&models.Expense{}).Where("user_id = ? AND expense_date >= ? AND expense_date < ?", userID, monthStart, nextMonth)
	if category != "" && category != "all" {
		monthTx = monthTx.Where("category = ?", category)
	}
	var thisMonth float64
	if err := monthTx.Select("COALESCE(SUM(amount), 0)").Scan(&thisMonth).Error; err != nil {
		return totals, err
	}
	totals.ThisMonth = thisMonth
	return totals, nil
}

// ExpenseCategories returns the distinct categories of a user for the filter.
func (q *ExpenseQueries) ExpenseCategories(userID uuid.UUID) ([]string, error) {
	categories := []string{}
	if err := q.Model(&models.Expense{}).Where("user_id = ?", userID).
		Distinct().Pluck("category", &categories).Error; err != nil {
		return categories, err
	}
	return categories, nil
}

// GetExpense returns one expense owned by a user.
func (q *ExpenseQueries) GetExpense(userID, id uuid.UUID) (models.Expense, error) {
	expense := models.Expense{}
	if err := q.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		return expense, notFound(err)
	}
	return expense, nil
}

// CreateExpense persists an expense.
func (q *ExpenseQueries) CreateExpense(expense *models.Expense) error {
	return q.Create(expense).Error
}

// UpdateExpense updates an expense owned by a user.
func (q *ExpenseQueries) UpdateExpense(expense *models.Expense) error {
	return q.Model(&models.Expense{}).Where("id = ? AND user_id = ?", expense.ID, expense.UserID).
		Updates(map[string]interface{}{
			"updated_at":   time.Now(),
			"vendor":       expense.Vendor,
			"category":     expense.Category,
			"expense_date": expense.ExpenseDate,
			"amount":       expense.Amount,
			"currency":     expense.Currency,
			"notes":        expense.Notes,
		}).Error
}

// DeleteExpense deletes an expense owned by a user.
func (q *ExpenseQueries) DeleteExpense(userID, id uuid.UUID) error {
	return q.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Expense{}).Error
}
