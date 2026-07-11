// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestGuardOwned(t *testing.T) {
	// An exister that says everything exists, and one that says nothing does.
	exists := func(string, uint, uint) bool { return true }
	gone := func(string, uint, uint) bool { return false }

	appOwned := models.SetOwner(nil, models.OwnerApp, 5, "shop-web")

	cases := []struct {
		name    string
		meta    models.Metadata
		exister OwnerExister
		blocked bool
	}{
		{"no owner", models.Metadata{"team": "core"}, exists, false},
		{"user owner is never blocked", models.SetOwner(nil, models.OwnerUser, 9, "alice"), exists, false},
		{"app owner that still exists blocks", appOwned, exists, true},
		{"app owner that is gone does not block", appOwned, gone, false},
		{"no exister wired skips the guard", appOwned, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{ownerOf: tc.exister}
			err := s.guardOwned(tc.meta, 1)
			if tc.blocked {
				if !errors.Is(err, ErrVolumeOwned) {
					t.Fatalf("expected ErrVolumeOwned, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
