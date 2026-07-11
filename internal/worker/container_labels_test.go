// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"testing"

	"github.com/miabi-io/miabi/internal/docker"
	"github.com/miabi-io/miabi/internal/models"
)

func TestContainerLabels_SystemWinsOverUser(t *testing.T) {
	app := &models.Application{
		ID:          42,
		WorkspaceID: 7,
		ContainerLabels: map[string]string{
			"traefik.enable":              "true",
			"traefik.http.routers.x.rule": "Host(`x`)",
			// Spoof attempts — must never override the platform's system labels.
			docker.LabelApp:              "999",
			docker.LabelWorkspace:        "13",
			"com.docker.compose.project": "evil",
		},
	}

	got := containerLabels(app, 100)

	// User labels are present.
	if got["traefik.enable"] != "true" {
		t.Errorf("user label dropped: %v", got)
	}
	// System labels win with the real values, not the spoofed ones.
	if got[docker.LabelApp] != "42" {
		t.Errorf("io.miabi.app should be 42 (system), got %q", got[docker.LabelApp])
	}
	if got[docker.LabelWorkspace] != "7" {
		t.Errorf("io.miabi.workspace should be 7 (system), got %q", got[docker.LabelWorkspace])
	}
	if got[docker.LabelDeployment] != "100" {
		t.Errorf("io.miabi.deployment should be 100, got %q", got[docker.LabelDeployment])
	}
	// The compose spoof was stripped (no stack set → key absent).
	if v, ok := got["com.docker.compose.project"]; ok {
		t.Errorf("compose project spoof should be stripped, got %q", v)
	}
}

func TestContainerLabels_NilUserLabels(t *testing.T) {
	app := &models.Application{ID: 1, WorkspaceID: 2}
	got := containerLabels(app, 3)
	if got[docker.LabelApp] != "1" || got[docker.LabelWorkspace] != "2" || got[docker.LabelDeployment] != "3" {
		t.Errorf("system labels missing with nil user labels: %v", got)
	}
}
