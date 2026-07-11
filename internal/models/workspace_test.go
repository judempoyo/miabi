// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "testing"

func TestWorkspaceRoleAtLeast(t *testing.T) {
	cases := []struct {
		role WorkspaceRole
		min  WorkspaceRole
		want bool
	}{
		{WorkspaceRoleOwner, WorkspaceRoleAdmin, true},
		{WorkspaceRoleAdmin, WorkspaceRoleAdmin, true},
		{WorkspaceRoleDeveloper, WorkspaceRoleAdmin, false},
		{WorkspaceRoleViewer, WorkspaceRoleDeveloper, false},
		{WorkspaceRoleDeveloper, WorkspaceRoleViewer, true},
		{WorkspaceRole("bogus"), WorkspaceRoleViewer, false},
	}
	for _, c := range cases {
		if got := c.role.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}

func TestWorkspaceRoleValid(t *testing.T) {
	if !WorkspaceRoleOwner.Valid() {
		t.Error("owner should be valid")
	}
	if WorkspaceRole("manager").Valid() {
		t.Error("manager should be invalid")
	}
}
