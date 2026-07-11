// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/backupsettings"
)

// WorkspaceBackupSettingsHandler exposes a workspace's shared S3 backup target.
type WorkspaceBackupSettingsHandler struct {
	svc   *backupsettings.Service
	audit *audit.Logger
}

func NewWorkspaceBackupSettingsHandler(svc *backupsettings.Service, auditLog *audit.Logger) *WorkspaceBackupSettingsHandler {
	return &WorkspaceBackupSettingsHandler{svc: svc, audit: auditLog}
}

// UpdateBackupSettingsRequest is the body for updating (and validating) settings.
// S3SecretKey is empty to keep the stored secret unchanged.
type UpdateBackupSettingsRequest struct {
	Body struct {
		S3Enabled        bool   `json:"s3_enabled"`
		S3Endpoint       string `json:"s3_endpoint"`
		S3Bucket         string `json:"s3_bucket"`
		S3Region         string `json:"s3_region"`
		S3AccessKey      string `json:"s3_access_key"`
		S3SecretKey      string `json:"s3_secret_key"`
		S3UseSSL         bool   `json:"s3_use_ssl"`
		S3ForcePathStyle bool   `json:"s3_force_path_style"`

		DatabaseBackupPath string `json:"database_backup_path"`
		VolumeBackupPath   string `json:"volume_backup_path"`
	} `json:"body"`
}

// Get returns the workspace's backup settings (secret omitted).
func (h *WorkspaceBackupSettingsHandler) Get(c *okapi.Context) error {
	st, err := h.svc.Get(middlewares.WorkspaceID(c))
	if err != nil {
		return c.AbortInternalServerError("failed to load backup settings", err)
	}
	return ok(c, st)
}

// Update upserts the workspace's backup settings.
func (h *WorkspaceBackupSettingsHandler) Update(c *okapi.Context, req *UpdateBackupSettingsRequest) error {
	wsID := middlewares.WorkspaceID(c)
	b := req.Body
	if b.S3Enabled && b.S3Bucket == "" {
		return c.AbortBadRequest("an S3 bucket is required when S3 is enabled")
	}
	var secret *string
	if b.S3SecretKey != "" {
		secret = &b.S3SecretKey
	}
	st, err := h.svc.Save(wsID, backupsettings.SaveInput{
		S3Enabled:          b.S3Enabled,
		S3Endpoint:         b.S3Endpoint,
		S3Bucket:           b.S3Bucket,
		S3Region:           b.S3Region,
		S3AccessKey:        b.S3AccessKey,
		S3SecretKey:        secret,
		S3UseSSL:           b.S3UseSSL,
		S3ForcePathStyle:   b.S3ForcePathStyle,
		DatabaseBackupPath: b.DatabaseBackupPath,
		VolumeBackupPath:   b.VolumeBackupPath,
	})
	if err != nil {
		return c.AbortInternalServerError("failed to save backup settings", err)
	}
	h.record(c, wsID, "backup.settings_update")
	return ok(c, st)
}

// Test validates the supplied (or stored) S3 settings. This is a structural
// check that the configuration is complete; a live bucket probe runs on the
// first backup via the one-shot backup tool.
func (h *WorkspaceBackupSettingsHandler) Test(c *okapi.Context, req *UpdateBackupSettingsRequest) error {
	wsID := middlewares.WorkspaceID(c)
	b := req.Body
	if b.S3Bucket == "" {
		return c.AbortBadRequest("an S3 bucket is required")
	}
	if b.S3AccessKey == "" {
		return c.AbortBadRequest("an access key is required")
	}
	// The secret may already be stored (the UI need not resend it).
	secretSet := b.S3SecretKey != ""
	if !secretSet {
		if cur, err := h.svc.Get(wsID); err == nil {
			secretSet = cur.S3SecretSet
		}
	}
	if !secretSet {
		return c.AbortBadRequest("a secret key is required")
	}
	return message(c, "backup settings look valid")
}

func (h *WorkspaceBackupSettingsHandler) record(c *okapi.Context, wsID uint, action string) {
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{
		ActorID:     &actor,
		WorkspaceID: &wsID,
		Action:      action,
		TargetType:  "backup_settings",
		TargetID:    strconv.Itoa(int(wsID)),
		IP:          c.RealIP(),
	})
}
