// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package logstore

import (
	"io"
	"strings"
	"testing"
	"time"
)

func newStore(t *testing.T, cfg Config) *Store {
	t.Helper()
	be, err := NewFSBackend(t.TempDir())
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	return New(be, cfg)
}

func read(t *testing.T, s *Store, ref string) string {
	t.Helper()
	rc, err := s.Open(ref)
	if err != nil {
		t.Fatalf("open %s: %v", ref, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read %s: %v", ref, err)
	}
	return string(data)
}

func TestExternalizeRoundTrip(t *testing.T) {
	for _, compress := range []bool{true, false} {
		s := newStore(t, Config{Compress: compress, TailBytes: 1 << 10})
		ref := DeploymentRef(42, 128, 57)
		content := "line one\nline two\nline three\n"
		res, err := s.Externalize(ref, content)
		if err != nil {
			t.Fatalf("externalize: %v", err)
		}
		if res.Ref != ref {
			t.Errorf("ref = %q, want %q", res.Ref, ref)
		}
		if res.Bytes != int64(len(content)) {
			t.Errorf("bytes = %d, want %d", res.Bytes, len(content))
		}
		if res.Lines != 3 {
			t.Errorf("lines = %d, want 3", res.Lines)
		}
		if got := read(t, s, ref); got != content {
			t.Errorf("stored = %q, want %q", got, content)
		}
	}
}

func TestDisabledStoreIsNoOpButTails(t *testing.T) {
	var s *Store // nil = disabled
	if s.Enabled() {
		t.Fatal("nil store should be disabled")
	}
	if s.Backend() != "off" {
		t.Errorf("backend = %q, want off", s.Backend())
	}
	res, err := s.Externalize("logs/x.log", "a\nb\nc\n")
	if err != nil {
		t.Fatalf("externalize: %v", err)
	}
	if res.Ref != "" {
		t.Errorf("ref = %q, want empty on disabled store", res.Ref)
	}
	if res.Tail != "a\nb\nc\n" {
		t.Errorf("tail = %q, want full input", res.Tail)
	}
	if err := s.Delete("logs/x.log"); err != nil {
		t.Errorf("delete on disabled store: %v", err)
	}
	if n, err := s.Sweep(time.Now()); err != nil || n != 0 {
		t.Errorf("sweep on disabled store = %d, %v", n, err)
	}
}

func TestMaxBytesMiddleTruncation(t *testing.T) {
	s := newStore(t, Config{Compress: true, MaxBytes: 400, TailBytes: 128})
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("this is a fairly long log line number\n")
	}
	res, err := s.Externalize(JobRef(1, 9), b.String())
	if err != nil {
		t.Fatalf("externalize: %v", err)
	}
	if !res.Truncated {
		t.Fatal("expected truncated")
	}
	stored := read(t, s, JobRef(1, 9))
	if int64(len(stored)) > 400+128 {
		t.Errorf("stored len %d exceeds cap+marker", len(stored))
	}
	if !strings.Contains(stored, "truncated") {
		t.Errorf("missing truncation marker in %q", stored)
	}
}

func TestTailBoundsToLineStart(t *testing.T) {
	s := newStore(t, Config{Compress: true, TailBytes: 12})
	ref := JobRef(2, 3)
	_, err := s.Externalize(ref, "aaaa\nbbbb\ncccc\ndddd\n")
	if err != nil {
		t.Fatalf("externalize: %v", err)
	}
	tail, err := s.Tail(ref, 12)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	// Tail must begin at a clean line boundary, so every returned line is whole.
	for _, line := range SplitLines(tail) {
		if len(line) != 4 {
			t.Errorf("tail line %q not whole", line)
		}
	}
}

func TestTailMissingObjectFallsBack(t *testing.T) {
	s := newStore(t, Config{Compress: true})
	tail, err := s.Tail(DeploymentRef(9, 9, 9), 1024)
	if err != nil {
		t.Fatalf("tail on missing: %v", err)
	}
	if tail != "" {
		t.Errorf("tail = %q, want empty for missing object", tail)
	}
}

func TestDeleteAndSweep(t *testing.T) {
	s := newStore(t, Config{Compress: true, RetentionDays: 30})
	ref := DeploymentRef(1, 2, 3)
	if _, err := s.Externalize(ref, "x\n"); err != nil {
		t.Fatalf("externalize: %v", err)
	}
	// Nothing is old enough yet.
	if n, err := s.Sweep(time.Now()); err != nil || n != 0 {
		t.Errorf("sweep = %d, %v; want 0", n, err)
	}
	// A far-future "now" makes the object older than retention.
	if n, err := s.Sweep(time.Now().AddDate(0, 0, 60)); err != nil || n != 1 {
		t.Errorf("sweep = %d, %v; want 1", n, err)
	}
	if _, err := s.Open(ref); err != ErrNotFound {
		t.Errorf("open after sweep = %v, want ErrNotFound", err)
	}
	// Delete of an absent object is not an error.
	if err := s.Delete(ref); err != nil {
		t.Errorf("delete absent: %v", err)
	}
}

func TestRefSchemes(t *testing.T) {
	cases := map[string]string{
		DeploymentRef(42, 128, 57): "deployment/ws_42/app-128/dep-57.log",
		PipelineStepRef(42, 9, 2):  "pipeline/ws_42/run-9/step-2.log",
		PipelineRunRef(42, 9):      "pipeline/ws_42/run-9/run.log",
		JobRef(42, 3):              "job/ws_42/job-3.log",
		BackupRef(7, 1):            "backup/ws_7/backup-1.log",
		VolumeBackupRef(7, 4):      "volume-backup/ws_7/vbackup-4.log",
		PlatformBackupRef(5):       "platform/pbackup-5.log",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("ref = %q, want %q", got, want)
		}
	}
}

func TestPathTraversalRejected(t *testing.T) {
	be, err := NewFSBackend(t.TempDir())
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	if _, err := be.Create("../escape.log"); err == nil {
		t.Error("expected traversal ref to be rejected")
	}
}
