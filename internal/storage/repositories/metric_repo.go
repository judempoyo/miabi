// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type MetricRepository struct {
	db *gorm.DB
}

func NewMetricRepository(db *gorm.DB) *MetricRepository { return &MetricRepository{db: db} }

func (r *MetricRepository) Insert(m *models.MetricSample) error { return r.db.Create(m).Error }

// ListByApp returns samples for an app recorded since `since`, oldest first.
func (r *MetricRepository) ListByApp(appID uint, since time.Time, limit int) ([]models.MetricSample, error) {
	var samples []models.MetricSample
	q := r.db.Where("application_id = ? AND recorded_at >= ?", appID, since).Order("recorded_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&samples).Error
	return samples, err
}

// ListByApps returns samples for any of the given apps recorded since `since`,
// oldest first. Used to aggregate a workspace's stored per-app history into a
// single workspace-level series. Returns nil for an empty id set.
func (r *MetricRepository) ListByApps(appIDs []uint, since time.Time) ([]models.MetricSample, error) {
	if len(appIDs) == 0 {
		return nil, nil
	}
	var samples []models.MetricSample
	err := r.db.Where("application_id IN ? AND recorded_at >= ?", appIDs, since).
		Order("recorded_at ASC").Find(&samples).Error
	return samples, err
}

// Prune deletes samples older than `before`.
func (r *MetricRepository) Prune(before time.Time) (int64, error) {
	res := r.db.Where("recorded_at < ?", before).Delete(&models.MetricSample{})
	return res.RowsAffected, res.Error
}
