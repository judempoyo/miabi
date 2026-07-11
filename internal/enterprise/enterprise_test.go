//go:build enterprise

// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: LicenseRef-Miabi-Enterprise

package enterprise

import (
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/enterprise/license"
)

// withClaims builds an impl with a license valid/grace/degraded relative to now,
// bypassing the DB (only snapshot()/gate logic is exercised).
func withClaims(notAfter time.Time, flags ...string) *impl {
	return &impl{claims: &license.Claims{
		LicenseID: "lic_t", Edition: EditionEnterprise, Flags: flags,
		Limits:    map[string]int{"node_limit": 4},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter, GraceDays: 14,
	}}
}

func TestGate_Valid(t *testing.T) {
	e := withClaims(time.Now().Add(48*time.Hour), FlagSSOSAML)
	if !e.Has(FlagSSOSAML) || !e.Mutable(FlagSSOSAML) {
		t.Fatal("entitled flag should be usable and mutable while valid")
	}
	if err := e.Require(FlagSSOSAML); err != nil {
		t.Fatalf("Require: %v", err)
	}
	if err := e.RequireMutable(FlagSSOSAML); err != nil {
		t.Fatalf("RequireMutable: %v", err)
	}
	// Entitled license but missing flag → entitlement denied (403), not 402.
	if err := e.Require(FlagHA); err != ErrEntitlementDenied {
		t.Fatalf("missing flag: got %v, want ErrEntitlementDenied", err)
	}
}

func TestGate_Degraded_ReadOnly(t *testing.T) {
	// Expired beyond grace: runtime use still works, but config is frozen.
	e := withClaims(time.Now().Add(-30*24*time.Hour), FlagSSOSAML)
	if e.Entitlements().State != "degraded" {
		t.Fatalf("state = %s, want degraded", e.Entitlements().State)
	}
	if !e.Has(FlagSSOSAML) {
		t.Fatal("degraded license should keep entitled features running (Has=true)")
	}
	if e.Mutable(FlagSSOSAML) {
		t.Fatal("degraded license must freeze config (Mutable=false)")
	}
	if err := e.RequireMutable(FlagSSOSAML); err != ErrLicenseExpired {
		t.Fatalf("RequireMutable on degraded: got %v, want ErrLicenseExpired", err)
	}
}

func TestGate_Community(t *testing.T) {
	e := &impl{} // no claims
	if e.Has(FlagSSOSAML) || e.Mutable(FlagSSOSAML) {
		t.Fatal("community must deny everything")
	}
	if err := e.Require(FlagSSOSAML); err != ErrLicenseRequired {
		t.Fatalf("community Require: got %v, want ErrLicenseRequired", err)
	}
	if e.Entitlements().Edition != EditionCommunity {
		t.Fatalf("edition = %s, want community", e.Entitlements().Edition)
	}
}
