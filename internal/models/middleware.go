// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"time"
)

// Middleware is a Goma Gateway middleware owned by a workspace. Routes reference
// it by Name. Rule holds the type-specific configuration as defined by Goma
// (mirrors goma-admin's JSON config approach).
type Middleware struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	WorkspaceID uint `json:"workspace_id" gorm:"index:idx_mw_workspace_name,unique;not null"`
	// Name is the unique slug handle scoped to the workspace (Goma middleware name).
	Name string `json:"name" gorm:"index:idx_mw_workspace_name,unique;not null"`
	// DisplayName is the free-text label shown in the UI; not unique.
	DisplayName string                 `json:"display_name"`
	Type        string                 `json:"type" gorm:"not null"` // Goma middleware kind, e.g. basicAuth, rateLimit
	Paths       []string               `json:"paths,omitempty" gorm:"serializer:json"`
	Rule        map[string]interface{} `json:"rule,omitempty" gorm:"serializer:json"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
