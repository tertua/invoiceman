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

func paymentResponse(row models.PaymentListRow) fiber.Map {
	return fiber.Map{
		"id":               row.PaymentID,
		"invoice_id":       row.InvoiceID,
		"invoice_number":   row.InvoiceNumber,
		"client_name":      row.ClientName,
		"invoice_currency": row.InvoiceCurrency,
		"amount":           row.Amount,
		"method":           row.Method,
		"paid_on":          formatDate(row.PaidOn),
		"txn_id":           row.TxnID,
		"notes":            row.Notes,
	}
}

// ListPayments returns payments and totals for the current user.
// @Description Get payments of current user.
// @Summary get payments
// @Tags Payments
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /payments [get]
func ListPayments(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	rows, err := db.ListPayments(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payments", nil)
	}
	now := time.Now()
	total := 0.0
	thisMonth := 0.0
	payments := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		total += row.Amount
		if row.PaidOn != nil && row.PaidOn.Year() == now.Year() && row.PaidOn.Month() == now.Month() {
			thisMonth += row.Amount
		}
		payments = append(payments, paymentResponse(row))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"payments": payments,
		"totals": fiber.Map{
			"total":     total,
			"thisMonth": thisMonth,
		},
	})
}

// CreatePayment records a payment against an invoice owned by the current user.
// @Description Record a payment.
// @Summary create payment
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body models.PaymentInput true "Payment payload"
// @Success 201 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /payments [post]
func CreatePayment(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.PaymentInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	paidOn, err := parseDate(input.PaidOn)
	if err != nil || paidOn == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid paid_on, expected YYYY-MM-DD", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(userID, invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	paid, err := db.PaidAmount(invoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice payments", nil)
	}
	if paid+input.Amount > invoice.Total {
		return utils.Fail(c, fiber.StatusBadRequest, "payment exceeds invoice balance", nil)
	}
	now := time.Now()
	payment := &models.Payment{
		ID:        uuid.New(),
		CreatedAt: now,
		UserID:    userID,
		InvoiceID: invoiceID,
		Amount:    input.Amount,
		Method:    input.Method,
		PaidOn:    paidOn,
		TxnID:     input.TxnID,
		Notes:     input.Notes,
	}
	if err := db.CreatePayment(payment); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create payment", nil)
	}
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"payment": fiber.Map{
		"id":         payment.ID,
		"invoice_id": payment.InvoiceID,
		"amount":     payment.Amount,
		"method":     payment.Method,
		"paid_on":    formatDate(payment.PaidOn),
		"txn_id":     payment.TxnID,
		"notes":      payment.Notes,
	}})
}

// DeletePayment deletes a payment owned by the current user.
// @Description Delete a payment.
// @Summary delete payment
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 204 {string} status "ok"
// @Security ApiKeyAuth
// @Router /payments/{id} [delete]
func DeletePayment(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid payment id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if _, err := db.GetPayment(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "payment not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment", nil)
	}
	if err := db.DeletePayment(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete payment", nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
