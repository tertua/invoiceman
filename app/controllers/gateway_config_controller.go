package controllers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/gateway"
)

// GatewayConfig returns public browser configuration for a gateway. Server
// credentials are never exposed here: only values a provider declares through
// the optional gateway.PayerConfigProvider capability (e.g. the public client
// key and the environment flag required by its browser SDK) are merged in.
// ?gateway=<name> selects the provider; it defaults to gateway.DefaultProvider().
// @Description Get public payment gateway browser configuration.
// @Summary get gateway browser config
// @Tags Gateway
// @Produce json
// @Param gateway query string false "Provider name (default: configured default gateway)"
// @Success 200 {object} map[string]interface{}
// @Router /gateway/config [get]
func GatewayConfig(c fiber.Ctx) error {
	name := strings.ToLower(strings.TrimSpace(c.Query("gateway")))
	if name == "" {
		name = gateway.DefaultProvider()
	}
	gw, err := gateway.Get(name)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "unknown payment gateway", nil)
	}
	out := fiber.Map{"gateway": name, "configured": gateway.ProviderReady(gw)}
	for k, v := range payerConfigFor(name) {
		out[k] = v
	}
	return utils.OK(c, fiber.StatusOK, out)
}

// payerConfigFor returns a gateway's browser-safe payer configuration (a
// provider's PayerConfig map, or empty when it declares none). It is shared by
// the public pay payload and /gateway/config so no controller hardcodes a
// provider. Junk names yield an empty map (never an error): the public payload
// is best-effort, while GatewayConfig validates the name first.
func payerConfigFor(gwName string) fiber.Map {
	gw, err := gateway.Get(strings.ToLower(strings.TrimSpace(gwName)))
	if err != nil {
		return fiber.Map{}
	}
	provider, ok := gw.(gateway.PayerConfigProvider)
	if !ok {
		return fiber.Map{}
	}
	out := fiber.Map{}
	for k, v := range provider.PayerConfig() {
		out[k] = v
	}
	return out
}
