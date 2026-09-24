package controllers

import (
	"strings"

	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/gateway"
)

// routeIntentGateway resolves the provider for an intent:
//   - an explicit `gateway` pins that provider (legacy escape hatch);
//   - an explicit `payment_method` routes to any configured provider that
//     supports it, ignoring the project default (true multi-provider);
//   - otherwise the project default, then "midtrans".
func routeIntentGateway(input *models.IntentInput, projectDefault string) (gateway.Gateway, error) {
	if requested := strings.TrimSpace(input.Gateway); requested != "" {
		return gateway.Route(requested, input.PaymentMethod)
	}
	if strings.TrimSpace(input.PaymentMethod) != "" {
		return gateway.Route("", input.PaymentMethod)
	}
	return gateway.Route(resolveGateway("", projectDefault), "")
}
