// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package marketplace

import (
	"testing"

	"github.com/miabi-io/miabi/internal/services/marketplace/manifest"
)

func TestImageRef(t *testing.T) {
	if got := imageRef(manifest.AppSpec{Image: "ghcr.io/a/b", Tag: "1.2"}); got != "ghcr.io/a/b:1.2" {
		t.Errorf("with tag: got %q", got)
	}
	if got := imageRef(manifest.AppSpec{Image: "redis"}); got != "redis" {
		t.Errorf("no tag: got %q", got)
	}
}

func TestDiffEnv(t *testing.T) {
	oldA := manifest.AppSpec{Env: map[string]string{"KEEP": "1", "CHANGE": "old", "GONE": "x"}}
	newA := manifest.AppSpec{
		Env:       map[string]string{"KEEP": "1", "CHANGE": "new", "ADD": "{{ .inputs.X }}"},
		SecretEnv: []string{"ADD"},
	}
	var warnings []string
	got := diffEnv(oldA, newA, &warnings)

	kinds := map[string]EnvChange{}
	for _, c := range got {
		kinds[c.Key] = c
	}
	if _, ok := kinds["KEEP"]; ok {
		t.Error("unchanged key should not appear in the diff")
	}
	if kinds["CHANGE"].Kind != "changed" {
		t.Errorf("CHANGE: %+v", kinds["CHANGE"])
	}
	if kinds["GONE"].Kind != "removed" {
		t.Errorf("GONE: %+v", kinds["GONE"])
	}
	add := kinds["ADD"]
	if add.Kind != "added" || !add.Secret || !add.Templated {
		t.Errorf("ADD should be added+secret+templated, got %+v", add)
	}
}

func TestDiffStackEnv(t *testing.T) {
	oldM := &manifest.Manifest{Stack: &manifest.StackSpec{
		Env: map[string]string{"KEEP": "1", "CHANGE": "old", "GONE": "x"},
	}}
	newM := &manifest.Manifest{Stack: &manifest.StackSpec{
		Env:       map[string]string{"KEEP": "1", "CHANGE": "new", "ADD": "{{ .databases.db.host }}"},
		SecretEnv: []string{"ADD"},
	}}
	kinds := map[string]EnvChange{}
	for _, c := range diffStackEnv(oldM, newM) {
		kinds[c.Key] = c
	}
	if _, ok := kinds["KEEP"]; ok {
		t.Error("unchanged shared key should not appear in the diff")
	}
	if kinds["CHANGE"].Kind != "changed" {
		t.Errorf("CHANGE: %+v", kinds["CHANGE"])
	}
	if kinds["GONE"].Kind != "removed" {
		t.Errorf("GONE: %+v", kinds["GONE"])
	}
	if add := kinds["ADD"]; add.Kind != "added" || !add.Secret || !add.Templated {
		t.Errorf("ADD should be added+secret+templated, got %+v", add)
	}

	// No stack on either side → no diff (and no panic on nil stacks).
	if got := diffStackEnv(&manifest.Manifest{}, &manifest.Manifest{}); len(got) != 0 {
		t.Errorf("expected no diff for stack-less manifests, got %v", got)
	}
}

func TestAddedAndMounts(t *testing.T) {
	old := map[string]bool{"a": true}
	cur := map[string]bool{"a": true, "b": true}
	got := added(old, cur)
	if len(got) != 1 || got[0] != "b" {
		t.Errorf("added = %v, want [b]", got)
	}

	oldA := manifest.AppSpec{Mounts: []manifest.Mount{{Volume: "data"}}}
	newA := manifest.AppSpec{Mounts: []manifest.Mount{{Volume: "data"}, {Volume: "cache"}}}
	nm := newMounts(oldA, newA)
	if len(nm) != 1 || nm[0] != "cache" {
		t.Errorf("newMounts = %v, want [cache]", nm)
	}
	if isMountNew(oldA, "data") {
		t.Error("data is not a new mount")
	}
	if !isMountNew(oldA, "cache") {
		t.Error("cache is a new mount")
	}
}

func TestIsTemplated(t *testing.T) {
	if !isTemplated("{{ .inputs.X }}") || isTemplated("literal") {
		t.Error("isTemplated mismatch")
	}
}
