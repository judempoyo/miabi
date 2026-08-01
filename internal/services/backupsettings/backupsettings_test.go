// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package backupsettings

import (
	"reflect"
	"testing"
)

// The three prefixes are each tested, but only once: bucket policies are
// commonly scoped by prefix, so testing one proves nothing about the others —
// while testing the same path three times is noise in the operator's result.
func TestDistinctPrefixes(t *testing.T) {
	cases := []struct {
		name string
		in   SaveInput
		want []string
	}{
		{
			name: "three distinct paths are each tested",
			in: SaveInput{
				DatabaseBackupPath: "backups/databases",
				VolumeBackupPath:   "backups/volumes",
				BundlePath:         "bundles",
			},
			want: []string{"backups/databases", "backups/volumes", "bundles"},
		},
		{
			name: "a shared path is tested once",
			in: SaveInput{
				DatabaseBackupPath: "backups",
				VolumeBackupPath:   "backups",
				BundlePath:         "backups",
			},
			want: []string{"backups"},
		},
		{
			name: "unset paths collapse to the bucket root",
			in:   SaveInput{},
			want: []string{""},
		},
		{
			name: "slashes and spaces do not create a second prefix",
			in: SaveInput{
				DatabaseBackupPath: "backups/",
				VolumeBackupPath:   " /backups ",
				BundlePath:         "bundles",
			},
			want: []string{"backups", "bundles"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := distinctPrefixes(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("distinctPrefixes() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The client and the *-bkup helpers must sign for the same region, or a mismatch
// surfaces as a credential problem rather than a region one.
func TestDefaultRegionMatchesTheHelpers(t *testing.T) {
	if got := defaultRegion(""); got != "us-east-1" {
		t.Fatalf("defaultRegion(%q) = %q, want us-east-1", "", got)
	}
	if got := defaultRegion("eu-central-1"); got != "eu-central-1" {
		t.Fatalf("defaultRegion rewrote a configured region: %q", got)
	}
}
