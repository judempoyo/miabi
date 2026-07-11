// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package dotenv

import "testing"

func TestParse(t *testing.T) {
	const content = `
# a comment
FOO=bar
export BAZ=qux
QUOTED="hello world"
SINGLE='single'
EMPTY=
URL=postgres://u:p@host:5432/db?sslmode=disable

  # indented comment
SPACED = value
NOEQUALS
=novalue
`
	got := Parse(content)
	want := []KeyValue{
		{"FOO", "bar"},
		{"BAZ", "qux"},
		{"QUOTED", "hello world"},
		{"SINGLE", "single"},
		{"EMPTY", ""},
		{"URL", "postgres://u:p@host:5432/db?sslmode=disable"},
		{"SPACED", "value"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d pairs, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("pair %d = %+v, want %+v", i, got[i], w)
		}
	}
}
