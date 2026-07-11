// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestNormalizeAndValidate(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		valid bool
	}{
		{"Example.COM", "example.com", true},
		{"  shop.example.com. ", "shop.example.com", true},
		{"*.example.com", "example.com", true}, // wildcard prefix stripped
		{"a.b.co", "a.b.co", true},
		{"nodot", "nodot", false},
		{"bad_underscore.com", "bad_underscore.com", false},
		{"", "", false},
	}
	for _, c := range cases {
		in := Input{Name: c.in}
		in.normalize()
		if in.Name != c.want {
			t.Errorf("normalize(%q) = %q, want %q", c.in, in.Name, c.want)
		}
		if c.want != "" && validName(in.Name) != c.valid {
			t.Errorf("validName(%q) = %v, want %v", in.Name, validName(in.Name), c.valid)
		}
	}
}

func TestNormalizeDefaultsTLS(t *testing.T) {
	in := Input{Name: "example.com"}
	in.normalize()
	if in.TLSMode != models.DomainTLSACME {
		t.Errorf("default TLS = %q, want acme", in.TLSMode)
	}
	in = Input{Name: "example.com", TLSMode: models.DomainTLSCustom}
	in.normalize()
	if in.TLSMode != models.DomainTLSCustom {
		t.Errorf("custom TLS not preserved: %q", in.TLSMode)
	}
}

func TestChallenge(t *testing.T) {
	d := &models.Domain{Name: "example.com", VerificationToken: "abc123"}
	if got := d.ChallengeHost(); got != "_miabi-challenge.example.com" {
		t.Errorf("ChallengeHost = %q", got)
	}
	if got := d.ChallengeValue(); got != "miabi-verification=abc123" {
		t.Errorf("ChallengeValue = %q", got)
	}
}

func TestDomainsOverlap(t *testing.T) {
	dom := func(name string, wildcard bool) *models.Domain {
		return &models.Domain{Name: name, Wildcard: wildcard}
	}
	cases := []struct {
		name string
		a, b *models.Domain
		want bool
	}{
		{"exact match", dom("example.com", false), dom("example.com", false), true},
		{"case-insensitive exact", dom("Example.COM", false), dom("example.com", false), true},
		{"unrelated names", dom("example.com", false), dom("other.com", false), false},
		{"a wildcard covers b subdomain", dom("example.com", true), dom("app.example.com", false), true},
		{"b wildcard covers a subdomain", dom("app.example.com", false), dom("example.com", true), true},
		{"no wildcard, subdomain relation", dom("example.com", false), dom("app.example.com", false), false},
		{"wildcard but sibling, not subdomain", dom("example.com", true), dom("example.org", false), false},
		{"wildcard does not cover parent's parent", dom("a.example.com", true), dom("example.com", false), false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := domainsOverlap(tt.a, tt.b); got != tt.want {
				t.Errorf("domainsOverlap(%q wc=%v, %q wc=%v) = %v, want %v",
					tt.a.Name, tt.a.Wildcard, tt.b.Name, tt.b.Wildcard, got, tt.want)
			}
		})
	}
}
