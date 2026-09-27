package nowpayments

import "strings"

const DefaultPayCurrency = "usdttrc20"

// supportedPayCurrencies is the ordered allowlist of NOWPayments pay
// currencies the on-page crypto widget may request. It is the backend source
// of truth for web/src/lib/cryptoAssets.js; TestPayCurrencyAllowlistMatchesWeb
// fails if the two drift.
var supportedPayCurrencies = []string{"usdttrc20", "usdterc20", "usdtbep20", "trx", "doge", "ltc"}

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
