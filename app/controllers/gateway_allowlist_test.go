package controllers

import (
	"testing"

	"github.com/tertua/tupay/platform/gateway"
)

// fakeMethodsGateway is a test double that declares its default methods
// through the optional DefaultMethodsProvider capability.
type fakeMethodsGateway struct {
	gateway.Gateway
	methods []string
}

func (f fakeMethodsGateway) Name() string             { return "fake" }
func (f fakeMethodsGateway) DefaultMethods() []string { return f.methods }

// applyEnabledMethods must take the narrowing from the gateway itself, never
// from a provider name hardcoded in the controller.
func TestApplyEnabledMethodsUsesProviderCapability(t *testing.T) {
	spec := chargeSpec{}
	applyEnabledMethods(&spec, fakeMethodsGateway{methods: []string{gateway.MethodQRIS, gateway.MethodGopay}})
	if len(spec.EnabledMethods) != 2 || spec.EnabledMethods[0] != gateway.MethodQRIS || spec.EnabledMethods[1] != gateway.MethodGopay {
		t.Fatalf("EnabledMethods = %v, want [qris gopay]", spec.EnabledMethods)
	}
}

// A provider without the capability keeps its own defaults: nothing is applied.
func TestApplyEnabledMethodsWithoutCapability(t *testing.T) {
	spec := chargeSpec{}
	applyEnabledMethods(&spec, noCapabilityGateway{})
	if spec.EnabledMethods != nil {
		t.Fatalf("EnabledMethods = %v, want nil", spec.EnabledMethods)
	}
}

type noCapabilityGateway struct{ gateway.Gateway }

func (noCapabilityGateway) Name() string { return "plain" }

// methodAllowedFor: a nil allowlist means unrestricted, an empty method is the
// provider default and is always allowed, and a non-nil allowlist vetoes any
// method it does not list.
func TestMethodAllowedFor(t *testing.T) {
	allow := map[string]bool{gateway.MethodQRIS: true}
	cases := []struct {
		name     string
		provider string
		allow    map[string]bool
		method   string
		want     bool
	}{
		{"nil allowlist allows anything", "midtrans", nil, gateway.MethodGopay, true},
		{"empty method always allowed", "midtrans", allow, "", true},
		{"listed method allowed", "midtrans", allow, gateway.MethodQRIS, true},
		{"unlisted method vetoed", "midtrans", allow, gateway.MethodGopay, false},
		{"unlisted method vetoed on other provider", "other", allow, gateway.MethodGopay, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := methodAllowedFor(tc.provider, tc.allow, tc.method); got != tc.want {
				t.Errorf("methodAllowedFor(%q, %v, %q) = %v, want %v", tc.provider, tc.allow, tc.method, got, tc.want)
			}
		})
	}
}
