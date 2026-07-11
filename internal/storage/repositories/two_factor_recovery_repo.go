// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type TwoFactorRecoveryRepository struct {
	db *gorm.DB
}

func NewTwoFactorRecoveryRepository(db *gorm.DB) *TwoFactorRecoveryRepository {
	return &TwoFactorRecoveryRepository{db: db}
}

// ReplaceForUser deletes the user's existing recovery codes and inserts the new
// set in a single transaction.
func (r *TwoFactorRecoveryRepository) ReplaceForUser(userID uint, codes []*models.TwoFactorRecoveryCode) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.TwoFactorRecoveryCode{}).Error; err != nil {
			return err
		}
		if len(codes) == 0 {
			return nil
		}
		return tx.Create(&codes).Error
	})
}

// FindUnusedByHash returns an unused recovery code matching the hash for a user.
func (r *TwoFactorRecoveryRepository) FindUnusedByHash(userID uint, hash string) (*models.TwoFactorRecoveryCode, error) {
	var code models.TwoFactorRecoveryCode
	err := r.db.Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hash).First(&code).Error
	if err != nil {
		return nil, err
	}
	return &code, nil
}

// MarkUsed flags a recovery code as consumed.
func (r *TwoFactorRecoveryRepository) MarkUsed(id uint) error {
	return r.db.Model(&models.TwoFactorRecoveryCode{}).
		Where("id = ?", id).Update("used_at", gorm.Expr("NOW()")).Error
}

// CountUnused returns how many unused recovery codes the user has left.
func (r *TwoFactorRecoveryRepository) CountUnused(userID uint) (int64, error) {
	var n int64
	err := r.db.Model(&models.TwoFactorRecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).Count(&n).Error
	return n, err
}

// DeleteForUser removes all of a user's recovery codes.
func (r *TwoFactorRecoveryRepository) DeleteForUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.TwoFactorRecoveryCode{}).Error
}
