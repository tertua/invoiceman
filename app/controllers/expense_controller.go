package controllers

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
)

func parseExpenseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, value)
}

func expenseResponse(expense models.Expense) fiber.Map {
	return fiber.Map{
		"id":           expense.ID,
		"vendor":       expense.Vendor,
		"category":     expense.Category,
		"expense_date": expense.ExpenseDate.Format(dateLayout),
		"amount":       expense.Amount,
		"currency":     expense.Currency,
		"notes":        expense.Notes,
	}
}

// ListExpenses returns expenses, categories, and totals for the current user.
// @Description Get expenses of current user.
// @Summary get expenses
// @Tags Expenses
// @Produce json
// @Param category query string false "Filter by category"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /expenses [get]
func ListExpenses(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	all, err := db.ListExpenses(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expenses", nil)
	}

	category := c.Query("category")
	categories := make([]string, 0)
	seen := make(map[string]struct{})
	for _, expense := range all {
		if _, ok := seen[expense.Category]; !ok {
			seen[expense.Category] = struct{}{}
			categories = append(categories, expense.Category)
		}
	}

	now := time.Now()
	total := 0.0
	thisMonth := 0.0
	expenses := make([]fiber.Map, 0, len(all))
	for _, expense := range all {
		if category != "" && category != "all" && expense.Category != category {
			continue
		}
		total += expense.Amount
		if expense.ExpenseDate.Year() == now.Year() && expense.ExpenseDate.Month() == now.Month() {
			thisMonth += expense.Amount
		}
		expenses = append(expenses, expenseResponse(expense))
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"expenses":   expenses,
		"categories": categories,
		"totals": fiber.Map{
			"total":     total,
			"thisMonth": thisMonth,
		},
	})
}

// CreateExpense creates an expense for the current user.
// @Description Create an expense.
// @Summary create expense
// @Tags Expenses
// @Accept json
// @Produce json
// @Param request body models.ExpenseInput true "Expense payload"
// @Success 201 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /expenses [post]
func CreateExpense(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.ExpenseInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	expenseDate, err := parseExpenseDate(input.ExpenseDate)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense_date, expected YYYY-MM-DD", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	currency := input.Currency
	if currency == "" {
		currency = settings.Currency
	}
	now := time.Now()
	expense := &models.Expense{
		ID:          uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   &now,
		UserID:      userID,
		Vendor:      input.Vendor,
		Category:    input.Category,
		ExpenseDate: expenseDate,
		Amount:      input.Amount,
		Currency:    currency,
		Notes:       input.Notes,
	}
	if err := db.CreateExpense(expense); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create expense", nil)
	}
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"expense": expenseResponse(*expense)})
}

// UpdateExpense updates an expense owned by the current user.
// @Description Update an expense.
// @Summary update expense
// @Tags Expenses
// @Accept json
// @Produce json
// @Param id path string true "Expense ID"
// @Param request body models.ExpenseInput true "Expense payload"
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /expenses/{id} [patch]
func UpdateExpense(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	input := &models.ExpenseInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	expenseDate, err := parseExpenseDate(input.ExpenseDate)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense_date, expected YYYY-MM-DD", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	settings, err := db.GetSettings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	expense, err := db.GetExpense(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "expense not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense", nil)
	}
	expense.Vendor = input.Vendor
	expense.Category = input.Category
	expense.ExpenseDate = expenseDate
	expense.Amount = input.Amount
	expense.Currency = input.Currency
	if expense.Currency == "" {
		expense.Currency = settings.Currency
	}
	expense.Notes = input.Notes
	if err := db.UpdateExpense(&expense); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update expense", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"expense": expenseResponse(expense)})
}

// DeleteExpense deletes an expense owned by the current user.
// @Description Delete an expense.
// @Summary delete expense
// @Tags Expenses
// @Produce json
// @Param id path string true "Expense ID"
// @Success 204 {string} status "ok"
// @Security ApiKeyAuth
// @Router /expenses/{id} [delete]
func DeleteExpense(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if _, err := db.GetExpense(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "expense not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense", nil)
	}
	if err := db.DeleteExpense(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete expense", nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
