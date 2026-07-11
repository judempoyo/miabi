// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package monitoring

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestHealthOf(t *testing.T) {
	cases := []struct {
		name       string
		status     models.AppStatus
		hasCurrent bool
		want       string
	}{
		{"running is healthy", models.AppStatusRunning, true, "healthy"},
		{"failed is unhealthy", models.AppStatusFailed, true, "unhealthy"},
		{"deploying over a live release stays healthy", models.AppStatusDeploying, true, "healthy"},
		{"first-ever deploy is unknown (nothing running)", models.AppStatusDeploying, false, "unknown"},
		{"stopped is unknown", models.AppStatusStopped, true, "unknown"},
		{"created is unknown", models.AppStatusCreated, false, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := healthOf(tc.status, tc.hasCurrent); got != tc.want {
				t.Errorf("healthOf(%q, %v) = %q, want %q", tc.status, tc.hasCurrent, got, tc.want)
			}
		})
	}
}
