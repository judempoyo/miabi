// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package database

import (
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/models"
)

func TestCmpVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"16", "17", -1},
		{"17", "16", 1},
		{"16.2", "16.4", -1},
		{"16", "16.2", -1}, // shorter prefix is smaller
		{"16.2", "16", 1},
		{"17-alpine", "17", 0}, // suffix ignored
		{"8.4", "8.4", 0},
		{"", "", 0},
	}
	for _, c := range cases {
		if got := cmpVersion(c.a, c.b); got != c.want {
			t.Errorf("cmpVersion(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestMajorOf(t *testing.T) {
	for _, c := range []struct {
		v    string
		want int
		ok   bool
	}{
		{"17-alpine", 17, true},
		{"8.4", 8, true},
		{"11", 11, true},
		{"", 0, false},
		{"latest", 0, false},
	} {
		got, ok := majorOf(c.v)
		if got != c.want || ok != c.ok {
			t.Errorf("majorOf(%q)=(%d,%v) want (%d,%v)", c.v, got, ok, c.want, c.ok)
		}
	}
}

func TestUpgradePath(t *testing.T) {
	cases := []struct {
		name     string
		engine   models.DBEngine
		from, to string
		path     string
		major    bool
		err      error
	}{
		{"pg minor in-place", models.DBEnginePostgres, "16.2", "16.4", PathInPlace, false, nil},
		{"pg major dump/restore", models.DBEnginePostgres, "16", "17", PathDumpRestore, true, nil},
		{"mysql minor in-place", models.DBEngineMySQL, "8.0", "8.0", "", false, ErrAlreadyOnVersion},
		{"mysql major dump/restore", models.DBEngineMySQL, "8.4", "9.0", PathDumpRestore, true, nil},
		{"redis major still in-place", models.DBEngineRedis, "7", "8", PathInPlace, true, nil},
		{"mongo minor in-place", models.DBEngineMongoDB, "7.0", "7.1", PathInPlace, false, nil},
		{"mongo major rejected", models.DBEngineMongoDB, "6.0", "7.0", "", true, ErrMongoMajorUpgrade},
		{"downgrade refused", models.DBEnginePostgres, "17", "16", "", false, ErrDowngrade},
		{"same version refused", models.DBEnginePostgres, "17", "17", "", false, ErrAlreadyOnVersion},
		{"empty target refused", models.DBEnginePostgres, "16", "", "", false, ErrInvalidVersion},
		{"non-numeric target refused", models.DBEnginePostgres, "16", "latest", "", false, ErrInvalidVersion},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, major, err := upgradePath(c.engine, c.from, c.to)
			if !errors.Is(err, c.err) {
				t.Fatalf("err=%v want %v", err, c.err)
			}
			if err == nil && (path != c.path || major != c.major) {
				t.Errorf("got (%q,%v) want (%q,%v)", path, major, c.path, c.major)
			}
		})
	}
}

func TestSuggestedVersions(t *testing.T) {
	got := suggestedVersions(models.DBEnginePostgres, "16")
	// Only strictly-newer majors should be suggested.
	for _, v := range got {
		if cmpVersion("16", v) >= 0 {
			t.Errorf("suggested %q is not newer than 16", v)
		}
	}
	if len(got) == 0 {
		t.Error("expected some newer postgres suggestions")
	}
}
