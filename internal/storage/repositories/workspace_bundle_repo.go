// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// WorkspaceBundleRepository stores portable workspace bundle runs.
type WorkspaceBundleRepository struct {
	db *gorm.DB
}

func NewWorkspaceBundleRepository(db *gorm.DB) *WorkspaceBundleRepository {
	return &WorkspaceBundleRepository{db: db}
}

func (r *WorkspaceBundleRepository) Create(b *models.WorkspaceBundle) error {
	return r.db.Create(b).Error
}

func (r *WorkspaceBundleRepository) Update(b *models.WorkspaceBundle) error {
	return r.db.Save(b).Error
}

func (r *WorkspaceBundleRepository) Delete(id uint) error {
	return r.db.Delete(&models.WorkspaceBundle{}, id).Error
}

func (r *WorkspaceBundleRepository) FindByID(id uint) (*models.WorkspaceBundle, error) {
	var b models.WorkspaceBundle
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *WorkspaceBundleRepository) FindInWorkspace(workspaceID, id uint) (*models.WorkspaceBundle, error) {
	var b models.WorkspaceBundle
	if err := r.db.Where("id = ? AND workspace_id = ?", id, workspaceID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// HasActive reports whether a run is pending or in flight for the workspace.
// Asked before starting one: two concurrent restores would race on every
// resource they create.
func (r *WorkspaceBundleRepository) HasActive(workspaceID uint) (bool, error) {
	var n int64
	err := r.db.Model(&models.WorkspaceBundle{}).
		Where("workspace_id = ? AND status IN ?", workspaceID,
			[]models.BackupStatus{models.BackupPending, models.BackupRunning}).
		Count(&n).Error
	return n > 0, err
}

// ListByWorkspace returns a workspace's bundle runs, most recent first.
func (r *WorkspaceBundleRepository) ListByWorkspace(workspaceID uint, limit int) ([]models.WorkspaceBundle, error) {
	var out []models.WorkspaceBundle
	q := r.db.Where("workspace_id = ?", workspaceID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}
