package controllers

import (
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/gateway"
)

// routeIntentGateway resolves the provider from the explicit request, then the
// project default, then "midtrans", and routes the requested payment method to
// a provider that supports it.
func routeIntentGateway(input *models.IntentInput, projectDefault string) (gateway.Gateway, error) {
	return gateway.Route(resolveGateway(input.Gateway, projectDefault), input.PaymentMethod)
}
