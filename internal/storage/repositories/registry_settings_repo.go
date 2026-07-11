// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// RegistrySettingsRepository persists the single-row internal-registry config.
type RegistrySettingsRepository struct {
	db *gorm.DB
}

func NewRegistrySettingsRepository(db *gorm.DB) *RegistrySettingsRepository {
	return &RegistrySettingsRepository{db: db}
}

// Get returns the single settings row, or gorm.ErrRecordNotFound when unset.
func (r *RegistrySettingsRepository) Get() (*models.RegistrySettings, error) {
	var st models.RegistrySettings
	if err := r.db.Order("id ASC").First(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// Upsert saves the single settings row (creating it on first save).
func (r *RegistrySettingsRepository) Upsert(st *models.RegistrySettings) error {
	return r.db.Save(st).Error
}
