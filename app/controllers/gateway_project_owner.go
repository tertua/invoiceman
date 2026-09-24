package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/invoiceman/pkg/utils"
)

func currentGatewayOwner(c fiber.Ctx) (uuid.UUID, error) {
	return utils.CurrentUserID(c)
}
