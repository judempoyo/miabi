// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package platformbackup

import "testing"

func TestVolumeArchiveName(t *testing.T) {
	cases := map[string]string{
		"mb-node-gateway-providers": "mb-node-gateway-providers",
		"mb_data":                   "mb_data",
		"weird/name:tag":            "weird-name-tag",
		"":                          "platform-volume",
	}
	for in, want := range cases {
		if got := volumeArchiveName(in); got != want {
			t.Errorf("volumeArchiveName(%q) = %q, want %q", in, got, want)
		}
	}
}
