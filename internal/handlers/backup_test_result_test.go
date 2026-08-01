// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"strings"
	"testing"

	"github.com/miabi-io/miabi/internal/services/backupsettings"
)

func TestBackupTestResultReportsWhatWasDone(t *testing.T) {
	res := backupTestResult([]backupsettings.PrefixCheck{
		{Prefix: "backups/databases", Removed: true},
		{Prefix: "bundles", Removed: true},
	})
	if !res.OK {
		t.Fatalf("a clean probe was reported as a failure: %+v", res)
	}
	for _, want := range []string{"Wrote, read back and removed", "backups/databases/", "bundles/"} {
		if !strings.Contains(res.Message, want) {
			t.Errorf("message %q does not mention %q", res.Message, want)
		}
	}
}

func TestBackupTestResultFailsOnAnyRefusedPrefix(t *testing.T) {
	res := backupTestResult([]backupsettings.PrefixCheck{
		{Prefix: "backups/databases", Removed: true},
		{Prefix: "bundles", Error: "could not write bundles/x: these credentials are not allowed to write here"},
	})
	if res.OK {
		t.Fatal("a refused prefix was reported as a pass")
	}
	if !strings.Contains(res.Message, "bundles/") || !strings.Contains(res.Message, "not allowed to write") {
		t.Fatalf("message does not say which prefix failed or why: %q", res.Message)
	}
}

func TestBackupTestResultCallsOutMissingDelete(t *testing.T) {
	res := backupTestResult([]backupsettings.PrefixCheck{{Prefix: "bundles", Removed: false}})
	if !res.OK {
		t.Fatal("a target that can be written and read was reported unusable")
	}
	if !strings.Contains(res.Message, "could not delete") || !strings.Contains(res.Message, "deleting them will not") {
		t.Fatalf("message hides the missing delete permission: %q", res.Message)
	}
}

// The bucket root is a real answer, and "/" is not a readable name for it.
func TestBucketRootIsNamed(t *testing.T) {
	res := backupTestResult([]backupsettings.PrefixCheck{{Prefix: "", Removed: true}})
	if !strings.Contains(res.Message, "the bucket root") {
		t.Fatalf("the root prefix was not named: %q", res.Message)
	}
}
