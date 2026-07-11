// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package slug

import (
	"regexp"
	"testing"
)

func TestToken(t *testing.T) {
	dnsSafe := regexp.MustCompile(`^[a-z0-9]+$`)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok := Token(8)
		if len(tok) != 8 {
			t.Fatalf("Token(8) length = %d, want 8", len(tok))
		}
		if !dnsSafe.MatchString(tok) {
			t.Fatalf("Token(8) = %q is not lowercase-alphanumeric", tok)
		}
		seen[tok] = true
	}
	// Astronomically unlikely to collide across 100 draws of 36^8 space.
	if len(seen) < 99 {
		t.Errorf("Token(8) produced too many collisions: %d unique of 100", len(seen))
	}
	if Token(0) != "" {
		t.Error("Token(0) should be empty")
	}
}

func TestIsValid(t *testing.T) {
	valid := []string{"api", "my-api", "basic-auth", "a", "web1", "x-y-z"}
	for _, s := range valid {
		if !IsValid(s) {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "My API", "Api", "-api", "api-", "a--b", "api_v2", "café", "api "}
	for _, s := range invalid {
		if IsValid(s) {
			t.Errorf("IsValid(%q) = true, want false", s)
		}
	}
}

func TestIsReserved(t *testing.T) {
	for _, s := range []string{"system", "admin", "api", "registry", "workspaces", "me", "SYSTEM", "Admin "} {
		if !IsReserved(s) {
			t.Errorf("IsReserved(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"acme", "my-team", "web1", "billing"} {
		if IsReserved(s) {
			t.Errorf("IsReserved(%q) = true, want false", s)
		}
	}
}

func TestIsAvailableHandle(t *testing.T) {
	// Valid and non-reserved.
	for _, s := range []string{"acme", "my-team", "web1"} {
		if !IsAvailableHandle(s) {
			t.Errorf("IsAvailableHandle(%q) = false, want true", s)
		}
	}
	// Reserved or malformed.
	for _, s := range []string{"system", "admin", "Acme", "bad name", ""} {
		if IsAvailableHandle(s) {
			t.Errorf("IsAvailableHandle(%q) = true, want false", s)
		}
	}
}

func TestUniqueAvailable(t *testing.T) {
	// "admin" is reserved, so the base must be bumped past it; "admin-1" is free.
	taken := map[string]bool{"acme": true, "acme-1": true}
	exists := func(s string) (bool, error) { return taken[s], nil }

	got, err := UniqueAvailable("Acme", "workspace", exists)
	if err != nil {
		t.Fatal(err)
	}
	if got != "acme-2" {
		t.Errorf("UniqueAvailable(Acme) = %q, want acme-2", got)
	}

	got, err = UniqueAvailable("admin", "workspace", exists)
	if err != nil {
		t.Fatal(err)
	}
	if IsReserved(got) {
		t.Errorf("UniqueAvailable(admin) returned reserved handle %q", got)
	}
	if got != "admin-1" {
		t.Errorf("UniqueAvailable(admin) = %q, want admin-1", got)
	}

	// An empty/strippable base falls back, and the fallback is never reserved.
	got, err = UniqueAvailable("!!!", "workspace", func(string) (bool, error) { return false, nil })
	if err != nil {
		t.Fatal(err)
	}
	if got != "workspace" {
		t.Errorf("UniqueAvailable(!!!) = %q, want workspace", got)
	}
}
