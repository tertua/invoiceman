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

// ListExpenses returns expenses owned by a user.
func (q *ExpenseQueries) ListExpenses(userID uuid.UUID) ([]models.Expense, error) {
	expenses := []models.Expense{}
	if err := q.Where("user_id = ?", userID).Order("expense_date DESC").Order("created_at DESC").Find(&expenses).Error; err != nil {
		return expenses, err
	}
	return expenses, nil
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
