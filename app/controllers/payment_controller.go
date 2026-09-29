package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"gorm.io/gorm"
)

func newPaymentToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func paymentResponse(row models.PaymentListRow) fiber.Map {
	return fiber.Map{
		"id":               row.PaymentID,
		"invoice_id":       row.InvoiceID,
		"invoice_number":   row.InvoiceNumber,
		"client_name":      row.ClientName,
		"invoice_currency": row.InvoiceCurrency,
		"amount":           row.Amount,
		"method":           row.Method,
		"paid_on":          utils.FormatDate(row.PaidOn),
		"txn_id":           row.TxnID,
		"notes":            row.Notes,
		"can_void":         row.CanVoid(),
	}
}

// ListPayments returns one page of payments plus global totals.
// @Description Get payments of current user.
// @Summary get payments
// @Tags Payments
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /payments [get]
func ListPayments(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	rows, err := db.ListPayments(orgID, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payments", nil)
	}
	total, err := db.CountPayments(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count payments", nil)
	}
	totals, err := db.GetPaymentTotals(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment totals", nil)
	}
	payments := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		payments = append(payments, paymentResponse(row))
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"payments": payments,
		"totals": fiber.Map{
			"total":     totals.Total,
			"thisMonth": totals.ThisMonth,
		},
		"meta": paging.Meta(total),
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
// @Security SessionCookie
// @Param Idempotency-Key header string false "Replay protection key (uuid per payment intent)"
// @Router /payments [post]
func CreatePayment(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
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
	if !input.Amount.GreaterThan(decimal.Zero) {
		return utils.Fail(c, fiber.StatusBadRequest, "amount must be greater than zero", nil)
	}
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
	}
	paidOn, err := utils.ParseDate(input.PaidOn)
	if err != nil || paidOn == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid paid_on, expected YYYY-MM-DD", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	invoice, err := db.GetInvoice(orgID, invoiceID)
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
	// Money in flight locks manual payments: a concurrent manual record would settle against a different balance than the Snap intent.
	if db.PendingInvoiceIDs(orgID)[invoiceID] {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
	}
	if paid.Add(input.Amount).GreaterThan(invoice.Total) {
		return utils.Fail(c, fiber.StatusBadRequest, "payment exceeds invoice balance", nil)
	}
	now := time.Now()
	payment := &models.Payment{
		ID:        uuid.New(),
		CreatedAt: now,
		UserID:    utils.CurrentActorID(c),
		OrgID:     orgID,
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
	// Auto-mark the invoice paid when payments now cover the total, so the
	// stored status column stays in sync (display also uses effective_status).
	if invoice.Status != models.InvoiceStatusPaid && paid.Add(input.Amount).GreaterThanOrEqual(invoice.Total) && invoice.Total.GreaterThan(decimal.Zero) {
		_ = db.UpdateInvoiceStatus(orgID, invoiceID, models.InvoiceStatusPaid)
	}
	recordAudit(c, db, utils.CurrentActorID(c), "payment.create", "payment", payment.ID.String(),
		`{"invoice_id":"`+invoiceID.String()+`","amount":"`+input.Amount.String()+`"}`)
	invalidateAggregates(c, orgID)
	enqueueOrgNotification(db, orgID, models.NotifEventPaymentCreated, "", paymentNotifData(db, *payment))
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"payment": fiber.Map{
		"id":         payment.ID,
		"invoice_id": payment.InvoiceID,
		"amount":     payment.Amount,
		"method":     payment.Method,
		"paid_on":    utils.FormatDate(payment.PaidOn),
		"txn_id":     payment.TxnID,
		"notes":      payment.Notes,
	}})
}

// VoidPayment voids a payment owned by the current user instead of deleting
// it. The row stays for audit but is excluded from every balance, list and
// aggregate query, so it disappears from the frontend.
// @Description Void a payment.
// @Summary void payment
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param reason query string false "Void reason (required, also accepted as JSON body {reason})"
// @Param request body models.PaymentVoidInput false "Void payload (alternative to query)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Param Idempotency-Key header string false "Replay protection key (uuid per void intent)"
// @Router /payments/{id} [delete]
func VoidPayment(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
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
	payment, err := db.GetPayment(orgID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "payment not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load payment", nil)
	}
	if payment.VoidedAt != nil {
		return utils.Fail(c, fiber.StatusConflict, "payment is already voided", nil)
	}
	// Gateway-settled payments are provider ledger entries: voiding them
	// here would silently diverge from the gateway's record.
	if !payment.CanVoid() {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "gateway payment cannot be voided", nil)
	}
	// A pending overlay locks money movement: voiding would change the
	// balance the live Snap intent settles against.
	if _, err := db.GetInvoice(orgID, payment.InvoiceID); err == nil {
		if db.PendingInvoiceIDs(orgID)[payment.InvoiceID] {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "invoice has a pending payment", nil)
		}
	}
	reason := strings.TrimSpace(c.Query("reason"))
	if reason == "" && len(c.Body()) > 0 {
		input := &models.PaymentVoidInput{}
		if err := c.Bind().Body(input); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
		}
		reason = strings.TrimSpace(input.Reason)
	}
	if reason == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "void reason is required", nil)
	}
	if err := utils.NewValidator().Var(reason, "lte=500"); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "void reason is too long", nil)
	}
	if err := db.VoidPayment(orgID, id, reason); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Fail(c, fiber.StatusConflict, "payment is already voided", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to void payment", nil)
	}
	// Reopen the invoice when a void drops payments below the total again.
	if invoice, err := db.GetInvoice(orgID, payment.InvoiceID); err == nil {
		if invoice.Status == models.InvoiceStatusPaid {
			if remaining, err := db.PaidAmount(payment.InvoiceID); err == nil && remaining.LessThan(invoice.Total) {
				_ = db.UpdateInvoiceStatus(orgID, payment.InvoiceID, models.InvoiceStatusSent)
			}
		}
	}
	meta, _ := json.Marshal(fiber.Map{
		"invoice_id": payment.InvoiceID.String(),
		"amount":     payment.Amount,
		"reason":     reason,
	})
	recordAudit(c, db, utils.CurrentActorID(c), "payment.void", "payment", id.String(), string(meta))
	voidData := paymentNotifData(db, payment)
	voidData["void_reason"] = reason
	enqueueOrgNotification(db, orgID, models.NotifEventPaymentVoided, "", voidData)
	invalidateAggregates(c, orgID)
	return utils.OK(c, fiber.StatusOK, fiber.Map{"payment": fiber.Map{
		"id":         payment.ID,
		"invoice_id": payment.InvoiceID,
		"voided":     true,
	}})
}
