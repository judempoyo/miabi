// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"testing"
)

func TestResolveID(t *testing.T) {
	const uid = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
	calls := 0
	byUID := func(u string) (uint, error) {
		calls++
		if u == uid {
			return 77, nil
		}
		return 0, errors.New("not found")
	}

	// Numeric ref resolves directly, without a uid lookup.
	if id, err := resolveID("42", byUID); err != nil || id != 42 {
		t.Fatalf("numeric: got id=%d err=%v", id, err)
	}
	if calls != 0 {
		t.Fatalf("numeric ref should not call the uid resolver (calls=%d)", calls)
	}

	// A uuid ref resolves via the lookup.
	if id, err := resolveID(uid, byUID); err != nil || id != 77 {
		t.Fatalf("uid: got id=%d err=%v", id, err)
	}

	// Junk and zero are rejected without a lookup.
	for _, bad := range []string{"", "0", "-3", "not-an-id", "abc123"} {
		if _, err := resolveID(bad, byUID); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}
