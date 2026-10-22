// Copyright (c) 2026 Jonas Kaninda.
//
// This file is part of Miabi.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package repositories

import (
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// PasskeyRepository stores users' WebAuthn credentials.
type PasskeyRepository struct{ db *gorm.DB }

func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository { return &PasskeyRepository{db: db} }

func (r *PasskeyRepository) Create(p *models.Passkey) error { return r.db.Create(p).Error }

// ListByUser returns a user's passkeys, oldest first.
func (r *PasskeyRepository) ListByUser(userID uint) ([]models.Passkey, error) {
	var out []models.Passkey
	err := r.db.Where("user_id = ?", userID).Order("id ASC").Find(&out).Error
	return out, err
}

func (r *PasskeyRepository) FindByCredentialID(credentialID string) (*models.Passkey, error) {
	var p models.Passkey
	if err := r.db.Where("credential_id = ?", credentialID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// RecordUse stores the credential's updated sign count and flags after a sign-in.
func (r *PasskeyRepository) RecordUse(id uint, credential string, at time.Time) error {
	return r.db.Model(&models.Passkey{}).Where("id = ?", id).
		Updates(map[string]any{"credential": credential, "last_used_at": at}).Error
}

// Rename changes a passkey's label; false when the user has no such passkey.
func (r *PasskeyRepository) Rename(id, userID uint, name string) (bool, error) {
	res := r.db.Model(&models.Passkey{}).Where("id = ? AND user_id = ?", id, userID).Update("name", name)
	return res.RowsAffected == 1, res.Error
}

// Delete removes a passkey; false when the user has no such passkey.
func (r *PasskeyRepository) Delete(id, userID uint) (bool, error) {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Passkey{})
	return res.RowsAffected == 1, res.Error
}

// DeleteAllForUser removes every passkey of a user, returning how many there were.
func (r *PasskeyRepository) DeleteAllForUser(userID uint) (int64, error) {
	res := r.db.Where("user_id = ?", userID).Delete(&models.Passkey{})
	return res.RowsAffected, res.Error
}

// Count returns how many passkeys a user has.
func (r *PasskeyRepository) Count(userID uint) (int64, error) {
	var n int64
	return n, r.db.Model(&models.Passkey{}).Where("user_id = ?", userID).Count(&n).Error
}
