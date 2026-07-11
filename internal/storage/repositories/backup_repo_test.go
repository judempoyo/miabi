// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"testing"

	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBackupDB(t *testing.T) *BackupRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Backup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewBackupRepository(db)
}

// TestBackupSetLogMeta verifies externalizing a backup run's log replaces the
// DB column with the bounded tail and records the store reference + counters,
// and that a zero ref is a no-op leaving the full log intact.
func TestBackupSetLogMeta(t *testing.T) {
	repo := newBackupDB(t)
	b := &models.Backup{WorkspaceID: 7, DatabaseID: 1, Logs: "dumping...\ndone\n"}
	if err := repo.Create(b); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.SetLogMeta(b.ID, "", "x", 1, 1, false); err != nil {
		t.Fatalf("noop: %v", err)
	}
	got, _ := repo.FindInWorkspace(7, b.ID)
	if got.Logs != "dumping...\ndone\n" || got.LogRef != "" {
		t.Fatalf("zero ref should be a no-op, got logs=%q ref=%q", got.Logs, got.LogRef)
	}

	ref := "logs/backup/ws_7/backup-1.log"
	if err := repo.SetLogMeta(b.ID, ref, "done\n", 16, 2, false); err != nil {
		t.Fatalf("SetLogMeta: %v", err)
	}
	got, _ = repo.FindInWorkspace(7, b.ID)
	if got.LogRef != ref || got.Logs != "done\n" || got.LogBytes != 16 || got.LogLines != 2 {
		t.Errorf("meta = ref=%q logs=%q bytes=%d lines=%d, want ref=%q tail bytes=16 lines=2",
			got.LogRef, got.Logs, got.LogBytes, got.LogLines, ref)
	}
}
