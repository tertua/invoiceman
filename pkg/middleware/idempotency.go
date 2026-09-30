package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/configs"
	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/pkg/utils"
	"github.com/tertua/tupay/platform/database"
	"gorm.io/gorm"
)

// Idempotency-Key replay protection: the client sends a unique key per
// intent; the first execution stores its response snapshot, repeats replay
// it, and a reused key with a different payload is rejected with 422.
// Fail-open on infrastructure errors and only active when the header is
// present, so existing clients are unaffected.
const (
	idempotencyHeader   = "Idempotency-Key"
	idempotencyMaxKey   = 255
	idempotencyMaxBody  = 64 << 10 // snapshots larger than this are not stored
	idempotencyPollWait = 5 * time.Second
	idempotencyPollStep = 100 * time.Millisecond
)

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
			deleteIdempotencyKey(db, rec.ID)
			return err
		}
		status := c.Response().StatusCode()
		body := append([]byte(nil), c.Response().Body()...)
		if status >= 500 || len(body) > idempotencyMaxBody {
			// Server errors stay retryable; oversized bodies are not cached.
			deleteIdempotencyKey(db, rec.ID)
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
		deleteIdempotencyKey(db, rec.ID)
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
			select {
			case <-c.Context().Done():
				return utils.Fail(c, fiber.StatusConflict,
					"conflicting request with the same idempotency key is in progress", nil)
			case <-time.After(idempotencyPollStep):
			}
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
