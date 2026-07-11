// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package oauth

import "testing"

func TestFirstClaim(t *testing.T) {
	claims := map[string]any{
		"email":      "  std@example.com  ",
		"work_email": "mapped@example.com",
		"num":        42, // non-string ignored
	}
	cases := []struct {
		name  string
		names []string
		want  string
	}{
		{"mapped claim wins", []string{"work_email", "email"}, "mapped@example.com"},
		{"falls back to standard", []string{"", "email"}, "std@example.com"},
		{"trims whitespace", []string{"email"}, "std@example.com"},
		{"missing → empty", []string{"nope"}, ""},
		{"non-string ignored", []string{"num", "email"}, "std@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := firstClaim(claims, tc.names...); got != tc.want {
				t.Fatalf("firstClaim(%v) = %q, want %q", tc.names, got, tc.want)
			}
		})
	}
}
