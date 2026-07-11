// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package runner

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func ptr(u uint) *uint { return &u }

func TestLabelsSatisfy(t *testing.T) {
	cases := []struct {
		name           string
		have, required []string
		want           bool
	}{
		{"no requirement matches any", []string{"arch=amd64"}, nil, true},
		{"exact subset", []string{"arch=amd64", "buildkit", "gpu"}, []string{"arch=amd64", "gpu"}, true},
		{"missing one fails", []string{"arch=amd64"}, []string{"arch=amd64", "gpu"}, false},
		{"empty runner cannot satisfy a requirement", nil, []string{"buildkit"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := labelsSatisfy(tc.have, tc.required); got != tc.want {
				t.Errorf("labelsSatisfy(%v,%v) = %v, want %v", tc.have, tc.required, got, tc.want)
			}
		})
	}
}

func TestInScope(t *testing.T) {
	owned := &models.Runner{WorkspaceID: ptr(42), Scope: models.ScopeWorkspace}
	shared := &models.Runner{WorkspaceID: nil, Scope: models.ScopeShared}
	if !inScope(owned, 42) || inScope(owned, 7) {
		t.Error("owned runner is in scope only for its own workspace")
	}
	if !inScope(shared, 42) || !inScope(shared, 7) {
		t.Error("shared runner is in scope for any workspace")
	}
}

func TestEligible(t *testing.T) {
	base := func() *models.Runner {
		return &models.Runner{
			WorkspaceID: ptr(1), Scope: models.ScopeWorkspace,
			Enabled: true, Cordoned: false, Labels: []string{"arch=amd64", "buildkit"},
		}
	}
	if !eligible(base(), 1, []string{"buildkit"}, true) {
		t.Error("a connected, enabled, in-scope, label-matching runner should be eligible")
	}
	// Each disqualifier independently makes it ineligible.
	r := base()
	r.Enabled = false
	if eligible(r, 1, nil, true) {
		t.Error("disabled runner must be ineligible")
	}
	r = base()
	r.Cordoned = true
	if eligible(r, 1, nil, true) {
		t.Error("cordoned runner must be ineligible")
	}
	if eligible(base(), 1, nil, false) {
		t.Error("disconnected runner must be ineligible")
	}
	if eligible(base(), 2, nil, true) {
		t.Error("out-of-scope runner must be ineligible")
	}
	if eligible(base(), 1, []string{"gpu"}, true) {
		t.Error("runner missing a required label must be ineligible")
	}
}

// pickLeastLoaded mirrors SelectRunner's choice logic over an in-memory
// candidate set + load map, so the selection policy is covered without a DB.
func pickLeastLoaded(candidates []models.Runner, workspaceID uint, required []string, loads map[uint]int, connected func(uint) bool) *models.Runner {
	var best *models.Runner
	bestLoad := 0
	for i := range candidates {
		r := &candidates[i]
		if !eligible(r, workspaceID, required, connected(r.ID)) {
			continue
		}
		load := loads[r.ID]
		if load >= r.Concurrency {
			continue
		}
		if best == nil || load < bestLoad || (load == bestLoad && r.ID < best.ID) {
			best, bestLoad = r, load
		}
	}
	return best
}

func TestSelectionPrefersLeastLoadedWithCapacity(t *testing.T) {
	runners := []models.Runner{
		{ID: 1, WorkspaceID: ptr(1), Scope: models.ScopeWorkspace, Enabled: true, Concurrency: 2, Labels: []string{"buildkit"}},
		{ID: 2, WorkspaceID: ptr(1), Scope: models.ScopeWorkspace, Enabled: true, Concurrency: 2, Labels: []string{"buildkit"}},
		{ID: 3, WorkspaceID: ptr(1), Scope: models.ScopeWorkspace, Enabled: true, Concurrency: 1, Labels: []string{"buildkit"}},
	}
	always := func(uint) bool { return true }

	// #2 has the fewest active leases → chosen.
	loads := map[uint]int{1: 1, 2: 0, 3: 0}
	// #3 is also at 0 but ties break to the lower id... #2 (0) vs #3 (0): equal,
	// lower id 2 wins over 3; #1 at 1 loses. So #2.
	if got := pickLeastLoaded(runners, 1, []string{"buildkit"}, loads, always); got == nil || got.ID != 2 {
		t.Fatalf("least-loaded selection = %v, want runner 2", got)
	}

	// Saturate #1 and #2; only #3 has spare capacity (0 < 1).
	full := map[uint]int{1: 2, 2: 2, 3: 0}
	if got := pickLeastLoaded(runners, 1, []string{"buildkit"}, full, always); got == nil || got.ID != 3 {
		t.Fatalf("with 1&2 saturated, selection = %v, want runner 3", got)
	}

	// Everyone saturated → no runner (caller queues, "waiting for a runner").
	saturated := map[uint]int{1: 2, 2: 2, 3: 1}
	if got := pickLeastLoaded(runners, 1, []string{"buildkit"}, saturated, always); got != nil {
		t.Fatalf("all saturated: selection = %v, want none", got)
	}

	// A required label no runner has → no match.
	if got := pickLeastLoaded(runners, 1, []string{"gpu"}, loads, always); got != nil {
		t.Fatalf("unmatched label: selection = %v, want none", got)
	}
}
