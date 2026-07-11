// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// ACMEAccountRepository persists the platform ACME account(s), keyed by CA URL.
type ACMEAccountRepository struct {
	db *gorm.DB
}

func NewACMEAccountRepository(db *gorm.DB) *ACMEAccountRepository {
	return &ACMEAccountRepository{db: db}
}

func (r *ACMEAccountRepository) Create(a *models.ACMEAccount) error { return r.db.Create(a).Error }

// FindByCA returns the account for a CA directory URL, or ErrRecordNotFound.
func (r *ACMEAccountRepository) FindByCA(caDirURL string) (*models.ACMEAccount, error) {
	var a models.ACMEAccount
	if err := r.db.Where("ca_dir_url = ?", caDirURL).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}
