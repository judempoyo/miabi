// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestFailedAppStatus(t *testing.T) {
	cases := []struct {
		name       string
		hasCurrent bool
		strategy   models.DeployStrategy
		want       models.AppStatus
	}{
		{"rolling with a live previous release stays running", true, models.DeployRolling, models.AppStatusRunning},
		{"canary with a live previous release stays running", true, models.DeployCanary, models.AppStatusRunning},
		{"empty strategy defaults to keeping the app running", true, "", models.AppStatusRunning},
		{"recreate stopped the old container first — failed", true, models.DeployRecreate, models.AppStatusFailed},
		{"first-ever deploy has nothing running — failed", false, models.DeployRolling, models.AppStatusFailed},
		{"first-ever recreate — failed", false, models.DeployRecreate, models.AppStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := failedAppStatus(tc.hasCurrent, tc.strategy); got != tc.want {
				t.Errorf("failedAppStatus(%v, %q) = %q, want %q", tc.hasCurrent, tc.strategy, got, tc.want)
			}
		})
	}
}

// Rolling runs the new container beside the old one, so it cannot be used while
// a host port is published — Docker gives the second container "port is already
// allocated" and the deploy fails. The worker downgrades instead of failing.
func TestEffectiveStrategy(t *testing.T) {
	cases := []struct {
		name      string
		requested models.DeployStrategy
		ports     int
		want      models.DeployStrategy
	}{
		{"rolling with no host ports is left alone", models.DeployRolling, 0, models.DeployRolling},
		{"rolling publishing a host port becomes recreate", models.DeployRolling, 1, models.DeployRecreate},
		{"rolling publishing several still becomes recreate", models.DeployRolling, 3, models.DeployRecreate},
		// Canary publishes no host ports at all, so it never reaches the rule with
		// a non-zero count; assert it is untouched either way.
		{"canary is never downgraded", models.DeployCanary, 0, models.DeployCanary},
		{"canary is never downgraded even with ports", models.DeployCanary, 2, models.DeployCanary},
		{"recreate already stops the old container first", models.DeployRecreate, 2, models.DeployRecreate},
		{"an unset strategy is not invented", "", 2, ""},
	}
	for _, tc := range cases {
		if got := effectiveStrategy(tc.requested, tc.ports); got != tc.want {
			t.Errorf("%s: effectiveStrategy(%q, %d) = %q, want %q", tc.name, tc.requested, tc.ports, got, tc.want)
		}
	}
}
