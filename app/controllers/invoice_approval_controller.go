package controllers

import (
	"database/sql"
	"errors"

	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// approvalTarget carries the shared load result (org, id, handle, row) for the three approval endpoints.
type approvalTarget struct {
	orgID    uuid.UUID
	id       uuid.UUID
	db       *database.Queries
	existing models.Invoice
}

// approvalLoad resolves org, invoice id, handle and row; a nil return means the failure response is already written.
func approvalLoad(c fiber.Ctx) *approvalTarget {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		_ = utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
		return nil
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		_ = utils.Fail(c, fiber.StatusBadRequest, "invalid invoice id", nil)
		return nil
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		_ = utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
		return nil
	}
	existing, err := db.GetInvoice(orgID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = utils.Fail(c, fiber.StatusNotFound, "invoice not found", nil)
		} else {
			_ = utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
		}
		return nil
	}
	return &approvalTarget{orgID: orgID, id: id, db: db, existing: existing}
}

// requireOwner rejects approve/reject callers that do not hold the org owner role.
func requireOwner(c fiber.Ctx) error {
	role, err := utils.CurrentOrgRole(c)
	if err != nil || role != models.RoleOwner {
		return ErrOwnerRequired
	}
	return nil
}

// approvalDetail answers with the freshly reloaded invoice after a status write.
func approvalDetail(c fiber.Ctx, target *approvalTarget) error {
	detail, err := invoiceDetail(*target.db, target.orgID, target.id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoice", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoice": detail})
}

// SubmitInvoice moves a draft invoice into the approval queue; staff and owner may submit.
// @Description Submit a draft invoice for approval.
// @Summary submit invoice for approval
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id}/submit [post]
func SubmitInvoice(c fiber.Ctx) error {
	target := approvalLoad(c)
	if target == nil {
		return nil
	}
	if target.existing.Status != models.InvoiceStatusDraft {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid status transition", nil)
	}
	if err := target.db.UpdateInvoiceStatus(target.orgID, target.id, models.InvoiceStatusPending); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}
	recordAudit(c, target.db, utils.CurrentActorID(c), "invoice.submitted", "invoice", target.id.String(), "")
	// Submit fans out on the existing invoice.status_updated event instead of inventing a second one.
	enqueueOrgNotification(target.db, target.orgID, models.NotifEventInvoiceStatusUpdated, "",
		invoiceNotifData(target.db, target.orgID, target.id))
	invalidateAggregates(c, target.orgID)
	return approvalDetail(c, target)
}

// ApproveInvoice marks a pending invoice as sent; only the org owner may approve and the send rule needs a client.
// @Description Approve a pending invoice.
// @Summary approve invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id}/approve [post]
func ApproveInvoice(c fiber.Ctx) error {
	target := approvalLoad(c)
	if target == nil {
		return nil
	}
	if err := requireOwner(c); err != nil {
		return failInvoiceRule(c, err)
	}
	if target.existing.Status != models.InvoiceStatusPending {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid status transition", nil)
	}
	if target.existing.ClientID == nil {
		return utils.Fail(c, fiber.StatusBadRequest, "client is required to send an invoice", nil)
	}
	if err := target.db.UpdateInvoiceStatus(target.orgID, target.id, models.InvoiceStatusSent); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}
	recordAudit(c, target.db, utils.CurrentActorID(c), "invoice.approved", "invoice", target.id.String(), "")
	enqueueOrgNotification(target.db, target.orgID, models.NotifEventInvoiceStatusUpdated, "",
		invoiceNotifData(target.db, target.orgID, target.id))
	invalidateAggregates(c, target.orgID)
	return approvalDetail(c, target)
}

// RejectInvoice sends a pending invoice back to draft; only the org owner may reject.
// @Description Reject a pending invoice.
// @Summary reject invoice
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices/{id}/reject [post]
func RejectInvoice(c fiber.Ctx) error {
	target := approvalLoad(c)
	if target == nil {
		return nil
	}
	if err := requireOwner(c); err != nil {
		return failInvoiceRule(c, err)
	}
	if target.existing.Status != models.InvoiceStatusPending {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid status transition", nil)
	}
	if err := target.db.UpdateInvoiceStatus(target.orgID, target.id, models.InvoiceStatusDraft); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update invoice status", nil)
	}
	recordAudit(c, target.db, utils.CurrentActorID(c), "invoice.rejected", "invoice", target.id.String(), "")
	// Rejection reaches only the submitter's own inbox (D11), not every org member.
	enqueueNotification(target.db, target.existing.UserID, models.NotifEventInvoiceStatusUpdated, "",
		invoiceNotifData(target.db, target.orgID, target.id))
	invalidateAggregates(c, target.orgID)
	return approvalDetail(c, target)
}
