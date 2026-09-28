package nowpayments

import "github.com/tertua/tupay/platform/gateway"

func (Gateway) Methods() []string { return []string{gateway.MethodCrypto} }

// Configured reports whether the NOWPayments API key is present.
func (Gateway) Configured() bool { return FromEnv().APIKey != "" }

// Sandbox reports whether NOWPayments runs against its sandbox API.
func (Gateway) Sandbox() bool { return FromEnv().Sandbox }
