// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"net/http"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/handlers"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/models"
)

// workspaceBackupSettingsRoutes registers the per-workspace S3 backup target.
// Admin-only: the settings expose the S3 access key.
func (r *Router) workspaceBackupSettingsRoutes() []okapi.RouteDefinition {
	g := r.v1.Group("/workspaces").WithTagInfo(okapi.GroupTag{Name: "Backup Settings", Description: "Per-workspace S3 backup target (shared by database & volume backups)."})
	scoped := func(min models.WorkspaceRole) []okapi.Middleware {
		return []okapi.Middleware{r.authenticate, r.scope, middlewares.RequireRole(min)}
	}
	const base = "/{workspace}/backup-settings"

	return []okapi.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        base,
			Group:       g,
			Middlewares: scoped(models.WorkspaceRoleAdmin),
			Handler:     r.h.backupSettings.Get,
			Summary:     "Get workspace backup settings",
		},
		{
			Method:      http.MethodPut,
			Path:        base,
			Group:       g,
			Middlewares: scoped(models.WorkspaceRoleAdmin),
			Handler:     okapi.H(r.h.backupSettings.Update),
			Summary:     "Update workspace backup settings",
			Request:     &handlers.UpdateBackupSettingsRequest{},
		},
		{
			Method:      http.MethodPost,
			Path:        base + "/test",
			Group:       g,
			Middlewares: scoped(models.WorkspaceRoleAdmin),
			Handler:     okapi.H(r.h.backupSettings.Test),
			Summary:     "Validate workspace backup settings",
			Request:     &handlers.UpdateBackupSettingsRequest{},
		},
	}
}
