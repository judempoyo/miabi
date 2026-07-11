// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package eventbus is a lightweight in-process publish/subscribe bus used to
// fan out live events (e.g. deploy logs) to SSE subscribers. Single-node only;
// a separate worker process would require a Redis-backed bus instead.
package eventbus

import "sync"

// Event is a single message on a topic.
type Event struct {
	Type string `json:"type"` // e.g. "log", "status"
	Data any    `json:"data"`
}

// Bus is a topic-based fan-out broker.
type Bus struct {
	mu   sync.RWMutex
	subs map[string]map[int]chan Event
	next int
}

func New() *Bus {
	return &Bus{subs: make(map[string]map[int]chan Event)}
}

// Subscribe returns a channel of events for a topic and an unsubscribe func.
func (b *Bus) Subscribe(topic string) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[int]chan Event)
	}
	id := b.next
	b.next++
	ch := make(chan Event, 64)
	b.subs[topic][id] = ch

	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if m, ok := b.subs[topic]; ok {
			if c, ok := m[id]; ok {
				close(c)
				delete(m, id)
			}
			if len(m) == 0 {
				delete(b.subs, topic)
			}
		}
	}
}

// Publish delivers an event to all current subscribers of a topic. Slow
// subscribers whose buffer is full drop the event rather than block producers.
func (b *Bus) Publish(topic string, e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[topic] {
		select {
		case ch <- e:
		default:
		}
	}
}
