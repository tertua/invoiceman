package gateway

import (
	"context"
	"errors"
	"testing"
)

type stubGateway struct{ name string }

func (s stubGateway) Name() string { return s.name }

func (s stubGateway) CreateTransaction(ctx context.Context, req *CreateTxRequest) (*CreateTxResponse, error) {
	return &CreateTxResponse{PaymentURL: "https://pay.example/" + req.OrderID}, nil
}

func (s stubGateway) ParseAndVerify(raw []byte) (*NotificationResult, error) {
	return &NotificationResult{OrderID: "x", Status: StatusSuccess}, nil
}

func TestRegistry(t *testing.T) {
	Register(stubGateway{name: "stubcoin"})
	g, err := Get("stubcoin")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.Name() != "stubcoin" {
		t.Fatalf("unexpected gateway: %s", g.Name())
	}
	if _, err := Get("nope"); !errors.Is(err, ErrUnknownGateway) {
		t.Fatalf("expected ErrUnknownGateway, got %v", err)
	}
	if len(Names()) == 0 {
		t.Fatal("expected registered names")
	}
}
