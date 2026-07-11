// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/enterprise"
	"github.com/miabi-io/miabi/internal/logstore"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/platformbackup"
)

// AdminPlatformBackupHandler exposes platform (control-plane) disaster-recovery
// backups to the platform super-admin. Every endpoint is additionally gated
// behind the Enterprise FlagPlatformBackup entitlement.
type AdminPlatformBackupHandler struct {
	svc   *platformbackup.Service
	ee    enterprise.EE
	audit *audit.Logger
	// reschedule re-registers the platform backup cron after a settings change.
	// Injected by the routes layer (it owns the cron Manager).
	reschedule func(*models.PlatformBackupSettings)
	logs       *logstore.Store
}

func NewAdminPlatformBackupHandler(svc *platformbackup.Service, ee enterprise.EE, auditLog *audit.Logger) *AdminPlatformBackupHandler {
	return &AdminPlatformBackupHandler{svc: svc, ee: ee, audit: auditLog}
}

// SetLogStore wires the shared execution-log store so a platform-backup run's
// full log can be downloaded from the store (falling back to the DB tail). nil
// keeps tail-only.
func (h *AdminPlatformBackupHandler) SetLogStore(s *logstore.Store) { h.logs = s }

// LogsDownload streams a platform-backup run's full log as a file download.
func (h *AdminPlatformBackupHandler) LogsDownload(c *okapi.Context) error {
	b, err := h.loadBackup(c)
	if err != nil {
		return c.AbortNotFound("platform backup not found")
	}
	return streamLogDownload(c, h.logs, b.LogRef, b.Logs, "platform-backup-"+strconv.FormatUint(uint64(b.ID), 10)+".log")
}

// SetReschedule wires the callback that re-registers the platform backup cron
// schedule whenever the settings change.
func (h *AdminPlatformBackupHandler) SetReschedule(fn func(*models.PlatformBackupSettings)) {
	h.reschedule = fn
}

// UpdatePlatformBackupSettingsRequest is the body for updating/validating the
// platform backup target + policy. S3SecretKey is empty to keep the stored
// secret unchanged.
type UpdatePlatformBackupSettingsRequest struct {
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

		ScheduleEnabled bool     `json:"schedule_enabled"`
		ScheduleCron    string   `json:"schedule_cron"`
		MaxBackups      int      `json:"max_backups"`
		RetentionDays   int      `json:"retention_days"`
		Volumes         []string `json:"volumes"`
	} `json:"body"`
}

// CreatePlatformBackupRequest selects what to back up now.
type CreatePlatformBackupRequest struct {
	Body struct {
		Database bool     `json:"database"` // back up the control-plane DB (default when nothing selected)
		Volumes  []string `json:"volumes"`  // platform volumes to back up
	} `json:"body"`
}

// RestorePlatformBackupRequest carries the explicit confirmation for a
// destructive restore.
type RestorePlatformBackupRequest struct {
	Body struct {
		Confirm bool `json:"confirm"`
	} `json:"body"`
}

