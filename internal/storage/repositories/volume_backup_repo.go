// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type VolumeBackupRepository struct {
	db *gorm.DB
}

func NewVolumeBackupRepository(db *gorm.DB) *VolumeBackupRepository {
	return &VolumeBackupRepository{db: db}
}

func (r *VolumeBackupRepository) Create(b *models.VolumeBackup) error { return r.db.Create(b).Error }
func (r *VolumeBackupRepository) Update(b *models.VolumeBackup) error { return r.db.Save(b).Error }

// SetLogMeta records the log-store reference + counters for a volume-backup run
// and replaces the DB column with the bounded tail (the full log lives in the
// store). A zero ref is ignored so a store failure leaves the full DB tail intact.
func (r *VolumeBackupRepository) SetLogMeta(id uint, ref, tail string, bytes int64, lines int, truncated bool) error {
	if ref == "" {
		return nil
	}
	return r.db.Model(&models.VolumeBackup{}).Where("id = ?", id).
		Updates(map[string]any{
			"logs":          tail,
			"log_ref":       ref,
			"log_bytes":     bytes,
			"log_lines":     lines,
			"log_truncated": truncated,
		}).Error
}
func (r *VolumeBackupRepository) Delete(id uint) error {
	return r.db.Delete(&models.VolumeBackup{}, id).Error
}

func (r *VolumeBackupRepository) FindByID(id uint) (*models.VolumeBackup, error) {
	var b models.VolumeBackup
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *VolumeBackupRepository) FindInWorkspace(workspaceID, id uint) (*models.VolumeBackup, error) {
	var b models.VolumeBackup
	if err := r.db.Where("id = ? AND workspace_id = ?", id, workspaceID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *VolumeBackupRepository) ListByVolume(volumeID uint) ([]models.VolumeBackup, error) {
	var backups []models.VolumeBackup
	err := r.db.Where("volume_id = ?", volumeID).Order("created_at DESC").Find(&backups).Error
	return backups, err
}
