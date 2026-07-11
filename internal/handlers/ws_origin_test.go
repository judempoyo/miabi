// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"testing"
)

func req(origin, host string) *http.Request {
	r := &http.Request{Host: host, Header: http.Header{}}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

func TestAllowWSOrigin(t *testing.T) {
	allow := allowWSOrigin([]string{"https://panel.example.com"})
	cases := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{"no origin (go client)", "", "panel.example.com", true},
		{"same-origin", "https://panel.example.com", "panel.example.com", true},
		{"allowlisted origin", "https://panel.example.com", "api.internal", true},
		{"allowlisted trailing slash", "https://panel.example.com/", "api.internal", true},
		{"cross-site origin rejected", "https://evil.example.com", "panel.example.com", false},
		{"malformed origin rejected", "not-a-url", "panel.example.com", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := allow(req(tt.origin, tt.host)); got != tt.want {
				t.Errorf("allow(origin=%q host=%q) = %v, want %v", tt.origin, tt.host, got, tt.want)
			}
		})
	}
}

func TestAllowWSOriginWildcardDisablesCheck(t *testing.T) {
	allow := allowWSOrigin([]string{"*"})
	if !allow(req("https://evil.example.com", "panel.example.com")) {
		t.Error("wildcard allowlist should permit any origin (dev)")
	}
}
