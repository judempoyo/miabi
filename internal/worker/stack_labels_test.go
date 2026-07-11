// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"testing"

	"github.com/miabi-io/miabi/internal/docker"
	"github.com/miabi-io/miabi/internal/models"
)

func TestStackLabels_NoStack(t *testing.T) {
	app := &models.Application{ID: 1, Name: "web", WorkspaceID: 4}
	got := stackLabels(app, map[string]string{docker.LabelApp: "1"})
	if _, ok := got["com.docker.compose.project"]; ok {
		t.Error("ungrouped app should not get a compose project label")
	}
	if got[docker.LabelApp] != "1" {
		t.Errorf("base labels must be preserved, got %v", got)
	}
	// Every app container is stamped with its owning workspace, stack or not.
	if got[docker.LabelWorkspace] != "4" {
		t.Errorf("workspace label = %q, want 4", got[docker.LabelWorkspace])
	}
}

func TestStackLabels_WithStack(t *testing.T) {
	app := &models.Application{
		ID:          7,
		Name:        "web",
		WorkspaceID: 1,
		Stack:       &models.Stack{ID: 3, DockerName: "ws1-blog"},
	}
	got := stackLabels(app, map[string]string{docker.LabelApp: "7"})

	want := map[string]string{
		"com.docker.compose.project": "ws1-blog", // groups in `docker compose ls` / Desktop
		"com.docker.compose.service": "web",
		docker.LabelStack:            "3",
		docker.LabelApp:              "7", // base label preserved
		docker.LabelWorkspace:        "1",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("label %s = %q, want %q", k, got[k], v)
		}
	}
}
