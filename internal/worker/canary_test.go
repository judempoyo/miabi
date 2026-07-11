// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"strings"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestNextCanaryWeight(t *testing.T) {
	cases := []struct{ current, step, want int }{
		{10, 20, 30},
		{90, 20, 100}, // capped
		{100, 20, 100},
		{50, 50, 100},
	}
	for _, c := range cases {
		if got := nextCanaryWeight(c.current, c.step); got != c.want {
			t.Errorf("nextCanaryWeight(%d,%d) = %d, want %d", c.current, c.step, got, c.want)
		}
	}
}

func TestBuildHealthcheck(t *testing.T) {
	if buildHealthcheck(&models.Application{HealthcheckType: models.HealthcheckNone}) != nil {
		t.Errorf("none should produce no healthcheck")
	}
	if buildHealthcheck(&models.Application{HealthcheckType: models.HealthcheckCommand, HealthcheckCommand: "  "}) != nil {
		t.Errorf("blank command should produce no healthcheck")
	}

	http := buildHealthcheck(&models.Application{HealthcheckType: models.HealthcheckHTTP, Port: 3000, HealthcheckHTTPPath: "/healthz"})
	if http == nil || len(http.Test) != 2 || http.Test[0] != "CMD-SHELL" {
		t.Fatalf("unexpected http healthcheck: %+v", http)
	}
	if !strings.Contains(http.Test[1], "http://localhost:3000/healthz") {
		t.Errorf("http test missing url: %s", http.Test[1])
	}

	cmd := buildHealthcheck(&models.Application{HealthcheckType: models.HealthcheckCommand, HealthcheckCommand: "pg_isready"})
	if cmd == nil || cmd.Test[1] != "pg_isready" {
		t.Errorf("unexpected command healthcheck: %+v", cmd)
	}
}

func TestCanaryTuningFallbacks(t *testing.T) {
	// Zero/invalid tuning falls back to safe defaults.
	zero := &models.Application{}
	if w := canaryInitialWeight(zero); w != defaultCanaryWeight {
		t.Errorf("initial weight fallback = %d, want %d", w, defaultCanaryWeight)
	}
	if s := canaryStep(zero); s != 20 {
		t.Errorf("step fallback = %d, want 20", s)
	}
	if i := canaryInterval(zero); i != 60 {
		t.Errorf("interval fallback = %d, want 60", i)
	}

	// Configured values are honored.
	app := &models.Application{CanaryInitialWeight: 5, CanaryStepWeight: 15, CanaryStepIntervalSeconds: 30}
	if w := canaryInitialWeight(app); w != 5 {
		t.Errorf("initial weight = %d, want 5", w)
	}
	if s := canaryStep(app); s != 15 {
		t.Errorf("step = %d, want 15", s)
	}
	if i := canaryInterval(app); i != 30 {
		t.Errorf("interval = %d, want 30", i)
	}
}
