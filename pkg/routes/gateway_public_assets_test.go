package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cryptoAssetFixture struct {
	app     *fiber.App
	cookies []*http.Cookie
	n       int
}

func setupCryptoAssetFixture(t *testing.T, tag string) *cryptoAssetFixture {
	t.Helper()
	npServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/payment" {
			var body struct {
				PayCurrency string `json:"pay_currency"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payment_id":"pay_` + body.PayCurrency + `","pay_address":"ADDR_` + body.PayCurrency + `","pay_amount":"100","pay_currency":"` + body.PayCurrency + `","price_amount":"100","price_currency":"USD","order_id":"INV-1"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"inv_assets","invoice_url":"https://nowpayments.io/payment/?iid=assets"}`))
	}))
	t.Cleanup(npServer.Close)

	t.Setenv("MIDTRANS_SERVER_KEY", "assets-server-key")
	t.Setenv("NOWPAYMENTS_API_KEY", "assets-np-key")
	t.Setenv("NOWPAYMENTS_BASE_URL", npServer.URL+"/v1")
	t.Setenv("NOWPAYMENTS_SANDBOX", "true")

	app := newTestApp()

	resp := doRequest(t, app, "POST", "/api/auth/register",
		`{"name":"Assets MP","email":"assets-`+tag+`@example.com","password":"secret123"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, app, "POST", "/api/auth/login",
		`{"email":"assets-`+tag+`@example.com","password":"secret123"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	cookies := resp.Cookies()
	resp.Body.Close()

	resp = doRequest(t, app, "PATCH", "/api/settings",
		`{"company_name":"Assets MP","currency":"USD","tax_rate":0,"invoice_prefix":"INV-","usd_to_idr":"18000"}`, cookies)
	require.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()

	return &cryptoAssetFixture{app: app, cookies: cookies}
}

func (f *cryptoAssetFixture) newToken(t *testing.T) string {
	t.Helper()
	f.n++
	spec := newInvoice()
	spec.ClientID, spec.Currency = createClient(t, f.app, f.cookies, "Assets Payer"), "USD"
	spec.Items = []invoiceLine{{Description: "Service", Quantity: 1, Rate: "100"}}
	invoiceID := createInvoiceID(t, f.app, f.cookies, spec)
	resp := doRequest(t, f.app, "POST", "/api/payments/online", `{"invoiceId":"`+invoiceID+`"}`, f.cookies)
	require.Equal(t, 200, resp.StatusCode)
	return decodeBody(t, resp)["token"].(string)
}

func (f *cryptoAssetFixture) transact(t *testing.T, token, body string) *http.Response {
	t.Helper()
	return doRequest(t, f.app, "POST", "/api/public/pay/"+token+"/transaction", body, nil)
}

func TestPublicPayCryptoAssets(t *testing.T) {
	f := setupCryptoAssetFixture(t, "multi")

	for _, asset := range []string{"usdttrc20", "usdterc20", "usdtbep20", "trx", "doge", "ltc"} {
		token := f.newToken(t)
		resp := f.transact(t, token, `{"payment_method":"crypto","pay_currency":"`+asset+`"}`)
		require.Equal(t, 200, resp.StatusCode, "asset %s", asset)
		intent := decodeBody(t, resp)
		assert.Equal(t, "ADDR_"+asset, intent["address"], "asset %s", asset)
		assert.Equal(t, asset, intent["pay_currency"], "asset %s", asset)
		assert.Empty(t, intent["payment_url"], "asset %s stays on page", asset)
		resp.Body.Close()
	}
}

func TestPublicPayCryptoSwitchReopensAsset(t *testing.T) {
	f := setupCryptoAssetFixture(t, "switch")
	token := f.newToken(t)

	resp := f.transact(t, token, `{"payment_method":"crypto","pay_currency":"usdttrc20"}`)
	first := decodeBody(t, resp)
	resp.Body.Close()
	require.Equal(t, "ADDR_usdttrc20", first["address"])
	firstOrder := first["order_id"]

	resp = f.transact(t, token, `{"payment_method":"crypto","pay_currency":"doge"}`)
	second := decodeBody(t, resp)
	resp.Body.Close()
	require.Equal(t, "ADDR_doge", second["address"])
	require.Equal(t, "doge", second["pay_currency"])
	require.NotEqual(t, firstOrder, second["order_id"])

	resp = f.transact(t, token, `{"payment_method":"crypto","pay_currency":"usdttrc20"}`)
	third := decodeBody(t, resp)
	resp.Body.Close()
	require.Equal(t, "ADDR_usdttrc20", third["address"], "switching back reuses the first asset's intent")
	require.Equal(t, firstOrder, third["order_id"])
}

func TestPublicPayCryptoUnsupportedAsset(t *testing.T) {
	f := setupCryptoAssetFixture(t, "unsupported")
	token := f.newToken(t)

	resp := f.transact(t, token, `{"payment_method":"crypto","pay_currency":"usdc"}`)
	assert.Equal(t, 400, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Equal(t, "unsupported payment method", body["error"].(map[string]any)["message"])
	resp.Body.Close()
}
