// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"testing"

	"github.com/miabi-io/miabi/internal/enterprise"
)

// fakeEE is a test double for enterprise.EE that grants a fixed flag set.
type fakeEE struct{ flags map[string]bool }

func (f fakeEE) Entitlements() enterprise.Entitlements { return enterprise.Entitlements{} }
func (f fakeEE) Has(flag string) bool                  { return f.flags[flag] }
func (f fakeEE) Mutable(flag string) bool              { return f.flags[flag] }
func (f fakeEE) Require(string) error                  { return nil }
func (f fakeEE) RequireMutable(string) error           { return nil }
func (f fakeEE) Install(context.Context, string, string) (enterprise.Entitlements, error) {
	return enterprise.Entitlements{}, nil
}
func (f fakeEE) Remove(context.Context) error       { return nil }
func (f fakeEE) InitSSO(enterprise.SSODeps)         {}
func (f fakeEE) SAML() enterprise.SAMLProvider      { return nil }
func (f fakeEE) SCIM() enterprise.SCIMProvider      { return nil }
func (f fakeEE) LDAP() enterprise.LDAPAuthenticator { return nil }

func grant(flags ...string) fakeEE {
	m := map[string]bool{}
	for _, f := range flags {
		m[f] = true
	}
	return fakeEE{flags: m}
}

func TestProviderCountAllowed(t *testing.T) {
	cases := []struct {
		name    string
		ee      enterprise.EE
		current int64
		want    bool
	}{
		{"community first provider", nil, 0, true},
		{"community second provider blocked", nil, 1, false},
		{"nil ee treated as community", grant() /* no flags */, 1, false},
		{"multi_sso lifts the cap", grant(enterprise.FlagMultiSSO), 5, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := providerCountAllowed(tc.ee, tc.current); got != tc.want {
				t.Fatalf("providerCountAllowed(%d) = %v, want %v", tc.current, got, tc.want)
			}
		})
	}
}

func TestHiddenAllowed(t *testing.T) {
	if hiddenAllowed(nil) {
		t.Fatal("nil ee (community) must not allow hidden providers")
	}
	if hiddenAllowed(grant()) {
		t.Fatal("community (no entitlement) must not allow hidden providers")
	}
	if !hiddenAllowed(grant(enterprise.FlagSSOHiddenProvider)) {
		t.Fatal("sso_hidden_provider entitlement must allow hidden providers")
	}
}
