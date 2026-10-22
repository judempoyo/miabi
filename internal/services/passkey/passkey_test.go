// Copyright (c) 2026 Jonas Kaninda.
//
// This file is part of Miabi.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package passkey

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestConfigFromWebURL(t *testing.T) {
	cfg, ok := ConfigFromWebURL("https://miabi.example.com/console", "", "Miabi", []string{" https://console.example.com/ ", ""})
	if !ok {
		t.Fatal("a web URL should enable passkeys")
	}
	if cfg.RPID != "miabi.example.com" {
		t.Errorf("RPID = %q", cfg.RPID)
	}
	if want := []string{"https://miabi.example.com", "https://console.example.com"}; !reflect.DeepEqual(cfg.Origins, want) {
		t.Errorf("Origins = %v, want %v", cfg.Origins, want)
	}
	if cfg, _ := ConfigFromWebURL("http://localhost:5173", "miabi.io", "Miabi", nil); cfg.RPID != "miabi.io" || cfg.Origins[0] != "http://localhost:5173" {
		t.Errorf("override not applied: %+v", cfg)
	}
	if _, ok := ConfigFromWebURL("", "", "Miabi", nil); ok {
		t.Error("no web URL must leave passkeys off")
	}
}

func TestName(t *testing.T) {
	if Name("  ") != "Passkey" {
		t.Error("an empty name should fall back to Passkey")
	}
	if got := Name(strings.Repeat("é", 80)); len([]rune(got)) != 64 {
		t.Errorf("name not cut to 64 runes: %d", len([]rune(got)))
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	mr := miniredis.RunT(t)
	cfg, _ := ConfigFromWebURL("https://miabi.example.com", "", "Miabi", nil)
	s, err := New(cfg, redis.NewClient(&redis.Options{Addr: mr.Addr()}), nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A ceremony answers once: a replayed response finds nothing.
func TestCeremonyIsSingleUse(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	token, opts, err := s.BeginLogin(ctx)
	if err != nil || token == "" || opts == nil {
		t.Fatalf("BeginLogin = %q, %v, %v", token, opts, err)
	}
	if opts.Response.UserVerification != "required" {
		t.Errorf("sign-in must require user verification, got %q", opts.Response.UserVerification)
	}
	if _, err := s.take(ctx, "login", token); err != nil {
		t.Fatalf("first take: %v", err)
	}
	if _, err := s.take(ctx, "login", token); !errors.Is(err, ErrCeremony) {
		t.Fatalf("second take = %v, want ErrCeremony", err)
	}
	if _, err := s.take(ctx, "reg", token); !errors.Is(err, ErrCeremony) {
		t.Fatal("a sign-in token must not answer a registration")
	}
}

func TestFinishLoginRejectsGarbage(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	token, _, err := s.BeginLogin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.FinishLogin(ctx, token, []byte(`{"id":"x"}`)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if _, err := s.FinishLogin(ctx, token, []byte(`{}`)); !errors.Is(err, ErrCeremony) {
		t.Fatalf("a used ceremony must not be retried, got %v", err)
	}
}

func TestDisabledService(t *testing.T) {
	var s *Service
	if s.Enabled() {
		t.Error("nil service should be disabled")
	}
	if _, _, err := s.BeginLogin(context.Background()); !errors.Is(err, ErrDisabled) {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
}
