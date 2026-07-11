// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package portbinding

import (
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/docker"
	"github.com/miabi-io/miabi/internal/models"
)

func TestAutoApprove(t *testing.T) {
	// Free host port → approved, reviewer + note set.
	b := &models.PortBinding{Status: models.PortBindingPending, HostPort: 30080}
	if err := autoApprove(b, 7, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Status != models.PortBindingApproved {
		t.Errorf("status = %q, want approved", b.Status)
	}
	if b.ReviewedBy == nil || *b.ReviewedBy != 7 {
		t.Errorf("reviewer not set: %+v", b.ReviewedBy)
	}
	if b.ReviewNote == "" {
		t.Errorf("expected an auto-approve note")
	}

	// Host port already taken → ErrHostPortTaken, status unchanged.
	taken := &models.PortBinding{Status: models.PortBindingPending, HostPort: 30080}
	if err := autoApprove(taken, 7, true); !errors.Is(err, ErrHostPortTaken) {
		t.Errorf("expected ErrHostPortTaken, got %v", err)
	}
	if taken.Status != models.PortBindingPending {
		t.Errorf("status should stay pending on conflict, got %q", taken.Status)
	}
}

func TestPublishedPorts(t *testing.T) {
	conts := []docker.Container{
		{Names: []string{"/web"}, Ports: []docker.Port{
			{PublicPort: 8080, PrivatePort: 80, Protocol: "tcp"},
			{PrivatePort: 9000, Protocol: "tcp"}, // not published → ignored
		}},
		{Names: []string{"db"}, Ports: []docker.Port{
			{PublicPort: 5432, PrivatePort: 5432, Protocol: "tcp"},
		}},
	}
	got := publishedPorts(conts)
	if got[portKey(8080, "tcp")] != "web" {
		t.Errorf("8080/tcp owner = %q, want web", got[portKey(8080, "tcp")])
	}
	if got[portKey(5432, "tcp")] != "db" {
		t.Errorf("5432/tcp owner = %q, want db", got[portKey(5432, "tcp")])
	}
	if _, ok := got[portKey(9000, "tcp")]; ok {
		t.Errorf("unpublished port 9000 should not appear")
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2: %v", len(got), got)
	}
}

func TestPortKeyNormalizesProto(t *testing.T) {
	if portKey(53, "udp") != "53/udp" {
		t.Errorf("got %q", portKey(53, "udp"))
	}
	// Unknown/empty protocol normalizes to tcp.
	if portKey(80, "") != "80/tcp" {
		t.Errorf("got %q", portKey(80, ""))
	}
}

func TestContainerNameFallback(t *testing.T) {
	if n := containerName(docker.Container{Names: []string{"/api"}}); n != "api" {
		t.Errorf("name = %q, want api", n)
	}
	if n := containerName(docker.Container{ID: "abcdef0123456789"}); n != "abcdef012345" {
		t.Errorf("name = %q, want short id", n)
	}
}
