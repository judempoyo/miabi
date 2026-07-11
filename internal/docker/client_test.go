// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package docker

import (
	"errors"
	"testing"
)

func TestLineWriterSplitsLines(t *testing.T) {
	var got []string
	w := &lineWriter{stream: "stdout", sink: func(l LogLine) error {
		got = append(got, l.Text)
		return nil
	}}
	// Write in chunks that split a line across calls.
	_, _ = w.Write([]byte("hello\nwor"))
	_, _ = w.Write([]byte("ld\nthird\n"))

	want := []string{"hello", "world", "third"}
	if len(got) != len(want) {
		t.Fatalf("got %d lines %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLineWriterStopsOnSinkError(t *testing.T) {
	w := &lineWriter{stream: "stdout", sink: func(l LogLine) error {
		return errors.New("client gone")
	}}
	_, err := w.Write([]byte("a\n"))
	if !errors.Is(err, errSinkStop) {
		t.Fatalf("expected errSinkStop, got %v", err)
	}
}

func TestSanitizeTail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "100"},
		{"all", "5000"},
		{"garbage", "100"},
		{"0", "100"},
		{"-5", "100"},
		{"50", "50"},
		{"5000", "5000"},
		{"999999", "5000"},
	}
	for _, c := range cases {
		if got := sanitizeTail(c.in); got != c.want {
			t.Errorf("sanitizeTail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
