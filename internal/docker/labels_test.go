// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package docker

import "testing"

func TestLabelValue(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		key    string
		want   string
		wantOK bool
	}{
		{"present", map[string]string{LabelApp: "42"}, LabelApp, "42", true},
		{"missing", map[string]string{"other": "x"}, LabelApp, "", false},
		{"nil map", nil, LabelApp, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := LabelValue(c.labels, c.key)
			if got != c.want || ok != c.wantOK {
				t.Errorf("LabelValue(%v,%q) = (%q,%v), want (%q,%v)", c.labels, c.key, got, ok, c.want, c.wantOK)
			}
		})
	}
}

func TestIsManaged(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		want   bool
	}{
		{"platform label", map[string]string{LabelApp: "1"}, true},
		{"managed flag", map[string]string{LabelManaged: "true"}, true},
		{"unmanaged", map[string]string{"com.example": "x"}, false},
		{"empty", map[string]string{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsManaged(c.labels); got != c.want {
				t.Errorf("IsManaged(%v) = %v, want %v", c.labels, got, c.want)
			}
		})
	}
}

func TestWorkspaceID(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		want   uint
		wantOK bool
	}{
		{"present", map[string]string{LabelWorkspace: "5"}, 5, true},
		{"none", map[string]string{LabelApp: "1"}, 0, false},
		{"zero invalid", map[string]string{LabelWorkspace: "0"}, 0, false},
		{"non-numeric invalid", map[string]string{LabelWorkspace: "abc"}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := WorkspaceID(c.labels)
			if got != c.want || ok != c.wantOK {
				t.Errorf("WorkspaceID(%v) = (%d,%v), want (%d,%v)", c.labels, got, ok, c.want, c.wantOK)
			}
		})
	}
}

func TestIsPlatformInfra(t *testing.T) {
	if !IsPlatformInfra(map[string]string{LabelRole: "node-gateway"}) {
		t.Error("role label should be infra")
	}
	if IsPlatformInfra(map[string]string{LabelApp: "1"}) {
		t.Error("app container is not infra")
	}
}
