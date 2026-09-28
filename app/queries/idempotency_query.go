package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"gorm.io/gorm"
)

// IdempotencyQueries persists executed mutating requests for replay.
type IdempotencyQueries struct {
	*gorm.DB
}

// GetIdempotencyKey returns the stored record for a scope+key hash,
// or gorm.ErrRecordNotFound when absent.
func (q *IdempotencyQueries) GetIdempotencyKey(keyHash, scope string) (models.IdempotencyKey, error) {
	rec := models.IdempotencyKey{}
	if err := q.Where("key_hash = ? AND scope = ?", keyHash, scope).First(&rec).Error; err != nil {
		return rec, err
	}
	return rec, nil
}

// CreateIdempotencyPlaceholder inserts a processing row to win races between
// concurrent retries with the same key. The unique (scope, key) index makes
// exactly one writer win; losers replay the winner.
func (q *IdempotencyQueries) CreateIdempotencyPlaceholder(rec *models.IdempotencyKey) error {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	return q.Create(rec).Error
}

// CompleteIdempotencyKey stores the executed response snapshot.
func (q *IdempotencyQueries) CompleteIdempotencyKey(rec *models.IdempotencyKey) error {
	return q.Save(rec).Error
}

// DeleteIdempotencyKey removes one record (expired miss, oversized body).
func (q *IdempotencyQueries) DeleteIdempotencyKey(id uuid.UUID) error {
	return q.Where("id = ?", id).Delete(&models.IdempotencyKey{}).Error
}

// DeleteExpiredIdempotencyKeys purges rows past their TTL. Called by the
// outbox worker on each tick; returns the purged count.
func (q *IdempotencyQueries) DeleteExpiredIdempotencyKeys(now time.Time) (int64, error) {
	res := q.Where("expires_at <= ?", now).Delete(&models.IdempotencyKey{})
	return res.RowsAffected, res.Error
}
