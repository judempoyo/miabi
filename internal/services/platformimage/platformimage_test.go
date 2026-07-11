// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package platformimage

import "testing"

// fakeStore is an in-memory settings.Store for the resolver tests.
type fakeStore map[string]string

func (f fakeStore) String(key, def string) string {
	if v, ok := f[key]; ok {
		return v
	}
	return def
}

func TestBuildCatalogDefaults(t *testing.T) {
	r := New(fakeStore{}, nil)

	if got := r.Ref(KeyPack); got != "miabi/pack:latest" {
		t.Errorf("KeyPack default = %q, want miabi/pack:latest", got)
	}
	if got := r.Ref(KeyBuildpackBuilder); got != "paketobuildpacks/builder-jammy-base" {
		t.Errorf("KeyBuildpackBuilder default = %q, want paketobuildpacks/builder-jammy-base", got)
	}
	if !r.ValidKey(KeyPack) || !r.ValidKey(KeyBuildpackBuilder) {
		t.Error("build keys should be valid catalog keys")
	}
}

func TestBuildCatalogOverride(t *testing.T) {
	r := New(fakeStore{SettingKey(KeyPack): "registry.example.com/miabi/pack:v1"}, nil)
	if got := r.Ref(KeyPack); got != "registry.example.com/miabi/pack:v1" {
		t.Errorf("KeyPack override = %q", got)
	}
}
