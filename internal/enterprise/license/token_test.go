// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package license

import (
	"testing"
	"time"
)

func sampleClaims(now time.Time) Claims {
	return Claims{
		LicenseID: "lic_test", Customer: "Acme GmbH", Edition: "enterprise",
		Flags:     []string{"sso_saml", "ha"},
		Limits:    map[string]int{"node_limit": 8},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(365 * 24 * time.Hour),
		GraceDays: 14, IssuedAt: now,
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	now := time.Now()
	tok, err := Sign(priv, sampleClaims(now))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	got, err := Verify(pub, tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.LicenseID != "lic_test" || got.Edition != "enterprise" || got.Limits["node_limit"] != 8 {
		t.Fatalf("claims round-trip mismatch: %+v", got)
	}
}

func TestVerifyTamperFails(t *testing.T) {
	pub, priv, _ := GenerateKey()
	tok, _ := Sign(priv, sampleClaims(time.Now()))
	// Flip one byte in the payload segment.
	b := []byte(tok)
	mid := len(b) / 2
	b[mid] ^= 0x01
	if _, err := Verify(pub, string(b)); err == nil {
		t.Fatal("expected verification to fail on a tampered token")
	}
}

func TestVerifyWrongKeyFails(t *testing.T) {
	_, priv, _ := GenerateKey()
	otherPub, _, _ := GenerateKey()
	tok, _ := Sign(priv, sampleClaims(time.Now()))
	if _, err := Verify(otherPub, tok); err != ErrBadSignature {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
}

func TestVerifyMalformed(t *testing.T) {
	pub, _, _ := GenerateKey()
	for _, tok := range []string{"", "nope", "miabi-v1.onlytwo", "wrong-prefix.a.b"} {
		if _, err := Verify(pub, tok); err == nil {
			t.Fatalf("expected error for %q", tok)
		}
	}
}

func TestEvaluateStates(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	base := Claims{Edition: "enterprise", Flags: []string{"ha"}, GraceDays: 14}

	cases := []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
		want      State
	}{
		{"valid", now.Add(-24 * time.Hour), now.Add(24 * time.Hour), StateValid},
		{"grace", now.Add(-48 * time.Hour), now.Add(-1 * time.Hour), StateGrace},
		{"degraded", now.Add(-100 * 24 * time.Hour), now.Add(-30 * 24 * time.Hour), StateDegraded},
		{"not-yet-active", now.Add(24 * time.Hour), now.Add(48 * time.Hour), StateNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			c.NotBefore, c.NotAfter = tc.notBefore, tc.notAfter
			got := Evaluate(c, now)
			if got.State != tc.want {
				t.Fatalf("state = %q, want %q", got.State, tc.want)
			}
			if !got.Flags["ha"] {
				t.Fatal("expected ha flag set in snapshot")
			}
		})
	}
}
