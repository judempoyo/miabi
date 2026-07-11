// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package workspace

import (
	"testing"

	"github.com/miabi-io/miabi/internal/slug"
)

// The workspace handle grammar is now the shared slug.Make; this guards the
// behaviour the service relies on (lowercasing, single hyphens, trimming).
func TestHandleGrammar(t *testing.T) {
	cases := map[string]string{
		"Acme Prod":        "acme-prod",
		"  Hello World!  ": "hello-world",
		"already-slug":     "already-slug",
		"Foo___Bar":        "foo-bar",
		"--edge--":         "edge",
	}
	for in, want := range cases {
		if got := slug.Make(in, ""); got != want {
			t.Errorf("slug.Make(%q) = %q, want %q", in, got, want)
		}
	}
}

// The reserved set protects the system handle and the route/registry namespace,
// so SetName rejects them (see slug.IsReserved). "system" is the system
// workspace's own handle and must be reserved for everyone else.
func TestReservedHandlesRejected(t *testing.T) {
	for _, h := range []string{"system", "admin", "api", "registry", "workspaces", "me"} {
		if !slug.IsReserved(h) {
			t.Errorf("expected %q to be reserved", h)
		}
	}
	if slug.IsReserved("acme") {
		t.Error("acme should not be reserved")
	}
}
