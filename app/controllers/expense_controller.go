package controllers

import (
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/storage"
)

func expenseResponse(expense models.Expense) expenseRow {
	return expenseRow{
		ID:          expense.ID,
		Vendor:      expense.Vendor,
		Category:    expense.Category,
		ExpenseDate: utils.FormatTime(expense.ExpenseDate),
		Amount:      expense.Amount,
		Currency:    expense.Currency,
		Notes:       expense.Notes,
		ReceiptURL:  receiptProxyURL(expense),
	}
}

// expenseRow is the expense payload used by list and mutation responses.
type expenseRow struct {
	ID          uuid.UUID    `json:"id"`
	Vendor      string       `json:"vendor"`
	Category    string       `json:"category"`
	ExpenseDate string       `json:"expense_date"`
	Amount      models.Money `json:"amount"`
	Currency    string       `json:"currency"`
	Notes       string       `json:"notes"`
	ReceiptURL  string       `json:"receipt_url"`
}

// expenseListResponse wraps one page of expenses with the global filter data.
type expenseListResponse struct {
	Expenses   []expenseRow `json:"expenses"`
	Categories []string     `json:"categories"`
	Totals     fiber.Map    `json:"totals"`
	Meta       fiber.Map    `json:"meta"`
}

// receiptProxyURL returns the authenticated proxy path for an attached receipt, or "" when none is attached.
func receiptProxyURL(expense models.Expense) string {
	if strings.TrimSpace(expense.ReceiptURL) == "" {
		return ""
	}
	return "/api/v1/expenses/" + expense.ID.String() + "/receipt"
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
// @Security SessionCookie
// @Router /expenses [get]
func ListExpenses(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	category := c.Query("category")
	paging := utils.ParsePagination(c)
	page, err := db.ListExpenses(orgID, category, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expenses", nil)
	}
	total, err := db.CountExpenses(orgID, category)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count expenses", nil)
	}
	totals, err := db.GetExpenseTotals(orgID, category)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense totals", nil)
	}
	categories, err := db.ExpenseCategories(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense categories", nil)
	}

	expenses := make([]expenseRow, 0, len(page))
	for _, expense := range page {
		expenses = append(expenses, expenseResponse(expense))
	}

	return utils.OK(c, fiber.StatusOK, expenseListResponse{
		Expenses:   expenses,
		Categories: categories,
		Totals: fiber.Map{
			"total":     totals.Total,
			"thisMonth": totals.ThisMonth,
		},
		Meta: paging.Meta(total),
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
// @Security SessionCookie
// @Router /expenses [post]
func CreateExpense(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
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
	if input.Amount.IsNegative() {
		return utils.Fail(c, fiber.StatusBadRequest, "amount cannot be negative", nil)
	}
	expenseDate, err := utils.ParseRequiredDate(input.ExpenseDate)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense_date, expected YYYY-MM-DD", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	settings, err := db.GetSettings(orgID)
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
		UserID:      utils.CurrentActorID(c),
		OrgID:       orgID,
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
	invalidateAggregates(c, orgID)
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
// @Security SessionCookie
// @Router /expenses/{id} [patch]
func UpdateExpense(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
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
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	settings, err := db.GetSettings(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load settings", nil)
	}
	expense, err := db.GetExpense(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "expense")
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
	invalidateAggregates(c, orgID)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"expense": expenseResponse(expense)})
}

// DeleteExpense deletes an expense owned by the current user.
// @Description Delete an expense.
// @Summary delete expense
// @Tags Expenses
// @Produce json
// @Param id path string true "Expense ID"
// @Success 204 {string} status "ok"
// @Security SessionCookie
// @Router /expenses/{id} [delete]
func DeleteExpense(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	if _, err := db.GetExpense(orgID, id); err != nil {
		return utils.NotFoundOrFailed(c, err, "expense")
	}
	if err := db.DeleteExpense(orgID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete expense", nil)
	}
	invalidateAggregates(c, orgID)
	return c.SendStatus(fiber.StatusNoContent)
}

// UploadReceipt stores a receipt attachment for an expense owned by the
// current user, replacing any previous one.
// @Description Upload an expense receipt.
// @Summary upload expense receipt
// @Tags Expenses
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Expense ID"
// @Param file formData file true "Receipt image or PDF (max 10MB)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /expenses/{id}/receipt [post]
func UploadReceipt(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	expense, err := db.GetExpense(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "expense")
	}
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is required", nil)
	}
	const maxReceiptUploadSize = 10 << 20 // local constant, kept for clarity
	if file.Size > maxReceiptUploadSize {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is too large", nil)
	}
	reader, err := file.Open()
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read receipt file", nil)
	}
	defer func() { _ = reader.Close() }()
	ct, ext, ok := utils.ValidateImage(reader, utils.ReceiptAllowedTypes)
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt must be an image or PDF", nil)
	}
	store, err := storage.Shared()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "file storage is not configured", nil)
	}
	key := storage.ReceiptKey(orgID.String(), id.String(), ext)
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read receipt file", nil)
	}
	if err := store.Put(c.Context(), key, reader, file.Size, ct); err != nil {
		return utils.Fail(c, fiber.StatusBadGateway, "failed to store receipt file", nil)
	}
	if old := strings.TrimSpace(expense.ReceiptURL); old != "" && old != key {
		_ = store.Delete(c.Context(), old)
	}
	expense.ReceiptURL = key
	if err := db.UpdateExpense(&expense); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to save receipt", nil)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "expense.receipt.upload", "expense", id.String(), "")
	return utils.OK(c, fiber.StatusOK, fiber.Map{"expense": expenseResponse(expense)})
}

