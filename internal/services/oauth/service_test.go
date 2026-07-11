// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import "testing"

func TestDomainAllowed(t *testing.T) {
	cases := []struct {
		name    string
		email   string
		allowed string
		want    bool
	}{
		{"empty allowlist permits any", "a@example.com", "", true},
		{"exact match", "a@example.com", "example.com", true},
		{"case insensitive", "A@Example.COM", "example.com", true},
		{"one of several", "a@acme.org", "example.com, acme.org", true},
		{"not in list", "a@evil.com", "example.com,acme.org", false},
		{"no at sign", "invalid", "example.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domainAllowed(tc.email, tc.allowed); got != tc.want {
				t.Errorf("domainAllowed(%q, %q) = %v, want %v", tc.email, tc.allowed, got, tc.want)
			}
		})
	}
}

func TestScopesOrDefault(t *testing.T) {
	if got := scopesOrDefault(""); got != "openid email profile" {
		t.Errorf("expected default scopes, got %q", got)
	}
	if got := scopesOrDefault("openid"); got != "openid" {
		t.Errorf("expected passthrough, got %q", got)
	}
}
