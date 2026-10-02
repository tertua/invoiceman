package controllers

import (
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ListInvoices returns one page of invoices of the current user.
// @Description Get invoices of current user.
// @Summary get invoices of current user
// @Tags Invoices
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param search query string false "Search by number or client"
// @Param sort query string false "Sort column"
// @Param order query string false "Sort order"
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /invoices [get]
func ListInvoices(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, ok := openDB(c)
	if !ok {
		return nil
	}

	status, search := c.Query("status"), c.Query("search")
	paging := utils.ParsePagination(c)
	rows, err := db.ListInvoices(
		orgID,
		status,
		search,
		c.Query("sort"),
		c.Query("order"),
		paging.Limit(),
		paging.Offset(),
	)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load invoices", nil)
	}
	total, err := db.CountInvoices(orgID, status, search)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count invoices", nil)
	}

	invoices := make([]invoiceListRow, 0, len(rows))
	pending := db.PendingInvoiceIDs(orgID)
	for _, row := range rows {
		invoices = append(invoices, invoiceListRow{
			ID:              row.ID,
			InvoiceNumber:   row.InvoiceNumber,
			ClientName:      row.ClientName,
			ClientCompany:   row.ClientCompany,
			IssueDate:       utils.FormatDate(row.IssueDate),
			DueDate:         utils.FormatDate(row.DueDate),
			Total:           row.Total,
			Currency:        row.Currency,
			EffectiveStatus: row.EffectiveStatus(pending[row.ID]),
		})
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoices": invoices, "meta": paging.Meta(total)})
}

// invoiceListRow is one invoice row in the invoice listing.
type invoiceListRow struct {
	ID              uuid.UUID    `json:"id"`
	InvoiceNumber   string       `json:"invoice_number"`
	ClientName      string       `json:"client_name"`
	ClientCompany   string       `json:"client_company"`
	IssueDate       string       `json:"issue_date"`
	DueDate         string       `json:"due_date"`
	Total           models.Money `json:"total"`
	Currency        string       `json:"currency"`
	EffectiveStatus string       `json:"effective_status"`
}
