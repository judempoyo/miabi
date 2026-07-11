// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkspaceBackupSettingsRepository struct {
	db *gorm.DB
}

func NewWorkspaceBackupSettingsRepository(db *gorm.DB) *WorkspaceBackupSettingsRepository {
	return &WorkspaceBackupSettingsRepository{db: db}
}

// FindByWorkspace returns the settings for a workspace, or gorm.ErrRecordNotFound
// when none have been saved yet.
func (r *WorkspaceBackupSettingsRepository) FindByWorkspace(workspaceID uint) (*models.WorkspaceBackupSettings, error) {
	var s models.WorkspaceBackupSettings
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// Upsert inserts or updates the workspace's settings (keyed by workspace_id).
func (r *WorkspaceBackupSettingsRepository) Upsert(s *models.WorkspaceBackupSettings) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "workspace_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"s3_enabled", "s3_endpoint", "s3_bucket", "s3_region", "s3_access_key",
			"s3_secret_key_enc", "s3_use_ssl", "s3_force_path_style",
			"database_backup_path", "volume_backup_path", "updated_at",
		}),
	}).Create(s).Error
}
