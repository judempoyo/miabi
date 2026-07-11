// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "testing"

func TestSplitImageRef(t *testing.T) {
	cases := []struct {
		ref       string
		wantImage string
		wantTag   string
	}{
		{"nginx:latest", "nginx", "latest"},
		{"nginx", "nginx", ""},
		{"ghcr.io/org/app:v2", "ghcr.io/org/app", "v2"},
		{"registry:5000/app:1.2.3", "registry:5000/app", "1.2.3"}, // port in host, tag at end
		{"registry:5000/app", "registry:5000/app", ""},            // port in host, no tag
		{"repo@sha256:abc", "repo@sha256:abc", ""},                // digest pin
		{"", "", ""},
	}
	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			img, tag := SplitImageRef(c.ref)
			if img != c.wantImage || tag != c.wantTag {
				t.Errorf("SplitImageRef(%q) = (%q,%q), want (%q,%q)", c.ref, img, tag, c.wantImage, c.wantTag)
			}
		})
	}
}

// Round-trip: composing then splitting yields the original repo + a tag.
func TestSplitComposeRoundTrip(t *testing.T) {
	img, tag := SplitImageRef(ComposeImageRef("nginx", "v2"))
	if img != "nginx" || tag != "v2" {
		t.Errorf("round-trip = (%q,%q), want (nginx,v2)", img, tag)
	}
	// No tag composes to :latest and splits back to latest.
	img, tag = SplitImageRef(ComposeImageRef("nginx", ""))
	if img != "nginx" || tag != "latest" {
		t.Errorf("round-trip default = (%q,%q), want (nginx,latest)", img, tag)
	}
}
