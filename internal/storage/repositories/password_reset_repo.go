// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type PasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(t *models.PasswordResetToken) error {
	return r.db.Create(t).Error
}

// FindValidByHash returns an unused, unexpired token by hash.
func (r *PasswordResetRepository) FindValidByHash(hash string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := r.db.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now()).
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepository) MarkUsed(id uint) error {
	return r.db.Model(&models.PasswordResetToken{}).
		Where("id = ?", id).Update("used_at", gorm.Expr("NOW()")).Error
}
