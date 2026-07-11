// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package application

import "testing"

func TestDeriveLiveStatus(t *testing.T) {
	cases := []struct {
		name       string
		state      string
		health     string
		restarting bool
		stopped    bool
		want       string
	}{
		{"running healthy", "running", "healthy", false, false, "running"},
		{"running no healthcheck", "running", "", false, false, "running"},
		{"running unhealthy", "running", "unhealthy", false, false, "unhealthy"},
		{"running starting", "running", "starting", false, false, "starting"},
		{"running but restarting flag", "running", "", true, false, "restarting"},
		{"restarting state", "restarting", "", false, false, "restarting"},
		{"exited unexpectedly", "exited", "", false, false, "exited"},
		{"exited after stop", "exited", "", false, true, "stopped"},
		{"dead crash", "dead", "", false, false, "exited"},
		{"paused", "paused", "", false, false, "paused"},
		{"created", "created", "", false, false, "created"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := deriveLiveStatus(c.state, c.health, c.restarting, c.stopped); got != c.want {
				t.Errorf("deriveLiveStatus(%q,%q,%v,%v) = %q, want %q", c.state, c.health, c.restarting, c.stopped, got, c.want)
			}
		})
	}
}