// GetReceipt streams the attached receipt through an authenticated,
// ownership-checked endpoint (works for local and S3 backends).
// @Description Download an expense receipt.
// @Summary download expense receipt
// @Tags Expenses
// @Produce octet-stream
// @Param id path string true "Expense ID"
// @Success 200 {file} binary
// @Security SessionCookie
// @Router /expenses/{id}/receipt [get]
func GetReceipt(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	expense, err := db.GetExpense(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "expense")
	}
	if strings.TrimSpace(expense.ReceiptURL) == "" {
		return utils.Fail(c, fiber.StatusNotFound, "receipt not found", nil)
	}
	store, err := storage.Shared()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "file storage is not configured", nil)
	}
	rc, ct, err := store.Get(c.Context(), expense.ReceiptURL)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "receipt not found", nil)
	}
	// No Close here: the stream is sent after this handler returns and
	// fasthttp closes body streams once written.
	// Legacy SVG receipts (blocked for new uploads) must never render
	// inline — force download so embedded scripts can't execute.
	if strings.HasSuffix(strings.ToLower(expense.ReceiptURL), ".svg") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "image/svg") {
		c.Set("Content-Type", "application/octet-stream")
		c.Set("Content-Disposition", `attachment; filename="receipt"`)
	} else {
		c.Set("Content-Type", ct)
	}
	return c.SendStream(rc)
}

// DeleteReceipt removes the attached receipt of an expense.
// @Description Delete an expense receipt.
// @Summary delete expense receipt
// @Tags Expenses
// @Produce json
// @Param id path string true "Expense ID"
// @Success 204 {string} status "ok"
// @Security SessionCookie
// @Router /expenses/{id}/receipt [delete]
func DeleteReceipt(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid expense id", nil)
	}
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	expense, err := db.GetExpense(orgID, id)
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "expense")
	}
	if key := strings.TrimSpace(expense.ReceiptURL); key != "" {
		if store, serr := storage.Shared(); serr == nil {
			_ = store.Delete(c.Context(), key)
		}
		expense.ReceiptURL = ""
		if err := db.UpdateExpense(&expense); err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "failed to remove receipt", nil)
		}
	}
	recordAudit(c, db, utils.CurrentActorID(c), "expense.receipt.delete", "expense", id.String(), "")
	return c.SendStatus(fiber.StatusNoContent)
}
