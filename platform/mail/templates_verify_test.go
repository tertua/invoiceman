package mail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderVerifyEmail covers the verification template added for the
// pending-registration flow.
func TestRenderVerifyEmail(t *testing.T) {
	verify, err := Render("verify_email", TemplateData{
		AppName: "Invoiceman", Name: "Ada", URL: "https://app.example.com/verify-email?token=abc",
	})
	require.NoError(t, err)
	assert.Contains(t, verify, "Verify your Invoiceman email")
	assert.Contains(t, verify, "https://app.example.com/verify-email?token=abc")
}
