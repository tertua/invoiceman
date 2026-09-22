package mail

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderTemplates ensures both mails render with branding and links.
func TestRenderTemplates(t *testing.T) {
	reset, err := Render("reset_password", TemplateData{
		AppName: "Invoiceman", Name: "Ada", URL: "https://app.example.com/reset-password?token=abc",
	})
	require.NoError(t, err)
	assert.Contains(t, reset, "Reset your Invoiceman password")
	assert.Contains(t, reset, "https://app.example.com/reset-password?token=abc")

	link, err := Render("payment_link", TemplateData{
		AppName: "Invoiceman", URL: "https://app.example.com/pay/tok", InvoiceNumber: "INV-1",
	})
	require.NoError(t, err)
	assert.Contains(t, link, "Pay invoice INV-1")
	assert.Contains(t, link, "https://app.example.com/pay/tok")

	_, err = Render("missing", TemplateData{})
	require.Error(t, err)
}

// TestRenderEscapesInput ensures user-controlled fields cannot inject HTML.
func TestRenderEscapesInput(t *testing.T) {
	out, err := Render("reset_password", TemplateData{
		AppName: "Invoiceman", Name: `<script>alert("x")</script>`, URL: "https://x.example.com/",
	})
	require.NoError(t, err)
	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
}

// TestBuildMessageMultipart verifies the text+HTML envelope.
func TestBuildMessageMultipart(t *testing.T) {
	raw, err := buildMessage("from@example.com", "to@example.com", "Subject", "plain body", "<p>html body</p>")
	require.NoError(t, err)
	msg := string(raw)
	assert.Contains(t, msg, "multipart/alternative")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assert.Contains(t, msg, "Content-Type: text/html; charset=UTF-8")
	assert.Contains(t, msg, "plain body")
	assert.Contains(t, msg, "<p>html body</p>")
}

// TestBuildMessageTextOnly keeps the legacy single-part shape.
func TestBuildMessageTextOnly(t *testing.T) {
	raw, err := buildMessage("from@example.com", "to@example.com", "Subject", "plain body", "")
	require.NoError(t, err)
	msg := string(raw)
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assert.NotContains(t, msg, "multipart/alternative")
	assert.True(t, strings.HasSuffix(msg, "plain body"))
}
