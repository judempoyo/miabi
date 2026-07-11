// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package events

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestNextStoredStatus(t *testing.T) {
	run := models.AppStatusRunning
	cases := []struct {
		name       string
		action     string
		exit       string
		current    models.AppStatus
		want       models.AppStatus
		wantChange bool
	}{
		{"crash flips running to failed", "die", "1", run, models.AppStatusFailed, true},
		{"oom flips running to failed", "oom", "", run, models.AppStatusFailed, true},
		{"graceful exit no change", "die", "0", run, run, false},
		{"sigterm exit no change", "die", "143", run, run, false},
		{"start recovers failed to running", "start", "", models.AppStatusFailed, models.AppStatusRunning, true},
		{"start when already running no change", "start", "", run, run, false},
		{"crash when stopped no change", "die", "1", models.AppStatusStopped, models.AppStatusStopped, false},
		{"health event ignored", "health_status: unhealthy", "", run, run, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, change := nextStoredStatus(c.action, c.exit, c.current)
			if got != c.want || change != c.wantChange {
				t.Errorf("nextStoredStatus(%q,%q,%q) = (%q,%v), want (%q,%v)", c.action, c.exit, c.current, got, change, c.want, c.wantChange)
			}
		})
	}
}

func TestDieIsStop(t *testing.T) {
	run := models.AppStatusRunning
	stopped := models.AppStatusStopped
	cases := []struct {
		name     string
		exit     string
		status   models.AppStatus
		wantStop bool
	}{
		{"graceful exit is a stop", "0", run, true},
		{"sigterm exit is a stop", "143", run, true},
		{"crash while running is not a stop", "1", run, false},
		{"sigkill while running is not a stop", "137", run, false},
		// A user-initiated stop is a stop regardless of exit code — Git/buildpack
		// images get SIGKILLed (137) because /bin/sh doesn't forward SIGTERM.
		{"sigkill of a stopped app is a stop", "137", stopped, true},
		{"any code of a stopped app is a stop", "1", stopped, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dieIsStop(c.exit, c.status); got != c.wantStop {
				t.Errorf("dieIsStop(%q,%q) = %v, want %v", c.exit, c.status, got, c.wantStop)
			}
		})
	}
}
