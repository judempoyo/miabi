// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"strings"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type OAuthProviderRepository struct {
	db *gorm.DB
}

func NewOAuthProviderRepository(db *gorm.DB) *OAuthProviderRepository {
	return &OAuthProviderRepository{db: db}
}

func (r *OAuthProviderRepository) Create(p *models.OAuthProvider) error {
	return r.db.Create(p).Error
}

func (r *OAuthProviderRepository) Update(p *models.OAuthProvider) error {
	return r.db.Save(p).Error
}

func (r *OAuthProviderRepository) Delete(id uint) error {
	return r.db.Delete(&models.OAuthProvider{}, id).Error
}

// Count returns the number of configured providers. Used by the single-provider
// quota guard (Community allows one; the multi_sso entitlement lifts the cap).
func (r *OAuthProviderRepository) Count() (int64, error) {
	var n int64
	if err := r.db.Model(&models.OAuthProvider{}).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// FindAll returns every provider (admin view), newest first.
func (r *OAuthProviderRepository) FindAll() ([]models.OAuthProvider, error) {
	var providers []models.OAuthProvider
	if err := r.db.Order("created_at DESC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// FindEnabled returns enabled, non-hidden providers for the login screen.
func (r *OAuthProviderRepository) FindEnabled() ([]models.OAuthProvider, error) {
	var providers []models.OAuthProvider
	if err := r.db.Where("enabled = ? AND hidden = ?", true, false).
		Order("name ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// HasEnabledHidden reports whether any enabled provider is hidden from the login
// buttons — i.e. reachable only via "Continue with SSO" email discovery. Drives
// whether that button is offered on the login screen.
func (r *OAuthProviderRepository) HasEnabledHidden() (bool, error) {
	var count int64
	if err := r.db.Model(&models.OAuthProvider{}).
		Where("enabled = ? AND hidden = ?", true, true).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *OAuthProviderRepository) FindEnabledByDomain(domain string) (*models.OAuthProvider, error) {
	target := strings.ToLower(strings.TrimSpace(domain))
	if target == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var providers []models.OAuthProvider
	if err := r.db.Where("enabled = ? AND allowed_domains <> ''", true).
		Order("name ASC").Find(&providers).Error; err != nil {
		return nil, err
	}
	for i := range providers {
		for _, d := range strings.Split(providers[i].AllowedDomains, ",") {
			if strings.ToLower(strings.TrimSpace(d)) == target {
				return &providers[i], nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *OAuthProviderRepository) FindByID(id uint) (*models.OAuthProvider, error) {
	var p models.OAuthProvider
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *OAuthProviderRepository) FindByName(name string) (*models.OAuthProvider, error) {
	var p models.OAuthProvider
	if err := r.db.Where("name = ?", strings.ToLower(strings.TrimSpace(name))).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *OAuthProviderRepository) ExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.OAuthProvider{}).
		Where("name = ?", strings.ToLower(strings.TrimSpace(name))).Count(&count).Error
	return count > 0, err
}
