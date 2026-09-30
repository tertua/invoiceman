package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/gateway"
)

// The per-provider allowlist resolver lives in gateway_allowlist.go
// (providerAllowlist / methodAllowedFor / applyEnabledMethods /
// normalizeProviderMethods). This file only holds the settings handler.

// ListSettingsMethods returns the methods an owner can enable for a provider,
// provider-neutral, for the settings UI. It enumerates through the registry
// (never a provider package) and mirrors gateway.MethodName so the dashboard
// labels match the public pay page. An optional ?gateway= selects the provider
// and defaults to gateway.DefaultProvider().
// @Description List selectable payment methods for the settings UI.
// @Summary list payment methods
// @Tags Settings
// @Produce json
// @Param gateway query string false "Provider name (defaults to the built-in provider)"
// @Success 200 {object} map[string]interface{}
// @Security SessionCookie
// @Router /settings/methods [get]
func ListSettingsMethods(c fiber.Ctx) error {
	name := strings.ToLower(strings.TrimSpace(c.Query("gateway")))
	if name == "" {
		name = gateway.DefaultProvider()
	}
	methods := make([]models.GatewayMethod, 0)
	gw, err := gateway.Get(name)
	if err != nil {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
	}
	provider, ok := gw.(gateway.PaymentMethodProvider)
	if !ok {
		return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
	}
	for _, method := range provider.Methods() {
		methods = append(methods, models.GatewayMethod{ID: method, Name: gateway.MethodName(method)})
	}
	return utils.OK(c, fiber.StatusOK, fiber.Map{"methods": methods})
}
