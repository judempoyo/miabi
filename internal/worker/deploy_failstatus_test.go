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
