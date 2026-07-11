// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// WorkspaceKeyRepository persists per-workspace wrapped data-encryption keys.
type WorkspaceKeyRepository struct {
	db *gorm.DB
}

func NewWorkspaceKeyRepository(db *gorm.DB) *WorkspaceKeyRepository {
	return &WorkspaceKeyRepository{db: db}
}

func (r *WorkspaceKeyRepository) Create(k *models.WorkspaceKey) error { return r.db.Create(k).Error }
func (r *WorkspaceKeyRepository) Update(k *models.WorkspaceKey) error { return r.db.Save(k).Error }

// FindActive returns the workspace's active key version, or ErrRecordNotFound.
func (r *WorkspaceKeyRepository) FindActive(workspaceID uint) (*models.WorkspaceKey, error) {
	var k models.WorkspaceKey
	if err := r.db.Where("workspace_id = ? AND active = ?", workspaceID, true).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// FindVersion returns a specific key version (for decrypting old ciphertext).
func (r *WorkspaceKeyRepository) FindVersion(workspaceID uint, version int) (*models.WorkspaceKey, error) {
	var k models.WorkspaceKey
	if err := r.db.Where("workspace_id = ? AND version = ?", workspaceID, version).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// ListByWorkspace returns all key versions for a workspace (newest first).
func (r *WorkspaceKeyRepository) ListByWorkspace(workspaceID uint) ([]models.WorkspaceKey, error) {
	var ks []models.WorkspaceKey
	err := r.db.Where("workspace_id = ?", workspaceID).Order("version DESC").Find(&ks).Error
	return ks, err
}

// MaxVersion returns the highest version number for a workspace (0 when none).
func (r *WorkspaceKeyRepository) MaxVersion(workspaceID uint) (int, error) {
	var max *int
	if err := r.db.Model(&models.WorkspaceKey{}).
		Where("workspace_id = ?", workspaceID).
		Select("MAX(version)").Scan(&max).Error; err != nil {
		return 0, err
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

// DeactivateAll clears the active flag for a workspace's keys (used before
// activating a new version during rotation).
func (r *WorkspaceKeyRepository) DeactivateAll(tx *gorm.DB, workspaceID uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Model(&models.WorkspaceKey{}).
		Where("workspace_id = ?", workspaceID).Update("active", false).Error
}

// ListActiveOlderThan returns active keys whose RotatedAt is before the cutoff
// (used by the auto-rotate cron). RotatedAt is set on create and on rotation.
func (r *WorkspaceKeyRepository) ListActiveOlderThan(cutoff time.Time) ([]models.WorkspaceKey, error) {
	var ks []models.WorkspaceKey
	err := r.db.Where("active = ? AND rotated_at < ?", true, cutoff).Find(&ks).Error
	return ks, err
}

// DeleteOldVersions deletes every key version for a workspace except `keep`
// (used to retire old DEKs after a fully-successful rotation sweep).
func (r *WorkspaceKeyRepository) DeleteOldVersions(workspaceID uint, keep int) error {
	return r.db.Where("workspace_id = ? AND version <> ?", workspaceID, keep).
		Delete(&models.WorkspaceKey{}).Error
}

// DeleteByWorkspace removes all of a workspace's keys (crypto-shred on delete).
func (r *WorkspaceKeyRepository) DeleteByWorkspace(workspaceID uint) error {
	return r.db.Where("workspace_id = ?", workspaceID).Delete(&models.WorkspaceKey{}).Error
}

// DB exposes the underlying handle for transactional rotation.
func (r *WorkspaceKeyRepository) DB() *gorm.DB { return r.db }
