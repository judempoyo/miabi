// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package dnsprovider

import "testing"

// relName must reduce both a Miabi FQDN and a provider's already-relative name to
// the same label, so conflict detection compares like for like.
func TestRelName(t *testing.T) {
	cases := []struct{ name, zone, want string }{
		{"_miabi-challenge.example.com", "example.com", "_miabi-challenge"},
		{"_miabi-challenge.example.com.", "example.com.", "_miabi-challenge"},
		{"_miabi-challenge", "example.com", "_miabi-challenge"}, // already relative
		{"example.com", "example.com", "@"},                     // zone apex
		{"app.sub.example.com", "example.com", "app.sub"},
	}
	for _, tc := range cases {
		if got := relName(tc.name, tc.zone); got != tc.want {
			t.Errorf("relName(%q, %q) = %q, want %q", tc.name, tc.zone, got, tc.want)
		}
	}
}

// addressRecord prefers an A/AAAA from the IP, falls back to a CNAME from the
// hostname, and never emits a CNAME at the zone apex.
func TestAddressRecord(t *testing.T) {
	if r := addressRecord("blog.example.com", "example.com", "1.2.3.4", "gw.example.net"); r.Type != "A" || r.Value != "1.2.3.4" {
		t.Errorf("ipv4 => %+v, want A 1.2.3.4", r)
	}
	if r := addressRecord("blog.example.com", "example.com", "2001:db8::1", ""); r.Type != "AAAA" {
		t.Errorf("ipv6 => %+v, want AAAA", r)
	}
	if r := addressRecord("blog.example.com", "example.com", "", "gw.example.net"); r.Type != "CNAME" || r.Value != "gw.example.net." {
		t.Errorf("hostname => %+v, want CNAME gw.example.net.", r)
	}
	if r := addressRecord("example.com", "example.com", "", "gw.example.net"); r.Type != "" {
		t.Errorf("apex CNAME => %+v, want none (CNAME forbidden at apex)", r)
	}
	if r := addressRecord("blog.example.com", "example.com", "", ""); r.Type != "" {
		t.Errorf("no target => %+v, want none", r)
	}
}
