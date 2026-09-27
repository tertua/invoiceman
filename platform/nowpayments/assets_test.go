package nowpayments

import (
	"os"
	"regexp"
	"slices"
	"testing"
)

func TestSupportedPayCurrencies(t *testing.T) {
	for _, code := range []string{"usdttrc20", "usdterc20", "usdtbsc", "trx", "doge", "ltc", "USDTTRC20", " Trx "} {
		if !IsSupportedPayCurrency(code) {
			t.Fatalf("expected %q to be supported", code)
		}
	}
	for _, code := range []string{"", "usdc", "btc", "usdtttc20"} {
		if IsSupportedPayCurrency(code) {
			t.Fatalf("expected %q to be rejected", code)
		}
	}
	if NormalizePayCurrency(" USDTTRC20 ") != "usdttrc20" {
		t.Fatalf("normalize = %q, want %q", NormalizePayCurrency(" USDTTRC20 "), "usdttrc20")
	}
	if NormalizePayCurrency(" USDTBSC ") != DefaultPayCurrency {
		t.Fatalf("normalize BSC = %q, want default %q", NormalizePayCurrency(" USDTBSC "), DefaultPayCurrency)
	}
}

// TestPayCurrencyAllowlistMatchesWeb guards the frontend mirror: the asset
// picker in web/src/lib/cryptoAssets.js must list exactly the pay currencies
// the backend will accept, in the same order.
func TestPayCurrencyAllowlistMatchesWeb(t *testing.T) {
	raw, err := os.ReadFile("../../web/src/lib/cryptoAssets.js")
	if err != nil {
		t.Fatalf("read cryptoAssets.js: %v", err)
	}
	matches := regexp.MustCompile(`code:\s*"([^"]+)"`).FindAllStringSubmatch(string(raw), -1)
	web := make([]string, 0, len(matches))
	for _, m := range matches {
		web = append(web, m[1])
	}
	if !slices.Equal(web, supportedPayCurrencies) {
		t.Fatalf("web crypto assets %v != backend allowlist %v (keep web/src/lib/cryptoAssets.js in sync)", web, supportedPayCurrencies)
	}
}
