// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package customrole

import (
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

// validate is pure (no repo access), so we exercise the no-escalation invariant
// directly. An admin actor is the canonical author.
func TestValidate(t *testing.T) {
	admin := models.RolePermissions(models.WorkspaceRoleAdmin)
	developer := models.RolePermissions(models.WorkspaceRoleDeveloper)
	s := &Service{}

	cases := []struct {
		name    string
		in      Input
		actor   map[models.Permission]bool
		wantErr error
		wantOK  bool
	}{
		{
			name:   "valid subset",
			in:     Input{Name: "Deployer", BaseRole: models.WorkspaceRoleDeveloper, Permissions: []models.Permission{models.PermAppRead, models.PermAppDeploy}},
			actor:  admin,
			wantOK: true,
		},
		{
			name:    "escalation: permission actor lacks",
			in:      Input{Name: "Super", BaseRole: models.WorkspaceRoleDeveloper, Permissions: []models.Permission{models.PermWorkspaceDelete}},
			actor:   admin, // admin lacks workspace:delete
			wantErr: ErrEscalation,
		},
		{
			name:    "escalation via base role rank",
			in:      Input{Name: "Owners", BaseRole: models.WorkspaceRoleOwner, Permissions: []models.Permission{models.PermAppRead}},
			actor:   admin, // owner preset ⊄ admin
			wantErr: ErrEscalation,
		},
		{
			name:    "developer actor cannot grant delete",
			in:      Input{Name: "X", BaseRole: models.WorkspaceRoleDeveloper, Permissions: []models.Permission{models.PermAppDelete}},
			actor:   developer,
			wantErr: ErrEscalation,
		},
		{
			name:    "unknown permission",
			in:      Input{Name: "X", BaseRole: models.WorkspaceRoleDeveloper, Permissions: []models.Permission{"app:teleport"}},
			actor:   admin,
			wantErr: ErrInvalidPermission,
		},
		{
			name:    "empty name",
			in:      Input{Name: "  ", BaseRole: models.WorkspaceRoleViewer, Permissions: []models.Permission{models.PermAppRead}},
			actor:   admin,
			wantErr: ErrNameRequired,
		},
		{
			name:    "no permissions",
			in:      Input{Name: "X", BaseRole: models.WorkspaceRoleViewer, Permissions: nil},
			actor:   admin,
			wantErr: ErrNoPermissions,
		},
		{
			name:    "invalid base role",
			in:      Input{Name: "X", BaseRole: "superuser", Permissions: []models.Permission{models.PermAppRead}},
			actor:   admin,
			wantErr: ErrInvalidBaseRole,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.validate(tc.in, tc.actor)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected ok, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}
