package controllers

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/events"
)

// StreamEvents opens a server-sent events stream for the current user.
// The server pushes one frame per aggregate change (payment recorded or
// voided, invoice created or settled) so the MPA bell refreshes without
// polling; clients refetch dashboard state on any frame. Heartbeats keep
// idle proxies from closing the stream, retry tells EventSource how fast
// to reconnect. No timeout middleware here by design: the connection is
// meant to stay open (see WithAITimeout, which this route skips).
// D11: this SSE stream stays per-user — org fan-out repeats the per-member publish, it never widens one shared stream.
// @Description Live aggregate-change events for the current user.
// @Summary subscribe to live events
// @Tags Events
// @Produce text/event-stream
// @Success 200 {string} string "event stream"
// @Security SessionCookie
// @Router /events [get]
func StreamEvents(c fiber.Ctx) error {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "unauthorized, please sign in again", nil)
	}
	ch, unsub := events.Default.Subscribe(userID.String())

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	done := c.Context().Done()
	// NOTE: the writer below runs after this handler returns (fasthttp
	// invokes body stream writers while sending), so unsub must be
	// deferred inside it — deferring here would drop the subscription
	// before the first frame is even written.
	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer unsub()
		if _, err := fmt.Fprintf(w, "retry: 5000\n\n"); err != nil {
			return
		}
		_ = w.Flush()
		beat := time.NewTicker(25 * time.Second)
		defer beat.Stop()
		for {
			select {
			case <-done:
				return
			case ev := <-ch:
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, ev.Data); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-beat.C:
				if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}
