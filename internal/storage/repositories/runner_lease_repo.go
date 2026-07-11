// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// RunnerLeaseRepository persists runner claims on pipeline runs (the at-most-once
// lease + concurrency accounting + dead-lease requeue).
type RunnerLeaseRepository struct {
	db *gorm.DB
}

func NewRunnerLeaseRepository(db *gorm.DB) *RunnerLeaseRepository {
	return &RunnerLeaseRepository{db: db}
}

func (r *RunnerLeaseRepository) Create(l *models.RunnerLease) error { return r.db.Create(l).Error }
func (r *RunnerLeaseRepository) Update(l *models.RunnerLease) error { return r.db.Save(l).Error }

// ActiveCountByRunner returns how many active leases a runner currently holds —
// its live load, compared against the runner's declared Concurrency by the
// scheduler.
func (r *RunnerLeaseRepository) ActiveCountByRunner(runnerID uint) (int, error) {
	var n int64
	err := r.db.Model(&models.RunnerLease{}).
		Where("runner_id = ? AND status = ?", runnerID, models.LeaseActive).Count(&n).Error
	return int(n), err
}

// ActiveCountsByRunner returns the active-lease count for every runner that has
// at least one, keyed by runner id — one query for the scheduler to rank all
// candidates without an N+1.
func (r *RunnerLeaseRepository) ActiveCountsByRunner() (map[uint]int, error) {
	var rows []struct {
		RunnerID uint
		N        int
	}
	err := r.db.Model(&models.RunnerLease{}).
		Select("runner_id, COUNT(*) AS n").
		Where("status = ?", models.LeaseActive).
		Group("runner_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(rows))
	for _, row := range rows {
		out[row.RunnerID] = row.N
	}
	return out, nil
}

// ActiveByRun returns the current active lease on a job (pipeline run or build,
// per kind), if any — used to enforce the at-most-once claim and to find the
// runner executing it.
func (r *RunnerLeaseRepository) ActiveByRun(kind models.LeaseKind, runID uint) (*models.RunnerLease, error) {
	var l models.RunnerLease
	if err := r.db.Where("kind = ? AND run_id = ? AND status = ?", kind, runID, models.LeaseActive).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// Release marks a job's active lease done (called when the job goes terminal).
func (r *RunnerLeaseRepository) Release(kind models.LeaseKind, runID uint) error {
	return r.db.Model(&models.RunnerLease{}).
		Where("kind = ? AND run_id = ? AND status = ?", kind, runID, models.LeaseActive).
		Update("status", models.LeaseDone).Error
}

// ExpireDue marks every active lease past now as expired and returns them, so
// the caller can requeue their runs onto another runner. Runs in one UPDATE to
// avoid a lost-update race between the sweeper and a late-completing runner.
func (r *RunnerLeaseRepository) ExpireDue(now time.Time) ([]models.RunnerLease, error) {
	var due []models.RunnerLease
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("status = ? AND expires_at < ?", models.LeaseActive, now).Find(&due).Error; e != nil {
			return e
		}
		if len(due) == 0 {
			return nil
		}
		ids := make([]uint, len(due))
		for i := range due {
			ids[i] = due[i].ID
		}
		return tx.Model(&models.RunnerLease{}).Where("id IN ?", ids).Update("status", models.LeaseExpired).Error
	})
	return due, err
}
