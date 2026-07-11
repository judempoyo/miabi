// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package proxy

import (
	"context"
	"sync"
)

// Memory is an in-process Manager used when no real gateway is configured
// (dev/tests). It records the desired routes and middlewares per workspace.
type Memory struct {
	mu          sync.RWMutex
	routes      map[uint]RenderedRoute      // by route id, across workspaces
	middlewares map[uint]RenderedMiddleware // by middleware id, across workspaces
	registry    RegistryProxy               // last registry proxy config
}

func NewMemory() *Memory {
	return &Memory{
		routes:      make(map[uint]RenderedRoute),
		middlewares: make(map[uint]RenderedMiddleware),
	}
}

// SyncWorkspace replaces the recorded state for a workspace: it drops the
// workspace's previous routes/middlewares and stores the supplied set.
func (m *Memory) SyncWorkspace(_ context.Context, workspaceID uint, routes []RenderedRoute, mws []RenderedMiddleware) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, r := range m.routes {
		if r.WorkspaceID == workspaceID {
			delete(m.routes, id)
		}
	}
	for id, mw := range m.middlewares {
		if mw.WorkspaceID == workspaceID {
			delete(m.middlewares, id)
		}
	}
	for _, r := range routes {
		m.routes[r.ID] = r
	}
	for _, mw := range mws {
		m.middlewares[mw.ID] = mw
	}
	return nil
}

// SyncRegistry records the registry proxy config (no-op store for the in-memory
// manager; the Goma manager writes the actual gateway file).
func (m *Memory) SyncRegistry(_ context.Context, cfg RegistryProxy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = cfg
	return nil
}

// GetRoute returns the recorded route (used in tests).
func (m *Memory) GetRoute(id uint) (RenderedRoute, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.routes[id]
	return r, ok
}
