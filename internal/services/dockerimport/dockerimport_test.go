// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package dockerimport

import "testing"

func TestSplitImageTag(t *testing.T) {
	cases := []struct {
		ref, image, tag string
	}{
		{"nginx", "nginx", ""},
		{"nginx:1.25", "nginx", "1.25"},
		{"ghcr.io/org/app:v2", "ghcr.io/org/app", "v2"},
		{"registry:5000/app", "registry:5000/app", ""}, // ":" before last "/" is a port, not a tag
		{"registry:5000/app:dev", "registry:5000/app", "dev"},
		{"repo@sha256:abc", "repo@sha256:abc", ""}, // digest-pinned
	}
	for _, c := range cases {
		gotImage, gotTag := splitImageTag(c.ref)
		if gotImage != c.image || gotTag != c.tag {
			t.Errorf("splitImageTag(%q) = (%q, %q), want (%q, %q)", c.ref, gotImage, gotTag, c.image, c.tag)
		}
	}
}

func TestIsSecretKey(t *testing.T) {
	secret := []string{"DB_PASSWORD", "API_TOKEN", "JWT_SECRET", "AWS_ACCESS_KEY", "STRIPE_APIKEY", "TLS_PRIVATE_KEY", "SIGNING_KEY", "KEY"}
	plain := []string{"PORT", "NODE_ENV", "HOSTNAME", "DEBUG", "KEYBOARD_LAYOUT"}
	for _, k := range secret {
		if !isSecretKey(k) {
			t.Errorf("isSecretKey(%q) = false, want true", k)
		}
	}
	for _, k := range plain {
		if isSecretKey(k) {
			t.Errorf("isSecretKey(%q) = true, want false", k)
		}
	}
}

func TestIsMiabiName(t *testing.T) {
	managed := []string{"mb-vol-1-data", "mb-ws3-abc", "miabi", "mb-stack-foo"}
	external := []string{"my_data", "postgres_data", "app-network", "redis"}
	for _, n := range managed {
		if !isMiabiName(n) {
			t.Errorf("isMiabiName(%q) = false, want true", n)
		}
	}
	for _, n := range external {
		if isMiabiName(n) {
			t.Errorf("isMiabiName(%q) = true, want false", n)
		}
	}
}

func TestIsManaged(t *testing.T) {
	if !isManaged(map[string]string{"io.miabi.app": "1"}) {
		t.Error("expected io.miabi.app to be managed")
	}
	if isManaged(map[string]string{"com.docker.compose.project": "blog"}) {
		t.Error("expected a compose-only container to be unmanaged")
	}
}
