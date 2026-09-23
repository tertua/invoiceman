package controllers

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"github.com/tertua/invoiceman/platform/storage"
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
		"receipt_url":  receiptProxyURL(expense),
	}
}

// receiptProxyURL returns the authenticated proxy path for an attached
// receipt, or "" when none is attached.
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
// @Security SessionCookie
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
	invalidateAggregates(c, userID)
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
	invalidateAggregates(c, userID)
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
	invalidateAggregates(c, userID)
	return c.SendStatus(fiber.StatusNoContent)
}

// maxReceiptUploadSize caps stored receipt attachments.
const maxReceiptUploadSize = 10 << 20

// receiptContentType sniffs the uploaded content; only raster images and
// PDFs are accepted. SVG is rejected even though it sniffs as image/*:
// stored SVG executes scripts in the viewer's origin when proxied.
func receiptContentType(header, sniffed string) (string, bool) {
	ct := sniffed
	if ct == "" || ct == "application/octet-stream" {
		ct = header
	}
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "image/svg+xml" {
		return "", false
	}
	if strings.HasPrefix(ct, "image/") || ct == "application/pdf" {
		return ct, true
	}
	return "", false
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
	expense, err := db.GetExpense(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "expense not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense", nil)
	}
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is required", nil)
	}
	if file.Size > maxReceiptUploadSize {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt file is too large", nil)
	}
	reader, err := file.Open()
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "failed to read receipt file", nil)
	}
	defer func() { _ = reader.Close() }()
	ct, ok := receiptContentType(file.Header.Get("Content-Type"), sniffContentType(reader))
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "receipt must be an image or PDF", nil)
	}
	store, err := storage.Shared()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "file storage is not configured", nil)
	}
	key := storage.ReceiptKey(userID.String(), id.String(), extForContentType(ct))
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
	recordAudit(c, db, userID, "expense.receipt.upload", "expense", id.String(), "")
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
	expense, err := db.GetExpense(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "expense not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense", nil)
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
	expense, err := db.GetExpense(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "expense not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load expense", nil)
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
	recordAudit(c, db, userID, "expense.receipt.delete", "expense", id.String(), "")
	return c.SendStatus(fiber.StatusNoContent)
}

// sniffContentType reads the first bytes for mime sniffing. The reader is
// left consumed; callers must Seek back before uploading.
func sniffContentType(r io.Reader) string {
	head := make([]byte, 512)
	n, _ := io.ReadFull(r, head)
	return http.DetectContentType(head[:n])
}

func extForContentType(ct string) string {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
