// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// Session tracks an active JWT session for a user (revocable via its JTI).
type Session struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	JTI       string    `json:"jti" gorm:"uniqueIndex;not null;size:36"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent" gorm:"size:512"`
	Revoked   bool      `json:"revoked" gorm:"default:false;not null"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsActive reports whether the session is neither revoked nor expired.
func (s *Session) IsActive() bool { return !s.Revoked && time.Now().Before(s.ExpiresAt) }
