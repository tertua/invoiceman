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

func expenseResponse(expense models.Expense) fiber.Map {
	return fiber.Map{
		"id":           expense.ID,
		"vendor":       expense.Vendor,
		"category":     expense.Category,
		"expense_date": utils.FormatTime(expense.ExpenseDate),
		"amount":       expense.Amount,
		"currency":     expense.Currency,
		"notes":        expense.Notes,
	}
}

// ListExpenses returns one page of expenses plus global totals.
// Totals and categories always cover the whole category filter, not just
// the current page.
// @Description Get expenses of current user.
// @Summary get expenses
// @Tags Expenses
// @Produce json
// @Param category query string false "Filter by category"
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
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
	category := c.Query("category")
	paging := utils.ParsePagination(c)
	page, err := db.ListExpenses(userID, category, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expenses", nil)
	}
	total, err := db.CountExpenses(userID, category)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count expenses", nil)
	}
	totals, err := db.GetExpenseTotals(userID, category)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense totals", nil)
	}
	categories, err := db.ExpenseCategories(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense categories", nil)
	}

	expenses := make([]fiber.Map, 0, len(page))
	for _, expense := range page {
		expenses = append(expenses, expenseResponse(expense))
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"expenses":   expenses,
		"categories": categories,
		"totals": fiber.Map{
			"total":     totals.Total,
			"thisMonth": totals.ThisMonth,
		},
		"meta": paging.Meta(total),
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
	expenseDate, err := utils.ParseRequiredDate(input.ExpenseDate)
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
	expenseDate, err := utils.ParseRequiredDate(input.ExpenseDate)
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
