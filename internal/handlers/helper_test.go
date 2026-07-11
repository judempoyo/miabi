// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"testing"
)

// TestOptionalUintRef covers the tri-state a partial update relies on so a
// spec-only PATCH leaves a pipeline's app binding untouched (absent), while an
// explicit null still unbinds and a number rebinds.
func TestOptionalUintRef(t *testing.T) {
	cases := []struct {
		name        string
		raw         json.RawMessage
		wantPresent bool
		wantVal     *uint
		wantErr     bool
	}{
		{"absent (nil)", nil, false, nil, false},
		{"absent (empty)", json.RawMessage(""), false, nil, false},
		{"absent (whitespace)", json.RawMessage("  "), false, nil, false},
		{"null clears", json.RawMessage("null"), true, nil, false},
		{"number binds", json.RawMessage("42"), true, uptr(42), false},
		{"number with spaces", json.RawMessage(" 7 "), true, uptr(7), false},
		{"zero is invalid", json.RawMessage("0"), true, nil, true},
		{"garbage is invalid", json.RawMessage(`"x"`), true, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			present, val, err := optionalUintRef(tc.raw)
			if present != tc.wantPresent {
				t.Errorf("present = %v, want %v", present, tc.wantPresent)
			}
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tc.wantErr)
			}
			switch {
			case tc.wantVal == nil && val != nil:
				t.Errorf("value = %d, want nil", *val)
			case tc.wantVal != nil && (val == nil || *val != *tc.wantVal):
				t.Errorf("value = %v, want %d", val, *tc.wantVal)
			}
		})
	}
}

func uptr(n uint) *uint { return &n }
