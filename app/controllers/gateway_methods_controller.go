package controllers

import (
	"sort"

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
	seen := make(map[string]bool)
	methods := make([]models.GatewayMethod, 0)
	for _, name := range gateway.Names() {
		provider, err := gateway.Get(name)
		if err != nil {
			continue
		}
		capability, ok := provider.(gateway.PaymentMethodProvider)
		if !ok {
			continue
		}
		if configured, ok := provider.(gateway.ConfiguredProvider); ok && !configured.Configured() {
			continue
		}
		for _, method := range capability.Methods() {
			if seen[method] {
				continue
			}
			seen[method] = true
			methods = append(methods, models.GatewayMethod{ID: method, Name: gateway.MethodName(method)})
		}
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].ID < methods[j].ID })
	return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
}
