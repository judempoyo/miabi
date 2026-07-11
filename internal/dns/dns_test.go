// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package dns

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestBuild(t *testing.T) {
	cases := []struct {
		name    string
		typ     string
		creds   Credentials
		wantErr bool
	}{
		{"cloudflare ok", models.DNSProviderCloudflare, Credentials{APIToken: "t"}, false},
		{"cloudflare missing token", models.DNSProviderCloudflare, Credentials{}, true},
		{"digitalocean ok", models.DNSProviderDigitalOcean, Credentials{APIToken: "t"}, false},
		{"route53 ok", models.DNSProviderRoute53, Credentials{AccessKeyID: "a", SecretAccessKey: "s", Region: "us-east-1"}, false},
		{"route53 missing secret", models.DNSProviderRoute53, Credentials{AccessKeyID: "a"}, true},
		{"unknown type", "googledns", Credentials{APIToken: "t"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Build(tc.typ, tc.creds)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got provider %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p == nil {
				t.Fatal("expected a provider, got nil")
			}
		})
	}
}

func TestToRRRelativizesName(t *testing.T) {
	a := &adapter{}
	rr := a.toRR("example.com.", Record{Type: "TXT", Name: "_miabi-challenge.example.com", Value: "v"})
	if rr.Name != "_miabi-challenge" {
		t.Errorf("name = %q, want %q (relative to zone)", rr.Name, "_miabi-challenge")
	}
	if rr.Type != "TXT" || rr.Data != "v" {
		t.Errorf("unexpected RR %+v", rr)
	}
}

func TestCanonicalZone(t *testing.T) {
	for in, want := range map[string]string{"example.com": "example.com.", "example.com.": "example.com.", "": "."} {
		if got := canonicalZone(in); got != want {
			t.Errorf("canonicalZone(%q) = %q, want %q", in, got, want)
		}
	}
}
