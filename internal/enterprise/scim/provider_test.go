//go:build enterprise

// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: LicenseRef-Miabi-Enterprise

package scim

import "testing"

func TestParseUserNameFilter(t *testing.T) {
	cases := map[string]string{
		`userName eq "alice@example.com"`: "alice@example.com",
		`userName Eq "BOB@EXAMPLE.COM"`:   "bob@example.com",
		`displayName eq "x"`:              "",
		`garbage`:                         "",
		``:                                "",
	}
	for in, want := range cases {
		if got := parseUserNameFilter(in); got != want {
			t.Errorf("parseUserNameFilter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmailAndName(t *testing.T) {
	u := scimUser{UserName: "  USER@Example.com ", Emails: []scimEmail{{Value: "alt@x.com"}}}
	if got := emailOf(u); got != "user@example.com" {
		t.Fatalf("emailOf userName = %q", got)
	}
	if got := emailOf(scimUser{Emails: []scimEmail{{Value: "Alt@X.com"}}}); got != "alt@x.com" {
		t.Fatalf("emailOf fallback = %q", got)
	}
	if got := nameOf(scimUser{}, "e@x.com"); got != "e@x.com" {
		t.Fatalf("nameOf fallback = %q", got)
	}
	if got := nameOf(scimUser{DisplayName: "Dee"}, "e@x.com"); got != "Dee" {
		t.Fatalf("nameOf displayName = %q", got)
	}
}
