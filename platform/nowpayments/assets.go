package nowpayments

import "strings"

const DefaultPayCurrency = "usdtbsc"

// supportedPayCurrencies is the ordered allowlist of NOWPayments pay
// currencies the on-page crypto widget may request. It is the backend source
// of truth for webui/src/lib/cryptoAssets.js; TestPayCurrencyAllowlistMatchesWeb
// fails if the two drift.
var supportedPayCurrencies = []string{"usdttrc20", "usdterc20", "usdtbsc", "trx", "doge", "ltc"}

func NormalizePayCurrency(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func IsSupportedPayCurrency(raw string) bool {
	c := NormalizePayCurrency(raw)
	for _, s := range supportedPayCurrencies {
		if s == c {
			return true
		}
	}
	return false
}

// payCurrencyOrDefault maps "the payer has not picked an asset yet" onto the
// canonical payout currency, so an unqualified request (a minimum check, a
// legacy client) still has a currency to ask the provider for.
func payCurrencyOrDefault(raw string) string {
	if c := NormalizePayCurrency(raw); c != "" {
		return c
	}
	return DefaultPayCurrency
}
