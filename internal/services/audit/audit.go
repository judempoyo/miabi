// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package audit writes append-only audit-log entries for mutating actions.
package audit

import (
	"github.com/jkaninda/logger"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/eventbus"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

// GlobalTopic is the eventbus topic on which every audit entry is published so
// the platform-admin Events stream can fan them out over SSE.
const GlobalTopic = "audit:global"

type Logger struct {
	repo *repositories.AuditLogRepository
	bus  *eventbus.Bus
}

// NewLogger builds an audit logger. The bus may be nil (audit still persists).
func NewLogger(repo *repositories.AuditLogRepository, bus *eventbus.Bus) *Logger {
	return &Logger{repo: repo, bus: bus}
}

// Entry describes a single audit event.
type Entry struct {
	ActorID     *uint
	WorkspaceID *uint
	Action      string
	TargetType  string
	TargetID    string
	IP          string
	Metadata    map[string]any
}

// Record persists an audit entry. Failures are logged but never block the
// caller — auditing must not break the request path.
func (l *Logger) Record(e Entry) {
	if l == nil || l.repo == nil {
		return
	}
	entry := &models.AuditLog{
		ActorID:     e.ActorID,
		WorkspaceID: e.WorkspaceID,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetID:    e.TargetID,
		IPAddress:   e.IP,
		Metadata:    e.Metadata,
	}
	if err := l.repo.Create(entry); err != nil {
		logger.Error("failed to write audit log", "action", e.Action, "error", err)
		return
	}
	if l.bus != nil {
		l.bus.Publish(GlobalTopic, eventbus.Event{Type: e.Action, Data: entry})
	}
}
