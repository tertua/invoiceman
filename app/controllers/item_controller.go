package controllers

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
)

// ListItems returns one page of catalog items owned by the current user.
// @Description Get catalog items of current user.
// @Summary get catalog items
// @Tags Items
// @Param page query int false "Page number (default 1)"
// @Param per_page query int false "Items per page (default 20, max 100)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /items [get]
func ListItems(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	paging := utils.ParsePagination(c)
	items, err := db.ListItems(orgID, paging.Limit(), paging.Offset())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load items", nil)
	}
	total, err := db.CountItems(orgID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count items", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"items": items, "meta": paging.Meta(total)})
}

// CreateItem creates a catalog item for the current user.
// @Description Create a catalog item.
// @Summary create catalog item
// @Tags Items
// @Accept json
// @Produce json
// @Param request body models.ItemInput true "Item payload"
// @Success 201 {object} map[string]interface{}
// @Security SessionCookie
// @Router /items [post]
func CreateItem(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	input := &models.ItemInput{}
	if err := c.Bind().Body(input); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid request body", nil)
	}
	if err := utils.NewValidator().Struct(input); err != nil {
		return utils.ValidationFailed(c, err)
	}
	if input.Rate.IsNegative() {
		return utils.Fail(c, fiber.StatusBadRequest, "rate cannot be negative", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	now := time.Now()
	item := &models.Item{
		ID:          uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   &now,
		UserID:      utils.CurrentActorID(c),
		OrgID:       orgID,
		Name:        input.Name,
		Description: input.Description,
		Rate:        input.Rate, Unit: input.Unit,
	}
	if err := db.CreateItem(item); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to create item", nil)
	}
	return utils.OK(c, fiber.StatusCreated, fiber.Map{"item": item})
}

// UpdateItem updates a catalog item owned by the current user.
// @Description Update a catalog item.
// @Summary update catalog item
// @Tags Items
// @Accept json
// @Produce json
// @Param id path string true "Item ID"
// @Param request body models.ItemInput true "Item payload"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /items/{id} [patch]
func UpdateItem(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid item id", nil)
	}
	input := &models.ItemInput{}
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
	item, err := db.GetItem(orgID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "item not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load item", nil)
	}
	item.Name = input.Name
	item.Description = input.Description
	item.Rate = input.Rate
	item.Unit = input.Unit
	if err := db.UpdateItem(&item); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to update item", nil)
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"item": item})
}

// DeleteItem deletes a catalog item owned by the current user.
// @Description Delete a catalog item.
// @Summary delete catalog item
// @Tags Items
// @Produce json
// @Param id path string true "Item ID"
// @Success 204 {string} status "ok"
// @Security SessionCookie
// @Router /items/{id} [delete]
func DeleteItem(c fiber.Ctx) error {
	orgID, err := utils.CurrentOrgID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid item id", nil)
	}
	db, err := database.OpenDBConnection()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "database connection error", nil)
	}
	if _, err := db.GetItem(orgID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.Fail(c, fiber.StatusNotFound, "item not found", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to load item", nil)
	}
	if err := db.DeleteItem(orgID, id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete item", nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
