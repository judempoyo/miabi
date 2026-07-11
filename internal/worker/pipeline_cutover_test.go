// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestSubjectUser(t *testing.T) {
	if got := subjectUser(&models.PipelineRun{}); got != 0 {
		t.Errorf("no trigger user → %d, want 0", got)
	}
	u := uint(9)
	if got := subjectUser(&models.PipelineRun{TriggeredByUserID: &u}); got != 9 {
		t.Errorf("trigger user = %d, want 9", got)
	}
}

// jobInputs maps a command-only pipeline (no bound app) onto the runner context
// without touching the app/workspace repos.
func TestJobInputsNoApp(t *testing.T) {
	h := &PipelineHandler{registry: "reg.example.com"}
	run := &models.PipelineRun{ID: 1, WorkspaceID: 42, Number: 3, Commit: "abc123"}
	def := &models.PipelineDefinition{Name: "deploy"}

	in, err := h.jobInputs(run, def)
	if err != nil {
		t.Fatalf("jobInputs: %v", err)
	}
	if in.Pipeline != "deploy" || in.Registry != "reg.example.com" || in.Ref != "abc123" {
		t.Errorf("job inputs = %+v", in)
	}
	if in.AppID != nil || in.Repository != "" || in.SourceURL != "" {
		t.Errorf("no-app pipeline should have no app/repo/source: %+v", in)
	}
	if in.Run != run {
		t.Error("run not carried through")
	}
}
