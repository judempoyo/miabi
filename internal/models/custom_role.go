// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// CustomRole is an admin-defined permission set assigned to workspace members
// (Enterprise; gated on the custom_roles entitlement for writes). BaseRole gives a
// rank so the legacy rank-based RequireRole still resolves sanely for a member on
// a custom role; the Permissions list drives the fine-grained RequirePermission
// checks. The table is empty in Community.
type CustomRole struct {
	ID          uint          `json:"id" gorm:"primaryKey"`
	WorkspaceID *uint         `json:"workspace_id" gorm:"index"`
	Name        string        `json:"name" gorm:"not null"`
	BaseRole    WorkspaceRole `json:"base_role" gorm:"not null"`
	Permissions []string      `json:"permissions" gorm:"serializer:json"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// PermissionSet returns the role's permissions as a lookup set.
func (r *CustomRole) PermissionSet() map[Permission]bool {
	out := make(map[Permission]bool, len(r.Permissions))
	for _, p := range r.Permissions {
		out[Permission(p)] = true
	}
	return out
}
