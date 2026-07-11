// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// WorkspaceQuotaRepository persists per-workspace quota overrides.
type WorkspaceQuotaRepository struct {
	db *gorm.DB
}

func NewWorkspaceQuotaRepository(db *gorm.DB) *WorkspaceQuotaRepository {
	return &WorkspaceQuotaRepository{db: db}
}

// FindByWorkspace returns the override row for a workspace, or
// gorm.ErrRecordNotFound when none is set.
func (r *WorkspaceQuotaRepository) FindByWorkspace(workspaceID uint) (*models.WorkspaceQuota, error) {
	var q models.WorkspaceQuota
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// Upsert creates or replaces a workspace's override row.
func (r *WorkspaceQuotaRepository) Upsert(q *models.WorkspaceQuota) error {
	return r.db.Save(q).Error
}

// Delete clears a workspace's override (falls back to the plan).
func (r *WorkspaceQuotaRepository) Delete(workspaceID uint) error {
	return r.db.Where("workspace_id = ?", workspaceID).Delete(&models.WorkspaceQuota{}).Error
}
