package nowpayments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tertua/invoiceman/pkg/constants"
	"github.com/tertua/invoiceman/platform/gateway"
)

// DefaultPayCurrency is the on-page crypto pay currency (USDT TRC20). The
// live minimum is checked against this currency; it is the single swap point
// if the widget ever moves to another network.
const DefaultPayCurrency = "usdttRC20"

// minAmountTTL bounds how long a fetched minimum is trusted. Minimums move
// with market rates, but every pay-page load must not become a live API call
// (rate limits, latency). One hour keeps the list honest without hammering.
const minAmountTTL = time.Hour

type minCacheEntry struct {
	amount  decimal.Decimal
	expires time.Time
}

var minCache = struct {
	sync.Mutex
	entries map[string]minCacheEntry
}{entries: make(map[string]minCacheEntry)}

// MinAmount implements gateway.MinAmountChecker: the live minimum charge in
// currencyFrom for a DefaultPayCurrency payout, served from a TTL cache.
func (Gateway) MinAmount(ctx context.Context, currencyFrom string) (decimal.Decimal, error) {
	return minAmount(ctx, FromEnv(), currencyFrom, DefaultPayCurrency)
}

func minAmount(ctx context.Context, cfg Config, currencyFrom, currencyTo string) (decimal.Decimal, error) {
	if cfg.APIKey == "" {
		return decimal.Zero, ErrNotConfigured
	}
	from := strings.ToLower(strings.TrimSpace(currencyFrom))
	to := strings.ToLower(strings.TrimSpace(currencyTo))
	key := cfg.BaseURL() + "|" + from + "|" + to

	minCache.Lock()
	if e, ok := minCache.entries[key]; ok && time.Now().Before(e.expires) {
		minCache.Unlock()
		return e.amount, nil
	}
	minCache.Unlock()

	amount, err := fetchMinAmount(ctx, cfg, from, to)
	if err != nil {
		return decimal.Zero, err
	}
	minCache.Lock()
	minCache.entries[key] = minCacheEntry{amount: amount, expires: time.Now().Add(minAmountTTL)}
	minCache.Unlock()
	return amount, nil
}

// fetchMinAmount calls GET /v1/min-amount (auth required) and parses
// {"min_amount": <number>}. Unknown charge currencies answer 404.
func fetchMinAmount(ctx context.Context, cfg Config, currencyFrom, currencyTo string) (decimal.Decimal, error) {
	endpoint := cfg.BaseURL() + "/min-amount?currency_from=" + url.QueryEscape(currencyFrom) + "&currency_to=" + url.QueryEscape(currencyTo)
	ctx, cancel := context.WithTimeout(ctx, constants.GatewayAPITimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return decimal.Zero, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", constants.UserAgent())
	httpReq.Header.Set("x-api-key", cfg.APIKey)
	resp, err := constants.DefaultHTTPClient.Do(httpReq)
	if err != nil {
		return decimal.Zero, fmt.Errorf("nowpayments min-amount: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, constants.MaxAPIResponseSize))
	if err != nil {
		return decimal.Zero, fmt.Errorf("nowpayments min-amount: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decimal.Zero, fmt.Errorf("nowpayments min-amount: %w", gateway.NewProviderError("nowpayments", resp.StatusCode, truncate(string(raw), constants.MaxErrorBodyLog)))
	}
	var decoded struct {
		MinAmount json.Number `json:"min_amount"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return decimal.Zero, fmt.Errorf("nowpayments min-amount decode: %w", err)
	}
	amount, err := decimal.NewFromString(decoded.MinAmount.String())
	if err != nil {
		return decimal.Zero, fmt.Errorf("nowpayments min-amount decode: %w", err)
	}
	return amount, nil
}

// resetMinAmountCache drops cached minimums; tests only.
func resetMinAmountCache() {
	minCache.Lock()
	defer minCache.Unlock()
	minCache.entries = make(map[string]minCacheEntry)
}

var _ gateway.MinAmountChecker = Gateway{}
