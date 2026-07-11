// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"time"
)

// Registry is a stored container-registry credential used to pull private
// images at deploy time. The Secret (password / access token) is encrypted at
// rest and never serialized.
type Registry struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	WorkspaceID uint `json:"workspace_id" gorm:"index:idx_registry_workspace_name,unique;not null"`
	// Name is the unique slug handle scoped to the workspace.
	Name string `json:"name" gorm:"index:idx_registry_workspace_name,unique;not null"`
	// DisplayName is the free-text label shown in the UI; not unique.
	DisplayName string `json:"display_name"`
	// Server is the registry host, e.g. "registry-1.docker.io", "ghcr.io",
	// "registry.gitlab.com". Empty defaults to Docker Hub.
	Server   string `json:"server"`
	Username string `json:"username"`
	// Secret holds the password or access token, encrypted at rest.
	Secret string `json:"-" gorm:"not null"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// HasSecret is a transient flag for responses (never persisted).
	HasSecret bool `json:"has_secret" gorm:"-"`
}
