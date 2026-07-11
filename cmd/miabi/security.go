// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"

	"github.com/miabi-io/miabi/internal/config"
	"github.com/miabi-io/miabi/internal/services/quota"
	"github.com/miabi-io/miabi/internal/worker"
)

// newSecurityResolver builds the container security resolver wired into the
// deploy and job handlers. An application or job container is hardened — run as
// RestrictedUID:0 with no-new-privileges and NET_RAW dropped (the "restricted"
// security profile, OpenShift-style) — when either the global ForceNonRootUser
// default is set, or the workspace's effective plan selects the restricted
// profile. Returns nil (no restriction) when no platform UID is configured.
//
// An app installed from an official marketplace template is exempted — it keeps
// the image's own default user — when its plan grants AllowOfficialImageUser and
// the restriction comes from the plan profile (not the platform-wide
// ForceNonRootUser mandate, which is absolute). This lets curated official images
// that require their baked-in user run under an otherwise restricted workspace.
func newSecurityResolver(cfg *config.Config, q *quota.Service) worker.SecurityResolver {
	if cfg.RestrictedUID <= 0 {
		return nil
	}
	user := fmt.Sprintf("%d:0", cfg.RestrictedUID) // GID 0: arbitrary-UID convention
	return worker.SecurityFunc(func(workspaceID uint, officialTemplate bool) worker.Security {
		if !cfg.ForceNonRootUser && !q.RestrictedProfile(workspaceID) {
			return worker.Security{} // profile is "default": image user, no hardening
		}
		// Restricted applies. Official-template apps may keep the image user when the
		// plan allows it — but never against a platform-wide non-root mandate.
		if !cfg.ForceNonRootUser && officialTemplate && q.AllowOfficialImageUser(workspaceID) {
			return worker.Security{}
		}
		return worker.Security{User: user, NoNewPrivileges: true, CapDrop: []string{"NET_RAW"}}
	})
}
