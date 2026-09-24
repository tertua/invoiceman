package nowpayments

import "github.com/tertua/invoiceman/platform/gateway"

func (Gateway) Methods() []string { return []string{gateway.MethodCrypto} }

// Configured reports whether the NOWPayments API key is present.
func (Gateway) Configured() bool { return FromEnv().APIKey != "" }
