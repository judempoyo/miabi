// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// PlatformBackupRepository persists platform (control-plane) backup runs.
type PlatformBackupRepository struct {
	db *gorm.DB
}

func NewPlatformBackupRepository(db *gorm.DB) *PlatformBackupRepository {
	return &PlatformBackupRepository{db: db}
}

func (r *PlatformBackupRepository) Create(b *models.PlatformBackup) error {
	return r.db.Create(b).Error
}
func (r *PlatformBackupRepository) Update(b *models.PlatformBackup) error { return r.db.Save(b).Error }

// SetLogMeta records the log-store reference + counters for a platform-backup
// run and replaces the DB column with the bounded tail (the full log lives in
// the store). A zero ref is ignored so a store failure leaves the full DB tail
// intact.
func (r *PlatformBackupRepository) SetLogMeta(id uint, ref, tail string, bytes int64, lines int, truncated bool) error {
	if ref == "" {
		return nil
	}
	return r.db.Model(&models.PlatformBackup{}).Where("id = ?", id).
		Updates(map[string]any{
			"logs":          tail,
			"log_ref":       ref,
			"log_bytes":     bytes,
			"log_lines":     lines,
			"log_truncated": truncated,
		}).Error
}
func (r *PlatformBackupRepository) Delete(id uint) error {
	return r.db.Delete(&models.PlatformBackup{}, id).Error
}

func (r *PlatformBackupRepository) FindByID(id uint) (*models.PlatformBackup, error) {
	var b models.PlatformBackup
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// List returns all platform backups, newest first.
func (r *PlatformBackupRepository) List() ([]models.PlatformBackup, error) {
	var backups []models.PlatformBackup
	err := r.db.Order("created_at DESC").Find(&backups).Error
	return backups, err
}

// ListPaged returns a page of platform backups (newest first) and the total count.
func (r *PlatformBackupRepository) ListPaged(limit, offset int) ([]models.PlatformBackup, int64, error) {
	var (
		backups []models.PlatformBackup
		total   int64
	)
	if err := r.db.Model(&models.PlatformBackup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&backups).Error
	return backups, total, err
}

// PlatformBackupSettingsRepository persists the single-row platform backup
// target + policy.
type PlatformBackupSettingsRepository struct {
	db *gorm.DB
}

func NewPlatformBackupSettingsRepository(db *gorm.DB) *PlatformBackupSettingsRepository {
	return &PlatformBackupSettingsRepository{db: db}
}

// Get returns the single settings row, or gorm.ErrRecordNotFound when unset.
func (r *PlatformBackupSettingsRepository) Get() (*models.PlatformBackupSettings, error) {
	var st models.PlatformBackupSettings
	if err := r.db.Order("id ASC").First(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// Upsert saves the single settings row (creating it on first save).
func (r *PlatformBackupSettingsRepository) Upsert(st *models.PlatformBackupSettings) error {
	return r.db.Save(st).Error
}
