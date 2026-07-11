// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// AuditLog is an append-only record of a mutating action.
type AuditLog struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ActorID     *uint          `json:"actor_id" gorm:"index"`
	WorkspaceID *uint          `json:"workspace_id" gorm:"index"`
	Action      string         `json:"action" gorm:"index;not null"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	IPAddress   string         `json:"ip_address"`
	Metadata    map[string]any `json:"metadata,omitempty" gorm:"serializer:json"`
	CreatedAt   time.Time      `json:"created_at"`
}
