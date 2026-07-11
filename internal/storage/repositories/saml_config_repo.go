// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// SAMLConfigRepository accesses SAML 2.0 IdP connections (Enterprise; empty in CE).
type SAMLConfigRepository struct{ db *gorm.DB }

func NewSAMLConfigRepository(db *gorm.DB) *SAMLConfigRepository {
	return &SAMLConfigRepository{db: db}
}

func (r *SAMLConfigRepository) Create(c *models.SAMLConfig) error { return r.db.Create(c).Error }
func (r *SAMLConfigRepository) Update(c *models.SAMLConfig) error { return r.db.Save(c).Error }
func (r *SAMLConfigRepository) Delete(id uint) error {
	return r.db.Delete(&models.SAMLConfig{}, id).Error
}

func (r *SAMLConfigRepository) FindAll() ([]models.SAMLConfig, error) {
	var cfgs []models.SAMLConfig
	err := r.db.Order("created_at DESC").Find(&cfgs).Error
	return cfgs, err
}

func (r *SAMLConfigRepository) FindByID(id uint) (*models.SAMLConfig, error) {
	var cfg models.SAMLConfig
	if err := r.db.First(&cfg, id).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *SAMLConfigRepository) ExistsByName(name string) (bool, error) {
	var n int64
	err := r.db.Model(&models.SAMLConfig{}).Where("name = ?", name).Count(&n).Error
	return n > 0, err
}
