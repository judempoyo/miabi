// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package image

import (
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/models"
)

func TestRetentionDefaults(t *testing.T) {
	p := RetentionPolicy{}.withDefaults()
	if p.KeepPerApp != 10 {
		t.Errorf("KeepPerApp default = %d, want 10", p.KeepPerApp)
	}
	if p.MinAge != 24*time.Hour {
		t.Errorf("MinAge default = %v, want 24h", p.MinAge)
	}
	// Explicit values are preserved.
	p = RetentionPolicy{KeepPerApp: 3, MinAge: time.Hour}.withDefaults()
	if p.KeepPerApp != 3 || p.MinAge != time.Hour {
		t.Errorf("explicit policy mutated: %+v", p)
	}
}

func TestProtectedHolds(t *testing.T) {
	p := protectedRefs{
		ids:     map[uint]bool{7: true},
		digests: map[string]bool{"sha256:live": true},
		refs:    map[string]bool{"miabi/app-1@sha256:pinned": true},
	}
	cases := []struct {
		name string
		img  models.Image
		want bool
	}{
		{"by id", models.Image{ID: 7, Digest: "sha256:other", Repository: "x"}, true},
		{"by digest", models.Image{ID: 99, Digest: "sha256:live", Repository: "x"}, true},
		{"by ref", models.Image{ID: 99, Repository: "miabi/app-1", Digest: "sha256:pinned"}, true},
		{"orphan", models.Image{ID: 42, Digest: "sha256:orphan", Repository: "miabi/app-9"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := p.holds(&c.img); got != c.want {
				t.Errorf("holds(%+v) = %v, want %v", c.img, got, c.want)
			}
		})
	}
}
