package models

// GatewayMethod is one provider-neutral payment method a downstream project
// can request; the provider name is intentionally not exposed.
type GatewayMethod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GatewayMethodsResponse is the payload for GET /gateway/methods.
type GatewayMethodsResponse struct {
	Methods []GatewayMethod `json:"methods"`
}
