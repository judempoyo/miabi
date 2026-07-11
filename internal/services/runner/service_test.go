// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package runner

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeLabels(t *testing.T) {
	got := normalizeLabels([]string{"  buildkit ", "arch=amd64", "buildkit", "", "  "})
	// trimmed, de-duplicated, blanks dropped, sorted for an order-independent
	// scheduler subset match.
	want := []string{"arch=amd64", "buildkit"}
	if len(got) != len(want) {
		t.Fatalf("normalizeLabels = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalizeLabels[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if normalizeLabels(nil) != nil || normalizeLabels([]string{" ", ""}) != nil {
		t.Error("all-blank / empty labels should normalize to nil")
	}
}

func TestNormalizeConcurrency(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 1}, {-3, 1}, {1, 1}, {5, 5}} {
		if got := normalizeConcurrency(tc.in); got != tc.want {
			t.Errorf("normalizeConcurrency(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestDisplayNameDefaultsToName(t *testing.T) {
	if got := displayName("  ", "web-builder"); got != "web-builder" {
		t.Errorf("blank display name should default to name, got %q", got)
	}
	if got := displayName(" Prod Builder ", "web-builder"); got != "Prod Builder" {
		t.Errorf("display name = %q, want trimmed label", got)
	}
}

func TestTokenGenerateAndHash(t *testing.T) {
	a := generateToken()
	b := generateToken()
	if !strings.HasPrefix(a, tokenPrefix) {
		t.Errorf("token %q missing prefix %q", a, tokenPrefix)
	}
	if a == b {
		t.Error("generateToken must not repeat")
	}
	// Hashing is deterministic (for the constant-time compare on authenticate) and
	// never returns the plaintext.
	if h1, h2 := hashToken(a), hashToken(a); h1 != h2 {
		t.Error("hashToken must be deterministic")
	}
	if hashToken(a) == a {
		t.Error("hashToken must not return the plaintext token")
	}
}

// Authenticate rejects a token without the runner prefix before touching the
// repository, so it is safe to call with no DB wired.
func TestAuthenticateRejectsBadPrefix(t *testing.T) {
	s := NewService(nil)
	if _, err := s.Authenticate("mbn_notarunner"); !errors.Is(err, ErrBadToken) {
		t.Errorf("wrong-prefix token: err = %v, want ErrBadToken", err)
	}
	if _, err := s.Authenticate(""); !errors.Is(err, ErrBadToken) {
		t.Errorf("empty token: err = %v, want ErrBadToken", err)
	}
}
