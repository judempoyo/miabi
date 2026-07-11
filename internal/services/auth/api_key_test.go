// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package auth

import (
	"reflect"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestNormalizeScopes(t *testing.T) {
	cases := []struct {
		name    string
		in      []string
		want    []string
		wantErr bool
	}{
		{"nil defaults to read", nil, []string{models.ScopeRead}, false},
		{"empty defaults to read", []string{}, []string{models.ScopeRead}, false},
		{"passes through valid", []string{models.ScopeWrite, models.ScopeDeploy}, []string{models.ScopeWrite, models.ScopeDeploy}, false},
		{"dedupes", []string{models.ScopeRead, models.ScopeRead, models.ScopeWrite}, []string{models.ScopeRead, models.ScopeWrite}, false},
		{"wildcard is valid", []string{models.ScopeAll}, []string{models.ScopeAll}, false},
		{"registry scopes are valid", []string{models.ScopeRegistryRead, models.ScopeRegistryWrite}, []string{models.ScopeRegistryRead, models.ScopeRegistryWrite}, false},
		{"unknown scope errors", []string{"delete-everything"}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeScopes(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeScopes(%v) = %v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeScopes(%v) unexpected error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NormalizeScopes(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestTokenFormat asserts the wire format the auth middleware keys off: a
// mb_-prefixed token whose stored lookup prefix is the first 11 characters.
func TestTokenFormat(t *testing.T) {
	raw, _ := generateToken()
	plaintext := keyPrefix + raw

	if plaintext[:prefixLen] != keyPrefix+raw[:8] {
		t.Errorf("prefix = %q, want %q", plaintext[:prefixLen], keyPrefix+raw[:8])
	}
	// The full token must hash deterministically and differently from the secret
	// alone (the legacy scheme), guarding against a silent format regression.
	if hashToken(plaintext) == hashToken(raw) {
		t.Error("full-token hash must differ from secret-only hash")
	}
}
