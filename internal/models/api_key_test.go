// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "testing"

func TestIsRegistryOnly(t *testing.T) {
	cases := []struct {
		scopes []string
		want   bool
	}{
		{nil, false}, // empty defaults to read (general)
		{[]string{ScopeRead}, false},
		{[]string{ScopeWrite}, false},
		{[]string{ScopeAll}, false},
		{[]string{ScopeRegistryRead}, true},
		{[]string{ScopeRegistryWrite}, true},
		{[]string{ScopeRegistryRead, ScopeRegistryWrite}, true},
		{[]string{ScopeRegistryWrite, ScopeRead}, false}, // mixed → general access too
	}
	for _, tc := range cases {
		k := &APIKey{Scopes: tc.scopes}
		if got := k.IsRegistryOnly(); got != tc.want {
			t.Errorf("IsRegistryOnly(%v) = %v, want %v", tc.scopes, got, tc.want)
		}
	}
}
