// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package gitrepo

import "testing"

func TestNormalizeGitURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/acme/api":     "https://github.com/acme/api.git",
		"https://github.com/acme/api.git": "https://github.com/acme/api.git",
		"https://github.com/acme/api/":    "https://github.com/acme/api.git",
		"  https://github.com/acme/api  ": "https://github.com/acme/api.git",
		"git@github.com:acme/api":         "git@github.com:acme/api.git",
		"git@github.com:acme/api.git":     "git@github.com:acme/api.git",
		"https://github.com/acme/api.GIT": "https://github.com/acme/api.GIT", // case-insensitive suffix check
		"":                                "",
	}
	for in, want := range cases {
		if got := normalizeGitURL(in); got != want {
			t.Errorf("normalizeGitURL(%q) = %q, want %q", in, got, want)
		}
	}
}
