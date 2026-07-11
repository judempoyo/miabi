// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/models"
)

// PermissionHandler serves the static permission catalog and built-in role
// presets — the source of truth for the role-picker UI.
type PermissionHandler struct{}

func NewPermissionHandler() *PermissionHandler { return &PermissionHandler{} }

// PermissionInfo describes one capability for the UI.
type PermissionInfo struct {
	ID       string `json:"id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// RolePreset is a built-in role and the permissions it grants.
type RolePreset struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// PermissionCatalog is the full catalog plus the built-in role presets.
type PermissionCatalog struct {
	Permissions []PermissionInfo `json:"permissions"`
	Roles       []RolePreset     `json:"roles"`
}

// Catalog returns the permission catalog and built-in role presets.
func (h *PermissionHandler) Catalog(c *okapi.Context) error {
	all := models.AllPermissions()
	perms := make([]PermissionInfo, 0, len(all))
	for _, p := range all {
		perms = append(perms, PermissionInfo{ID: string(p), Resource: p.Resource(), Action: p.Action()})
	}
	roles := make([]RolePreset, 0, 4)
	for _, role := range []models.WorkspaceRole{
		models.WorkspaceRoleViewer, models.WorkspaceRoleDeveloper,
		models.WorkspaceRoleAdmin, models.WorkspaceRoleOwner,
	} {
		granted := models.RolePermissions(role)
		// Emit in catalog order for stable output.
		list := make([]string, 0, len(granted))
		for _, p := range all {
			if granted[p] {
				list = append(list, string(p))
			}
		}
		roles = append(roles, RolePreset{Role: string(role), Permissions: list})
	}
	return ok(c, PermissionCatalog{Permissions: perms, Roles: roles})
}
