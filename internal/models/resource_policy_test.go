// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "testing"

func TestResourcePolicyGrants(t *testing.T) {
	p := &ResourcePolicy{Permissions: []string{string(PermAppRead), string(PermAppDeploy)}}
	if !p.Grants(PermAppDeploy) {
		t.Fatal("expected app:deploy granted")
	}
	if p.Grants(PermAppDelete) {
		t.Fatal("app:delete must not be granted")
	}
	empty := &ResourcePolicy{}
	if empty.Grants(PermAppRead) {
		t.Fatal("empty policy grants nothing")
	}
}

func TestIsValidResourceType(t *testing.T) {
	for _, ok := range []string{ResourceTypeApp, ResourceTypeDatabase, ResourceTypeDomain} {
		if !IsValidResourceType(ok) {
			t.Fatalf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "workspace", "node"} {
		if IsValidResourceType(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}
