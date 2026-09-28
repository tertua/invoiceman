package controllers

import (
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"

	"github.com/gofiber/fiber/v3"
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
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	status, search := c.Query("status"), c.Query("search")
	paging := utils.ParsePagination(c)
	rows, err := db.ListInvoices(
		userID,
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
	total, err := db.CountInvoices(userID, status, search)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count invoices", nil)
	}

	invoices := make([]fiber.Map, 0, len(rows))
	pending := db.PendingInvoiceIDs(userID)
	for _, row := range rows {
		invoices = append(invoices, fiber.Map{
			"id":               row.ID,
			"invoice_number":   row.InvoiceNumber,
			"client_name":      row.ClientName,
			"client_company":   row.ClientCompany,
			"issue_date":       utils.FormatDate(row.IssueDate),
			"due_date":         utils.FormatDate(row.DueDate),
			"total":            row.Total,
			"currency":         row.Currency,
			"effective_status": row.EffectiveStatus(pending[row.ID]),
		})
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"invoices": invoices, "meta": paging.Meta(total)})
}
