package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/pkg/utils"
)

// currentUserOrg resolves the session user and the active org that stamp new invoice rows.
func currentUserOrg(c fiber.Ctx) (orgID, userID uuid.UUID, err error) {
	userID, err = utils.CurrentUserID(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	orgID, err = utils.CurrentOrgID(c)
	return orgID, userID, err
}
