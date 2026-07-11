// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "testing"

func TestSanitizeUserMetadataDropsReserved(t *testing.T) {
	in := Metadata{"team": "core", MetaManagedBy: "spoofed", "miabi.io/x": "y"}
	out := SanitizeUserMetadata(in)
	if out["team"] != "core" {
		t.Errorf("user key dropped: %v", out)
	}
	if _, ok := out[MetaManagedBy]; ok {
		t.Error("reserved managed-by must be stripped from user input")
	}
	if _, ok := out["miabi.io/x"]; ok {
		t.Error("reserved-prefix key must be stripped")
	}
}

func TestMergeUserMetadataProtectsBuiltins(t *testing.T) {
	current := Metadata{
		MetaManagedBy: ManagedByMarketplace, MetaTemplate: "ghost", "team": "core",
	}
	// User tries to: change a built-in, drop their old label, add a new one.
	overlay := Metadata{MetaManagedBy: "hacked", "env": "prod"}
	out := MergeUserMetadata(current, overlay)

	if out[MetaManagedBy] != ManagedByMarketplace {
		t.Errorf("built-in managed-by must be protected, got %q", out[MetaManagedBy])
	}
	if out[MetaTemplate] != "ghost" {
		t.Error("built-in template key must be preserved")
	}
	if out["env"] != "prod" {
		t.Error("new user label should be applied")
	}
	if _, ok := out["team"]; ok {
		t.Error("user metadata is replaced wholesale; old label should be gone")
	}
}

func TestDefaultManagedBy(t *testing.T) {
	if got := DefaultManagedBy(nil, ManagedByUser); got[MetaManagedBy] != ManagedByUser {
		t.Errorf("nil → default user, got %v", got)
	}
	pre := Metadata{MetaManagedBy: ManagedByMarketplace}
	if got := DefaultManagedBy(pre, ManagedByUser); got[MetaManagedBy] != ManagedByMarketplace {
		t.Error("existing managed-by must not be overwritten by the default")
	}
}

func TestSetAndReadOwner(t *testing.T) {
	m := SetOwner(nil, OwnerDatabase, 7, "shop-db")
	ref, ok := Owner(m)
	if !ok || ref.Kind != OwnerDatabase || ref.ID != 7 || ref.Name != "shop-db" {
		t.Fatalf("round-trip owner mismatch: %+v ok=%v", ref, ok)
	}
	// A zero id and empty name must not leave stale keys behind.
	m = SetOwner(m, OwnerUser, 0, "")
	if _, has := m[MetaOwnerID]; has {
		t.Error("owner-id should be cleared when id is 0")
	}
	if _, has := m[MetaOwnerName]; has {
		t.Error("owner-name should be cleared when name is empty")
	}
	ref, _ = Owner(m)
	if ref.Kind != OwnerUser || ref.ID != 0 {
		t.Errorf("expected bare user owner, got %+v", ref)
	}
}

func TestOwnerAbsent(t *testing.T) {
	if _, ok := Owner(nil); ok {
		t.Error("nil metadata has no owner")
	}
	if _, ok := Owner(Metadata{"team": "core"}); ok {
		t.Error("metadata without owner-kind has no owner")
	}
}

func TestDefaultOwnerYieldsToExisting(t *testing.T) {
	// A richer owner (set first) must survive a creation-path default.
	m := SetOwner(nil, OwnerStack, 3, "web")
	m = DefaultOwner(m, OwnerUser, 9, "alice")
	if ref, _ := Owner(m); ref.Kind != OwnerStack || ref.ID != 3 {
		t.Errorf("existing owner must win, got %+v", ref)
	}
	// With no owner present, the default applies.
	got := DefaultOwner(Metadata{MetaManagedBy: ManagedByUser}, OwnerUser, 9, "alice")
	if ref, _ := Owner(got); ref.Kind != OwnerUser || ref.ID != 9 || ref.Name != "alice" {
		t.Errorf("default owner not applied, got %+v", ref)
	}
}
