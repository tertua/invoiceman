package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/platform/database"
)

// newQRImageStub serves a tiny png so the proxy test can verify the backend
// forwards provider bytes with a download disposition.
func newQRImageStub(t *testing.T, body []byte, contentType string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pointIntentAt rewrites the stored intent's PaymentURL so the proxy fetches
// the controlled image stub instead of the Core API stub URL.
func pointIntentAt(t *testing.T, orderID, paymentURL string) {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	stored, err := db.GetTransaction(orderID)
	require.NoError(t, err)
	stored.PaymentURL = paymentURL
	require.NoError(t, db.SaveTransaction(&stored))
}

// The proxy streams the provider QR image same-origin with an attachment
// disposition, so the browser saves the file instead of opening a new tab.
func TestPublicPayQrProxyDownloadsImage(t *testing.T) {
	core, _ := newQRISCoreStub(t, "2099-01-01 00:00:00")
	t.Setenv("MIDTRANS_SERVER_KEY", "qris-qr-key")
	t.Setenv("MIDTRANS_CORE_BASE_URL", core.URL)

	img := newQRImageStub(t, []byte{0x89, 0x50, 0x4e, 0x47}, "image/png")

	app := newTestApp()
	_, token := qrisFixture(t, app, "qris-qr@example.com")

	resp := doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"qris"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	first := decodeBody(t, resp)
	pointIntentAt(t, first["order_id"].(string), img.URL+"/qris.png")

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token+"/qr", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	assert.Equal(t, `attachment; filename="`+expectedQrFilename(first["order_id"].(string))+`.png"`, resp.Header.Get("Content-Disposition"))
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, raw)
}

// expectedQrFilename mirrors the controller's sanitizeExternal so the test
// asserts the exact download name without assuming the order id shape.
func expectedQrFilename(orderID string) string {
	var b strings.Builder
	for _, r := range orderID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 48 {
		out = out[:48]
	}
	if out == "" {
		out = "order"
	}
	return out
}

// renameIntentOrder rewrites a stored intent's order id to simulate a hostile
// invoice number flowing into the download filename.
func renameIntentOrder(t *testing.T, old, evil string) {
	t.Helper()
	db, err := database.OpenDBConnection()
	require.NoError(t, err)
	stored, err := db.GetTransaction(old)
	require.NoError(t, err)
	h := db.GatewayQueries.DB
	require.NoError(t, h.Where("order_id = ?", old).Delete(&models.GatewayTransaction{}).Error)
	stored.OrderID = evil
	require.NoError(t, db.CreateTransaction(&stored))
}

// A hostile order id (quotes, semicolons from a crafted invoice number)
// must still produce a header-safe filename, never breaking the header.
func TestPublicPayQrProxySanitizesOrderFilename(t *testing.T) {
	core, _ := newQRISCoreStub(t, "2099-01-01 00:00:00")
	t.Setenv("MIDTRANS_SERVER_KEY", "qris-qr-evil-key")
	t.Setenv("MIDTRANS_CORE_BASE_URL", core.URL)

	img := newQRImageStub(t, []byte{0x89, 0x50, 0x4e, 0x47}, "image/png")

	app := newTestApp()
	_, token := qrisFixture(t, app, "qris-qr-evil@example.com")

	resp := doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"qris"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	first := decodeBody(t, resp)
	evil := `PAY-"evil";x-1`
	renameIntentOrder(t, first["order_id"].(string), evil)
	pointIntentAt(t, evil, img.URL+"/qris.png")

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token+"/qr", "", nil)
	require.Equal(t, 200, resp.StatusCode)
	disp := resp.Header.Get("Content-Disposition")
	assert.Equal(t, `attachment; filename="PAY-evilx-1.png"`, disp)
	resp.Body.Close()
}

// An unknown token stays a 404 with the shared error envelope.
func TestPublicPayQrProxyUnknownToken(t *testing.T) {
	app := newTestApp()
	resp := doRequest(t, app, "GET", "/api/public/pay/nope-token/qr", "", nil)
	require.Equal(t, 404, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Contains(t, body["error"].(map[string]interface{})["message"], "payment link not found")
}

// Without a payable intent (expired QR) there is nothing to proxy.
func TestPublicPayQrProxyWithoutIntent(t *testing.T) {
	core, _ := newQRISCoreStub(t, "2000-01-01 00:00:00")
	t.Setenv("MIDTRANS_SERVER_KEY", "qris-qr-empty-key")
	t.Setenv("MIDTRANS_CORE_BASE_URL", core.URL)

	app := newTestApp()
	_, token := qrisFixture(t, app, "qris-qr-empty@example.com")

	resp := doRequest(t, app, "GET", "/api/public/pay/"+token+"/qr", "", nil)
	require.Equal(t, 404, resp.StatusCode)
	body := decodeBody(t, resp)
	assert.Contains(t, body["error"].(map[string]interface{})["message"], "qr image not found")
}

// A non-image upstream body is rejected so the proxy never serves HTML/JS.
func TestPublicPayQrProxyRejectsNonImage(t *testing.T) {
	core, _ := newQRISCoreStub(t, "2099-01-01 00:00:00")
	t.Setenv("MIDTRANS_SERVER_KEY", "qris-qr-html-key")
	t.Setenv("MIDTRANS_CORE_BASE_URL", core.URL)

	html := newQRImageStub(t, []byte(`<html>nope</html>`), "text/html")

	app := newTestApp()
	_, token := qrisFixture(t, app, "qris-qr-html@example.com")

	resp := doRequest(t, app, "POST", "/api/public/pay/"+token+"/transaction", `{"payment_method":"qris"}`, nil)
	require.Equal(t, 200, resp.StatusCode)
	first := decodeBody(t, resp)
	pointIntentAt(t, first["order_id"].(string), html.URL+"/qris.png")

	resp = doRequest(t, app, "GET", "/api/public/pay/"+token+"/qr", "", nil)
	require.Equal(t, 502, resp.StatusCode)
	resp.Body.Close()
}
