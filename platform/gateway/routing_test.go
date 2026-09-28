package gateway

import (
	"context"
	"testing"
)

type routingGateway struct {
	name    string
	methods []string
}

func (g routingGateway) Name() string      { return g.name }
func (g routingGateway) Methods() []string { return g.methods }
func (g routingGateway) CreateTransaction(context.Context, *CreateTxRequest) (*CreateTxResponse, error) {
	return &CreateTxResponse{}, nil
}
func (g routingGateway) ParseAndVerify([]byte) (*NotificationResult, error) { return nil, nil }

func TestRouteSelectsProviderByMethod(t *testing.T) {
	Register(routingGateway{name: "route-test-bank", methods: []string{MethodBankTransfer}})
	t.Cleanup(func() {
		mu.Lock()
		delete(registry, "route-test-bank")
		mu.Unlock()
	})

	selected, err := Route("", MethodBankTransfer, nil)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if selected.Name() != "route-test-bank" && selected.Name() != "midtrans" {
		t.Fatalf("Route() selected %q", selected.Name())
	}
}

func TestRouteRejectsUnsupportedMethod(t *testing.T) {
	Register(routingGateway{name: "route-test-bank", methods: []string{MethodBankTransfer}})
	Register(routingGateway{name: "route-test-crypto", methods: []string{MethodCrypto}})
	t.Cleanup(func() {
		mu.Lock()
		delete(registry, "route-test-bank")
		delete(registry, "route-test-crypto")
		mu.Unlock()
	})
	if _, err := Route("route-test-bank", MethodCrypto, nil); err != ErrUnsupportedPaymentMethod {
		t.Fatalf("Route() error = %v, want %v", err, ErrUnsupportedPaymentMethod)
	}
}

func TestRouteHonorsPreferredProvider(t *testing.T) {
	Register(routingGateway{name: "route-test-preferred", methods: []string{MethodBankTransfer}})
	t.Cleanup(func() {
		mu.Lock()
		delete(registry, "route-test-preferred")
		mu.Unlock()
	})
	selected, err := Route("route-test-preferred", MethodBankTransfer, nil)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if selected.Name() != "route-test-preferred" {
		t.Fatalf("Route() selected %q, want the preferred provider", selected.Name())
	}
}

func TestRouteIsDeterministicWithoutPreferred(t *testing.T) {
	for _, name := range []string{"route-test-z", "route-test-a"} {
		Register(routingGateway{name: name, methods: []string{MethodCreditCard}})
	}
	t.Cleanup(func() {
		mu.Lock()
		delete(registry, "route-test-z")
		delete(registry, "route-test-a")
		mu.Unlock()
	})
	selected, err := Route("", MethodCreditCard, nil)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if selected.Name() != "route-test-a" {
		t.Fatalf("Route() selected %q, want route-test-a", selected.Name())
	}
}
