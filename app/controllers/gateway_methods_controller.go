package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/gateway"
)

// ListGatewayMethods returns provider-agnostic payment methods available to
// the authenticated downstream project.
// @Description List payment methods without exposing provider names.
// @Summary list payment methods
// @Tags Gateway
// @Produce json
// @Success 200 {object} models.GatewayMethodsResponse
// @Router /gateway/methods [get]
func ListGatewayMethods(c fiber.Ctx) error {
	refs := gateway.OfferedMethods()
	methods := make([]models.GatewayMethod, 0, len(refs))
	for _, ref := range refs {
		methods = append(methods, models.GatewayMethod{ID: ref.ID, Name: gateway.MethodName(ref.ID)})
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
}
