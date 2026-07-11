// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/miabi-io/miabi/internal/docker"
)

// foreignWorkspaceContainer gates whether a platform admin may read a
// container's logs: true (foreign) means blocked.
func TestForeignWorkspaceContainer(t *testing.T) {
	member := map[uint]bool{1: true, 2: true}
	cases := []struct {
		name        string
		labels      map[string]string
		wantForeign bool
	}{
		{"no labels (raw container)", map[string]string{}, false},
		{"non-platform labels", map[string]string{"com.example": "x"}, false},
		{"infra gateway never foreign", map[string]string{docker.LabelRole: "node-gateway", docker.LabelWorkspace: "999"}, false},
		{"own workspace app", map[string]string{docker.LabelApp: "10", docker.LabelWorkspace: "1"}, false},
		{"other workspace app is foreign", map[string]string{docker.LabelApp: "10", docker.LabelWorkspace: "3"}, true},
		{"other workspace database is foreign", map[string]string{docker.LabelDatabase: "4", docker.LabelWorkspace: "3"}, true},
		{"app with no workspace label not foreign", map[string]string{docker.LabelApp: "10"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := foreignWorkspaceContainer(c.labels, member); got != c.wantForeign {
				t.Errorf("foreignWorkspaceContainer(%v) = %v, want %v", c.labels, got, c.wantForeign)
			}
		})
	}
}
