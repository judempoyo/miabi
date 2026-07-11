// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"testing"
	"time"
)

func TestMemLimiterAllowsUpToLimitThenBlocks(t *testing.T) {
	m := newMemLimiter()
	const limit = 3
	for i := 1; i <= limit; i++ {
		if !m.allow("1.2.3.4:/login", limit, time.Minute) {
			t.Fatalf("request %d within limit should be allowed", i)
		}
	}
	if m.allow("1.2.3.4:/login", limit, time.Minute) {
		t.Fatal("request over the limit should be blocked")
	}
}

func TestMemLimiterKeysAreIndependent(t *testing.T) {
	m := newMemLimiter()
	if !m.allow("a:/login", 1, time.Minute) {
		t.Fatal("first hit on key a should be allowed")
	}
	if m.allow("a:/login", 1, time.Minute) {
		t.Fatal("second hit on key a should be blocked")
	}
	if !m.allow("b:/login", 1, time.Minute) {
		t.Fatal("a different key must have its own window")
	}
}

func TestMemLimiterWindowResets(t *testing.T) {
	m := newMemLimiter()
	if !m.allow("k", 1, 10*time.Millisecond) {
		t.Fatal("first hit allowed")
	}
	if m.allow("k", 1, 10*time.Millisecond) {
		t.Fatal("second hit blocked within window")
	}
	time.Sleep(15 * time.Millisecond)
	if !m.allow("k", 1, 10*time.Millisecond) {
		t.Fatal("after the window elapses the limit should reset")
	}
}
