// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package nodes

import (
	"context"
	"errors"
	"testing"

	"github.com/miabi-io/miabi/internal/docker"
)

// stubClient is a no-op docker.Client used to populate the registry in tests.
type stubClient struct{ docker.Client }

func (s *stubClient) Close() error { return nil }

// pingClient is a docker.Client whose Ping returns a fixed result, for probing.
type pingClient struct {
	docker.Client
	err error
}

func (p *pingClient) Ping(context.Context) error { return p.err }
func (p *pingClient) Close() error               { return nil }

func TestClientsResolution(t *testing.T) {
	local := &stubClient{}
	c := NewClients(1, local)

	// Local id and the zero id both resolve to the local client.
	for _, id := range []uint{1, 0} {
		got, err := c.For(id)
		if err != nil || got != docker.Client(local) {
			t.Errorf("For(%d) = (%v, %v), want local", id, got, err)
		}
		if !c.IsLocal(id) || !c.Connected(id) {
			t.Errorf("id %d should be local+connected", id)
		}
	}

	// Unknown remote node is offline.
	if _, err := c.For(2); !errors.Is(err, ErrNodeOffline) {
		t.Errorf("For(2) err = %v, want ErrNodeOffline", err)
	}
	if c.Connected(2) {
		t.Error("node 2 should not be connected")
	}

	// Register, resolve, then remove a remote node.
	remote := &stubClient{}
	c.SetRemote(2, remote)
	if got, err := c.For(2); err != nil || got != docker.Client(remote) {
		t.Errorf("For(2) after SetRemote = (%v, %v), want remote", got, err)
	}
	if !c.Connected(2) {
		t.Error("node 2 should be connected after SetRemote")
	}
	c.RemoveRemote(2)
	if _, err := c.For(2); !errors.Is(err, ErrNodeOffline) {
		t.Errorf("For(2) after RemoveRemote err = %v, want ErrNodeOffline", err)
	}
}

func TestClientsRemoteIDs(t *testing.T) {
	c := NewClients(1, &stubClient{})
	if len(c.RemoteIDs()) != 0 {
		t.Fatal("no remotes yet")
	}
	c.SetRemote(2, &stubClient{})
	c.SetRemote(3, &stubClient{})
	ids := c.RemoteIDs()
	if len(ids) != 2 {
		t.Fatalf("RemoteIDs = %v, want 2", ids)
	}
	for _, id := range ids {
		if id == 1 {
			t.Error("local id must not appear in RemoteIDs")
		}
	}
	c.RemoveRemote(2)
	if len(c.RemoteIDs()) != 1 {
		t.Errorf("after RemoveRemote, RemoteIDs = %v, want 1", c.RemoteIDs())
	}
}

func TestManagerProbe(t *testing.T) {
	c := NewClients(1, &stubClient{})
	c.SetRemote(2, &pingClient{err: nil})                       // healthy
	c.SetRemote(3, &pingClient{err: errors.New("tunnel dead")}) // dead
	m := NewManager(c, nil)                                     // node service unused by probe

	if err := m.probe(context.Background(), 2); err != nil {
		t.Errorf("healthy probe = %v, want nil", err)
	}
	if err := m.probe(context.Background(), 3); err == nil {
		t.Error("dead node probe should fail")
	}
	if err := m.probe(context.Background(), 99); !errors.Is(err, ErrNodeOffline) {
		t.Errorf("offline node probe err = %v, want ErrNodeOffline", err)
	}
}
