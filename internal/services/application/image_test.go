// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package application

import "testing"

func TestNormalizeImageTag(t *testing.T) {
	cases := []struct {
		name      string
		image     string
		tag       string
		wantImage string
		wantTag   string
	}{
		{"image with tag, no tag field", "nginx:1.2", "", "nginx", "1.2"},
		{"bare image, tag field", "nginx", "v2", "nginx", "v2"},
		{"bare image, no tag", "nginx", "", "nginx", ""},
		{"registry path with tag", "ghcr.io/org/app:v2", "", "ghcr.io/org/app", "v2"},
		{"host:port and tag", "registry:5000/app:1.2.3", "", "registry:5000/app", "1.2.3"},
		{"embedded tag wins over tag field", "nginx:1.2", "v9", "nginx", "1.2"},
		{"digest pin kept intact", "repo@sha256:abc", "", "repo@sha256:abc", ""},
		{"whitespace trimmed", "  nginx:1.2  ", "  ", "nginx", "1.2"},
		{"empty image", "", "v2", "", "v2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			img, tag := normalizeImageTag(c.image, c.tag)
			if img != c.wantImage || tag != c.wantTag {
				t.Errorf("normalizeImageTag(%q, %q) = (%q, %q), want (%q, %q)", c.image, c.tag, img, tag, c.wantImage, c.wantTag)
			}
		})
	}
}
