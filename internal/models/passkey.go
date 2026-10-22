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
package models

import "time"

// Passkey is a WebAuthn credential a user signs in with instead of a password.
type Passkey struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"index;not null" json:"-"`
	Name   string `gorm:"size:64;not null" json:"name"`
	// CredentialID is the authenticator's credential id, base64url without padding.
	CredentialID string `gorm:"uniqueIndex;size:1400;not null" json:"-"`
	// UserHandle is the opaque WebAuthn user id (base64url), shared by all of a user's passkeys.
	UserHandle string `gorm:"index;size:128;not null" json:"-"`
	// Credential is the serialized webauthn.Credential: public key, sign count and flags.
	Credential string `gorm:"type:text;not null" json:"-"`
	// Synced reports a backup-eligible passkey, which a password manager or platform copies across devices.
	Synced     bool       `gorm:"not null;default:false" json:"synced"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
