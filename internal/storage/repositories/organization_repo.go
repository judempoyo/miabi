// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// OrganizationRepository accesses identity realms (SSO/enforced-login/SCIM scope).
type OrganizationRepository struct{ db *gorm.DB }

func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// FindDefault returns the single default organization.
func (r *OrganizationRepository) FindDefault() (*models.Organization, error) {
	var org models.Organization
	if err := r.db.Where("is_default = ?", true).First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepository) FindByID(id uint) (*models.Organization, error) {
	var org models.Organization
	if err := r.db.First(&org, id).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepository) Update(org *models.Organization) error {
	return r.db.Save(org).Error
}
