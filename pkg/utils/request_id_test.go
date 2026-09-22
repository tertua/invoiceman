package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func testRequestID(t *testing.T, a *fiber.App) string {
	t.Helper()
	var seen string
	a.Get("/", func(c fiber.Ctx) error {
		seen = RequestID(c)
		return c.SendStatus(200)
	})
	resp, err := a.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return seen
}

func TestRequestID(t *testing.T) {
	with := fiber.New()
	with.Use(requestid.New())
	if got := testRequestID(t, with); got == "" {
		t.Error("expected non-empty request id with middleware")
	}

	without := fiber.New()
	if got := testRequestID(t, without); got != "" {
		t.Errorf("expected empty request id without middleware, got %q", got)
	}

	if RequestID(nil) != "" {
		t.Error("expected empty request id for nil ctx")
	}
	if RequestLogger(nil) == nil {
		t.Error("expected non-nil logger for nil ctx")
	}
}
