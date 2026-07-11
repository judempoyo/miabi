// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// Environment is a promotion stage within a workspace (dev → staging → prod). A
// Release is promoted by updating the target environment's desired state — a Git
// commit (GitOps) or an API write — so promotion is auditable and reconciled.
type Environment struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	WorkspaceID uint `json:"workspace_id" gorm:"index:idx_env_ws_name,unique;not null"`
	// Name is the unique slug handle scoped to the workspace (e.g. "prod").
	Name string `json:"name" gorm:"index:idx_env_ws_name,unique;not null"`
	// DisplayName is the free-text label shown in the UI; not unique.
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	// Rank orders environments for promotion flow (lower promotes into higher).
	Rank int `json:"rank" gorm:"not null;default:0"`
	// RequiredApprovals gates promotion into this environment (0 = no gate).
	RequiredApprovals int `json:"required_approvals" gorm:"not null;default:0"`
	// GitSourceID optionally binds the environment to a GitOps source/folder.
	GitSourceID *uint `json:"git_source_id,omitempty" gorm:"index"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReleaseApproval records one approver's decision on promoting a release into an
// environment. A promotion needs Environment.RequiredApprovals approvals.
type ReleaseApproval struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	WorkspaceID   uint      `json:"workspace_id" gorm:"index;not null"`
	ReleaseID     uint      `json:"release_id" gorm:"index;not null"`
	EnvironmentID *uint     `json:"environment_id,omitempty" gorm:"index"`
	ApproverID    uint      `json:"approver_id" gorm:"not null"`
	Approved      bool      `json:"approved" gorm:"not null;default:true"`
	Comment       string    `json:"comment,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