// GetSettings returns the platform backup settings (secret omitted).
func (h *AdminPlatformBackupHandler) GetSettings(c *okapi.Context) error {
	if err := h.ee.Require(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	st, err := h.svc.GetSettings()
	if err != nil {
		return c.AbortInternalServerError("failed to load platform backup settings", err)
	}
	return ok(c, st)
}

// UpdateSettings upserts the platform backup settings.
func (h *AdminPlatformBackupHandler) UpdateSettings(c *okapi.Context, req *UpdatePlatformBackupSettingsRequest) error {
	if err := h.ee.RequireMutable(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	b := req.Body
	if b.S3Enabled && b.S3Bucket == "" {
		return c.AbortBadRequest("an S3 bucket is required when S3 is enabled")
	}
	if b.ScheduleEnabled && b.ScheduleCron == "" {
		return c.AbortBadRequest("a cron expression is required when the schedule is enabled")
	}
	var secret *string
	if b.S3SecretKey != "" {
		secret = &b.S3SecretKey
	}
	st, err := h.svc.SaveSettings(platformbackup.SaveInput{
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
		ScheduleEnabled:    b.ScheduleEnabled,
		ScheduleCron:       b.ScheduleCron,
		MaxBackups:         b.MaxBackups,
		RetentionDays:      b.RetentionDays,
		Volumes:            b.Volumes,
	})
	if err != nil {
		return c.AbortInternalServerError("failed to save platform backup settings", err)
	}
	if h.reschedule != nil {
		h.reschedule(st)
	}
	h.record(c, "platform.backup.settings_update", 0)
	return ok(c, st)
}

// TestSettings validates that the supplied (or stored) S3 settings are complete.
// A live bucket probe runs on the first backup via the one-shot backup tool.
func (h *AdminPlatformBackupHandler) TestSettings(c *okapi.Context, req *UpdatePlatformBackupSettingsRequest) error {
	if err := h.ee.Require(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	b := req.Body
	if b.S3Bucket == "" {
		return c.AbortBadRequest("an S3 bucket is required")
	}
	if b.S3AccessKey == "" {
		return c.AbortBadRequest("an access key is required")
	}
	secretSet := b.S3SecretKey != ""
	if !secretSet {
		if cur, err := h.svc.GetSettings(); err == nil {
			secretSet = cur.S3SecretSet
		}
	}
	if !secretSet {
		return c.AbortBadRequest("a secret key is required")
	}
	return message(c, "platform backup settings look valid")
}

// ListBackups returns the platform backup history (paginated, newest first).
func (h *AdminPlatformBackupHandler) ListBackups(c *okapi.Context) error {
	if err := h.ee.Require(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	page, size, offset := normalizePageParams(queryInt(c, "page", 0), queryInt(c, "size", 20))
	items, total, err := h.svc.ListPaged(size, offset)
	if err != nil {
		return c.AbortInternalServerError("failed to list platform backups", err)
	}
	return paginated(c, items, total, page, size)
}

// CreateBackup runs a manual platform backup of the control-plane database and/or
// the selected platform volumes. Each backup starts pending; status advances as
// the worker runs.
func (h *AdminPlatformBackupHandler) CreateBackup(c *okapi.Context, req *CreatePlatformBackupRequest) error {
	if err := h.ee.RequireMutable(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	ctx := c.Request().Context()
	b := req.Body
	// Default to a database backup when nothing is selected.
	wantDB := b.Database || len(b.Volumes) == 0

	recs := make([]*models.PlatformBackup, 0, len(b.Volumes)+1)
	if wantDB {
		rec, err := h.svc.Create(ctx, models.PlatformBackupDatabase, "", "manual")
		if err != nil {
			return h.createErr(c, err)
		}
		recs = append(recs, rec)
	}
	for _, v := range b.Volumes {
		rec, err := h.svc.Create(ctx, models.PlatformBackupVolume, v, "manual")
		if err != nil {
			return h.createErr(c, err)
		}
		recs = append(recs, rec)
	}
	for _, rec := range recs {
		h.record(c, "platform.backup.create", rec.ID)
	}
	return created(c, recs)
}

func (h *AdminPlatformBackupHandler) createErr(c *okapi.Context, err error) error {
	if errors.Is(err, platformbackup.ErrS3NotConfigured) {
		return c.AbortBadRequest("configure an S3 target before backing up volumes (volume backups have no local destination)")
	}
	return c.AbortInternalServerError("failed to start platform backup", err)
}

// Restore restores a completed platform backup. Destructive: the control-plane DB
// is overwritten in place (recommend restoring onto a fresh instance for true DR),
// or the target volume is overwritten. The original MIABI_ENCRYPTION_KEY is still
// required afterward to decrypt the restored ciphertext.
func (h *AdminPlatformBackupHandler) Restore(c *okapi.Context, req *RestorePlatformBackupRequest) error {
	if err := h.ee.RequireMutable(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	if !req.Body.Confirm {
		return c.AbortBadRequest("restore must be explicitly confirmed")
	}
	b, err := h.loadBackup(c)
	if err != nil {
		return c.AbortNotFound("platform backup not found")
	}
	if b.Status != models.BackupCompleted {
		return c.AbortBadRequest("only a completed backup can be restored")
	}
	if err := h.svc.Restore(c.Request().Context(), b); err != nil {
		switch {
		case errors.Is(err, platformbackup.ErrNoArtifact):
			return c.AbortBadRequest("this backup has no artifact to restore")
		case errors.Is(err, platformbackup.ErrS3NotConfigured):
			return c.AbortBadRequest("configure the S3 target before restoring")
		default:
			return c.AbortInternalServerError("restore failed", err)
		}
	}
	h.record(c, "platform.backup.restore", b.ID)
	return message(c, "platform backup restored")
}

// Delete removes a platform backup record (and the local artifact, if any).
func (h *AdminPlatformBackupHandler) Delete(c *okapi.Context) error {
	if err := h.ee.RequireMutable(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	b, err := h.loadBackup(c)
	if err != nil {
		return c.AbortNotFound("platform backup not found")
	}
	if b.Status == models.BackupPending || b.Status == models.BackupRunning {
		return c.AbortBadRequest("cannot delete a backup that is still running")
	}
	if err := h.svc.Delete(c.Request().Context(), b); err != nil {
		return c.AbortInternalServerError("failed to delete platform backup", err)
	}
	h.record(c, "platform.backup.delete", b.ID)
	return message(c, "platform backup deleted")
}

// Download streams a local DB backup artifact to the client.
func (h *AdminPlatformBackupHandler) Download(c *okapi.Context) error {
	if err := h.ee.Require(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	b, err := h.loadBackup(c)
	if err != nil {
		return c.AbortNotFound("platform backup not found")
	}
	rc, size, filename, err := h.svc.Download(c.Request().Context(), b)
	if err != nil {
		switch {
		case errors.Is(err, platformbackup.ErrDownloadRemote):
			return c.AbortBadRequest(err.Error())
		case errors.Is(err, platformbackup.ErrNoArtifact):
			return c.AbortBadRequest("backup has no downloadable artifact")
		default:
			return c.AbortInternalServerError("failed to read backup", err)
		}
	}
	defer func() { _ = rc.Close() }()

	h.record(c, "platform.backup.download", b.ID)
	c.SetHeader("Content-Type", "application/gzip")
	c.SetHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
	if size > 0 {
		c.SetHeader("Content-Length", strconv.FormatInt(size, 10))
	}
	w := c.Response()
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
	return nil
}

// Volumes discovers candidate platform/system volumes the admin can include.
func (h *AdminPlatformBackupHandler) Volumes(c *okapi.Context) error {
	if err := h.ee.Require(enterprise.FlagPlatformBackup); err != nil {
		return entitlementAbort(c, err)
	}
	vols, err := h.svc.DiscoverVolumes(c.Request().Context())
	if err != nil {
		return c.AbortInternalServerError("failed to discover platform volumes", err)
	}
	return ok(c, vols)
}

func (h *AdminPlatformBackupHandler) loadBackup(c *okapi.Context) (*models.PlatformBackup, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return nil, errors.New("invalid backup id")
	}
	return h.svc.Get(uint(id))
}

func (h *AdminPlatformBackupHandler) record(c *okapi.Context, action string, id uint) {
	actor := middlewares.UserID(c)
	target := ""
	if id > 0 {
		target = strconv.Itoa(int(id))
	}
	h.audit.Record(audit.Entry{
		ActorID:    &actor,
		Action:     action,
		TargetType: "platform_backup",
		TargetID:   target,
		IP:         c.RealIP(),
	})
}
