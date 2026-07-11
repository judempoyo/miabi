// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// TwoFactorRecoveryCode is a single-use backup code for two-factor
// authentication. Only the sha256 hash of the code is stored; the plaintext is
// shown to the user once at generation time.
type TwoFactorRecoveryCode struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"index;not null"`
	CodeHash  string     `json:"-" gorm:"uniqueIndex;not null"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}
