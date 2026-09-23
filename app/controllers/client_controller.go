package controllers

import (
	"database/sql"
	"errors"
	"time"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ListClients returns one page of clients of the current user.
// @Description Get all clients of current user.
// @Summary get all clients of current user
// @Tags Clients
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /clients [get]
func ListClients(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	paging := utils.ParsePagination(c)
	clients, err := db.ListClients(userID, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load clients", nil)
	}
	total, err := db.CountClients(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count clients", nil)
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{"clients": clients, "meta": paging.Meta(total)})
}

// GetClient returns one client with invoices and stats.
// @Description Get client by ID with invoices and stats.
// @Summary get client by ID with invoices and stats
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /clients/{id} [get]
func GetClient(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid client id", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	client, err := db.GetClient(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
	}

	rows, err := db.ClientInvoices(userID, id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client invoices", nil)
	}

	invoices := make([]fiber.Map, 0, len(rows))
	var totalBilled, paidTotal float64
	for _, row := range rows {
		paid := row.PaidAmount
		// Billed invoices are sent + paid; drafts are not billed yet.
		if row.Status == models.InvoiceStatusSent || row.Status == models.InvoiceStatusPaid {
			totalBilled += row.Total
			paidTotal += paid
		}
		invoices = append(invoices, fiber.Map{
			"id":               row.ID,
			"invoice_number":   row.InvoiceNumber,
			"issue_date":       utils.FormatDate(row.IssueDate),
			"due_date":         utils.FormatDate(row.DueDate),
			"total":            row.Total,
			"currency":         row.Currency,
			"status":           row.Status,
			"effective_status": row.EffectiveStatus(),
			"paid_amount":      paid,
			"balance":          row.Total - paid,
		})
	}

	stats := models.ClientStats{
		Count:       len(rows),
		TotalBilled: totalBilled,
		Outstanding: totalBilled - paidTotal,
	}

	return utils.OK(c, fiber.StatusOK, fiber.Map{
		"client":   client,
		"invoices": invoices,
		"stats":    stats,
	})
}

// CreateClient creates a new client.
// @Description Create a new client.
// @Summary create a new client
// @Tags Clients
// @Accept json
// @Produce json
// @Param request body models.ClientInput true "Create client payload"
// @Success 201 {object} map[string]interface{}
// @Security SessionCookie
// @Router /clients [post]
func CreateClient(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	input := &models.ClientInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	now := time.Now()
	client := &models.Client{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: &now,
		UserID:    userID,
		Name:      input.Name,
		Email:     input.Email,
		Company:   input.Company,
		Phone:     input.Phone,
		Address:   input.Address,
		Notes:     input.Notes,
	}
	if err := db.CreateClient(client); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create client", nil)
	}
	invalidateAggregates(c, userID)

	return utils.OK(c, fiber.StatusCreated, fiber.Map{"client": client})
}

// UpdateClient updates a client.
// @Description Update a client.
// @Summary update a client
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param request body models.ClientInput true "Update client payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /clients/{id} [patch]
func UpdateClient(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid client id", nil)
	}

	input := &models.ClientInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	client, err := db.GetClient(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
	}

	now := time.Now()
	client.UpdatedAt = &now
	client.Name = input.Name
	client.Email = input.Email
	client.Company = input.Company
	client.Phone = input.Phone
	client.Address = input.Address
	client.Notes = input.Notes

	if err := db.UpdateClient(&client); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update client", nil)
	}
	invalidateAggregates(c, userID)

	return utils.OK(c, fiber.StatusOK, fiber.Map{"client": client})
}

// DeleteClient deletes a client.
// @Description Delete a client.
// @Summary delete a client
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 204 {string} status "ok"
// @Security SessionCookie
// @Router /clients/{id} [delete]
func DeleteClient(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid client id", nil)
	}

	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}

	if _, err := db.GetClient(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "client not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load client", nil)
	}

	if err := db.DeleteClient(userID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete client", nil)
	}
	recordAudit(c, db, userID, "client.delete", "client", id.String(), "")
	invalidateAggregates(c, userID)

	return c.SendStatus(fiber.StatusNoContent)
}
