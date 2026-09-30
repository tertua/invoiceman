package nowpayments

import "github.com/tertua/tupay/platform/gateway"

func (Gateway) Methods() []string { return []string{gateway.MethodCrypto} }

// Configured reports whether the NOWPayments API key is present.
func (Gateway) Configured() bool { return FromEnv().APIKey != "" }

// Sandbox reports whether NOWPayments runs against its sandbox API.
func (Gateway) Sandbox() bool { return FromEnv().Sandbox }

// ChargeCurrency reports that NOWPayments invoices are always charged in USD:
// the hosted checkout rejects non-USD fiat, so callers convert first.
func (Gateway) ChargeCurrency() string { return gateway.FiatUSD }

// RequiresDecimalAmount reports that NOWPayments needs a decimal amount
// because crypto amounts are not whole minor units.
func (Gateway) RequiresDecimalAmount() bool { return true }

// BrowserSDK reports that the stored token is a hosted payment-page id, not a
// browser widget token, so payloads must not expose it as "snap_token".
func (Gateway) BrowserSDK() bool { return false }
