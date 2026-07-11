// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package siem

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/models"
)

func sampleEvents() []models.AuditLog {
	ws := uint(2)
	return []models.AuditLog{
		{ID: 10, Action: "app.deploy", TargetType: "app", TargetID: "3", IPAddress: "1.2.3.4", CreatedAt: time.Unix(1700000000, 0).UTC()},
		{ID: 11, WorkspaceID: &ws, Action: "workspace.member_role_update", TargetType: "user", TargetID: "5", CreatedAt: time.Unix(1700000100, 0).UTC()},
	}
}

func TestWebhookSinkShipsNDJSON(t *testing.T) {
	var gotBody, gotAuth, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotAuth = r.Header.Get("Authorization")
		gotType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink := newWebhookSink(srv.URL, "Bearer secret")
	if err := sink.Ship(sampleEvents()); err != nil {
		t.Fatalf("Ship: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(gotBody), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %q", len(lines), gotBody)
	}
	if !strings.Contains(lines[0], `"action":"app.deploy"`) || !strings.Contains(lines[0], `"id":10`) {
		t.Fatalf("first line wrong: %s", lines[0])
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if gotType != "application/x-ndjson" {
		t.Fatalf("content-type = %q", gotType)
	}
}

func TestWebhookSinkNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := newWebhookSink(srv.URL, "").Ship(sampleEvents()); err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

func TestSyslogEndpointParsing(t *testing.T) {
	if _, err := newSyslogSink(&models.SIEMConfig{Endpoint: ""}); err == nil {
		t.Fatal("empty endpoint should error")
	}
	if _, err := newSyslogSink(&models.SIEMConfig{Endpoint: "ftp://host:1"}); err == nil {
		t.Fatal("non tcp/udp scheme should error")
	}
	s, err := newSyslogSink(&models.SIEMConfig{Endpoint: "tcp://host:514", Format: models.SIEMFormatCEF})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.network != "tcp" || s.addr != "host:514" || !s.cef {
		t.Fatalf("parsed wrong: %+v", s)
	}
	// Default scheme is udp.
	d, _ := newSyslogSink(&models.SIEMConfig{Endpoint: "host:514"})
	if d.network != "udp" {
		t.Fatalf("default network = %q, want udp", d.network)
	}
}

func TestSyslogFormat(t *testing.T) {
	e := sampleEvents()[0]
	jsonSink := &syslogSink{cef: false}
	msg, _ := jsonSink.format(&e)
	if !strings.Contains(msg, `"action":"app.deploy"`) {
		t.Fatalf("json format wrong: %s", msg)
	}
	cefSink := &syslogSink{cef: true}
	cef, _ := cefSink.format(&e)
	if !strings.HasPrefix(cef, "CEF:0|Miabi|Miabi|1|app.deploy|") {
		t.Fatalf("cef format wrong: %s", cef)
	}
}
