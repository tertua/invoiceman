package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/utils"
)

// DeleteProject removes a gateway project that has never been used.
// @Description Delete an unused gateway project.
// @Summary delete gateway project
// @Tags Admin
// @Produce json
// @Param slug path string true "Project slug"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /admin/gateway/projects/{slug} [delete]
func DeleteProject(c fiber.Ctx) error {
	db, ok := openDB(c)
	if !ok {
		return nil
	}
	p, err := db.GetProjectBySlug(c.Params("slug"))
	if err != nil {
		return utils.NotFoundOrFailed(c, err, "project")
	}
	total, err := db.CountTransactionsByProject(p.Slug)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to count transactions", nil)
	}
	if total > 0 {
		return utils.Fail(c, fiber.StatusConflict, "project has transactions, disable it instead", nil)
	}
	if err := db.DeleteProject(p.Slug); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "failed to delete project", nil)
	}
	if adminID := utils.CurrentActorID(c); adminID != uuid.Nil {
		recordAudit(c, db, adminID, "gateway.project.delete", "project", p.Slug, "")
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"message": "project deleted"})
}
