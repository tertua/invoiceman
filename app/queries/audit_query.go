package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/tertua/invoiceman/app/models"
	"gorm.io/gorm"
)

// AuditQueries persists and lists the audit trail.
type AuditQueries struct {
	*gorm.DB
}

// RecordAudit stores one audit entry. Failures are returned so callers can
// log them; an audit write must never fail the user action itself.
func (q *AuditQueries) RecordAudit(entry *models.AuditLog) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	return q.Create(entry).Error
}

// ListAuditLogs returns one page of the trail, newest first.
func (q *AuditQueries) ListAuditLogs(limit, offset int) ([]models.AuditLog, error) {
	out := []models.AuditLog{}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CountAuditLogs returns the total trail size for pagination meta.
func (q *AuditQueries) CountAuditLogs() (int64, error) {
	var total int64
	if err := q.Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
