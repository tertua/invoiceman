package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/invoiceman/app/models"
	"github.com/tertua/invoiceman/pkg/configs"
	"github.com/tertua/invoiceman/pkg/logger"
	"github.com/tertua/invoiceman/pkg/utils"
	"github.com/tertua/invoiceman/platform/database"
	"gorm.io/gorm"
)

// Idempotency-Key replay protection for mutating payment routes.
//
// Problem: a retried POST (double-click, timeout, FE retry) executes again
// and creates a second payment/transaction. Fix: the client sends a unique
// key per payment intent; the first execution stores its response snapshot
// for IdempotencyTTLHours, repeats replay the snapshot, and a reused key
// with a different payload is rejected with 422.
//
// The middleware is fail-open on infrastructure errors (a downed lookup
// must not block payments) and only activates when the header is present,
// so existing clients are unaffected.
const (
	idempotencyHeader   = "Idempotency-Key"
	idempotencyMaxKey   = 255
	idempotencyMaxBody  = 64 << 10 // snapshots larger than this are not stored
	idempotencyPollWait = 5 * time.Second
	idempotencyPollStep = 100 * time.Millisecond
)

// IdempotencyScope extracts the dedup scope for a request
// (e.g. "user:<id>", "project:<slug>", "pay:<token>").
type IdempotencyScope func(c fiber.Ctx) (string, error)

// SessionIdempotencyScope scopes keys to the authenticated session user.
// AuthRequired must run before this middleware.
func SessionIdempotencyScope(c fiber.Ctx) (string, error) {
	userID, err := utils.CurrentUserID(c)
	if err != nil {
		return "", err
	}
	return "user:" + userID.String(), nil
}

// GatewayIdempotencyScope scopes keys to the calling service project.
// GatewayAuth must run before this middleware.
func GatewayIdempotencyScope(c fiber.Ctx) (string, error) {
	project, err := utils.CurrentServiceProject(c)
	if err != nil {
		return "", err
	}
	return "project:" + project.Slug, nil
}

// PublicPayIdempotencyScope scopes keys to the public payment token.
func PublicPayIdempotencyScope(c fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "payment token is required")
	}
	return "pay:" + token, nil
}

// Idempotency dedups mutating requests by the Idempotency-Key header.
func Idempotency(scope IdempotencyScope) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawKey := strings.TrimSpace(c.Get(idempotencyHeader))
		if rawKey == "" {
			return c.Next()
		}
		if len(rawKey) > idempotencyMaxKey {
			return utils.Fail(c, fiber.StatusBadRequest, "idempotency key is too long", nil)
		}

		scopeValue, err := scope(c)
		if err != nil {
			logger.L().Warn("idempotency scope failed, proceeding without dedup", "err", err)
			return c.Next()
		}

		db, err := database.OpenDBConnection()
		if err != nil {
			logger.L().Warn("idempotency lookup skipped, database unavailable", "err", err)
			return c.Next()
		}

		keyHash := hashString(rawKey)
		reqHash := hashString(c.Method() + "\n" + c.Path() + "\n" + string(c.Body()))

		rec, err := db.GetIdempotencyKey(keyHash, scopeValue)
		if err == nil {
			return replayOrWait(c, db, rec, reqHash)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.L().Warn("idempotency lookup failed, proceeding without dedup", "err", err)
			return c.Next()
		}

		// First sighting: plant a processing placeholder so a concurrent
		// retry with the same key loses the race and replays the winner.
		rec = models.IdempotencyKey{
			KeyHash:     keyHash,
			Scope:       scopeValue,
			Method:      c.Method(),
			Path:        c.Path(),
			RequestHash: reqHash,
			ExpiresAt:   time.Now().Add(time.Duration(configs.Get().Idempotency.TTLHours) * time.Hour),
			CreatedAt:   time.Now(),
		}
		if err := db.CreateIdempotencyPlaceholder(&rec); err != nil {
			// Lost the race (or an expired row): load the winner and replay.
			if winner, werr := db.GetIdempotencyKey(keyHash, scopeValue); werr == nil {
				return replayOrWait(c, db, winner, reqHash)
			}
			logger.L().Warn("idempotency placeholder failed, proceeding without dedup", "err", err)
			return c.Next()
		}

		// Execute, then snapshot the response for replays.
		if err := c.Next(); err != nil {
			_ = db.DeleteIdempotencyKey(rec.ID)
			return err
		}
		status := c.Response().StatusCode()
		body := append([]byte(nil), c.Response().Body()...)
		if status >= 500 || len(body) > idempotencyMaxBody {
			// Server errors stay retryable; oversized bodies are not cached.
			_ = db.DeleteIdempotencyKey(rec.ID)
			return nil
		}
		rec.StatusCode = status
		rec.ResponseBody = string(body)
		if err := db.CompleteIdempotencyKey(&rec); err != nil {
			logger.L().Warn("idempotency snapshot failed", "err", err)
		}
		return nil
	}
}

// replayOrWait replays a completed record, waits briefly for an in-flight
// one, or rejects a key reused with a different payload.
func replayOrWait(c fiber.Ctx, db *database.Queries, rec models.IdempotencyKey, reqHash string) error {
	if !rec.ExpiresAt.After(time.Now()) {
		_ = db.DeleteIdempotencyKey(rec.ID)
		return c.Next()
	}
	if rec.RequestHash != reqHash {
		return utils.Fail(c, fiber.StatusUnprocessableEntity,
			"idempotency key already used with a different request", nil)
	}
	if rec.StatusCode == 0 {
		// Another request is executing right now; wait for its snapshot.
		deadline := time.Now().Add(idempotencyPollWait)
		for time.Now().Before(deadline) {
			time.Sleep(idempotencyPollStep)
			updated, err := db.GetIdempotencyKey(rec.KeyHash, rec.Scope)
			if err != nil {
				break
			}
			rec = updated
			if rec.StatusCode != 0 {
				break
			}
		}
		if rec.StatusCode == 0 {
			return utils.Fail(c, fiber.StatusConflict,
				"conflicting request with the same idempotency key is in progress", nil)
		}
	}
	c.Set("Idempotent-Replayed", "true")
	c.Set("Content-Type", "application/json")
	return c.Status(rec.StatusCode).SendString(rec.ResponseBody)
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
