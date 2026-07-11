// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPipelineDB(t *testing.T) *PipelineRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Only the run tables are needed here; PipelineDefinition's UIDModel carries a
	// Postgres-only gen_random_uuid() default that SQLite AutoMigrate rejects.
	if err := db.AutoMigrate(&models.PipelineRun{}, &models.PipelineStepRun{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewPipelineRepository(db)
}

// TestLatestRunByPipeline verifies the list-enrichment query returns the
// highest-numbered run for each pipeline and omits pipelines with no runs.
func TestLatestRunByPipeline(t *testing.T) {
	repo := newPipelineDB(t)

	// Pipeline 1 has three runs; #3 is the latest even if inserted out of order.
	for _, n := range []int{1, 3, 2} {
		status := models.PipelineRunSucceeded
		if n == 3 {
			status = models.PipelineRunFailed
		}
		if err := repo.CreateRun(&models.PipelineRun{WorkspaceID: 7, PipelineID: 1, Number: n, Status: status}); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}
	// Pipeline 2 has a single run.
	if err := repo.CreateRun(&models.PipelineRun{WorkspaceID: 7, PipelineID: 2, Number: 1, Status: models.PipelineRunRunning}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	// Pipeline 3 has no runs and must be absent from the result.

	latest, err := repo.LatestRunByPipeline([]uint{1, 2, 3})
	if err != nil {
		t.Fatalf("LatestRunByPipeline: %v", err)
	}

	if got := len(latest); got != 2 {
		t.Fatalf("expected 2 pipelines with runs, got %d", got)
	}
	if r, ok := latest[1]; !ok {
		t.Error("pipeline 1 missing")
	} else if r.Number != 3 || r.Status != models.PipelineRunFailed {
		t.Errorf("pipeline 1: want run #3/failed, got #%d/%s", r.Number, r.Status)
	}
	if r, ok := latest[2]; !ok {
		t.Error("pipeline 2 missing")
	} else if r.Number != 1 || r.Status != models.PipelineRunRunning {
		t.Errorf("pipeline 2: want run #1/running, got #%d/%s", r.Number, r.Status)
	}
	if _, ok := latest[3]; ok {
		t.Error("pipeline 3 has no runs but appeared in the result")
	}

	// Summary() must project the run faithfully for the list view.
	sum := latest[1].Summary()
	if sum.Number != 3 || sum.Status != models.PipelineRunFailed || sum.ID != latest[1].ID {
		t.Errorf("Summary mismatch: %+v", sum)
	}
}

// TestLatestRunByPipelineEmpty guards the no-IDs short-circuit.
func TestLatestRunByPipelineEmpty(t *testing.T) {
	repo := newPipelineDB(t)
	latest, err := repo.LatestRunByPipeline(nil)
	if err != nil {
		t.Fatalf("LatestRunByPipeline(nil): %v", err)
	}
	if len(latest) != 0 {
		t.Errorf("expected empty map, got %d entries", len(latest))
	}
}
