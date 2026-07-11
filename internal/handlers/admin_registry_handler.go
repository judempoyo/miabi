// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"

	"github.com/jkaninda/logger"
	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/enterprise"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/registryserver"
)

// AdminRegistryHandler exposes the platform's built-in Docker registry settings
// to the super-admin. The registry itself is a Community feature (local storage);
// S3/MinIO storage is gated behind the FlagRegistryS3 entitlement.
type AdminRegistryHandler struct {
	svc   *registryserver.Service
	ee    enterprise.EE
	audit *audit.Logger
	// ensure re-creates/tears down the registry container after a settings change.
	// Injected by the routes layer (it owns the control-plane Docker client).
	ensure func(context.Context) error
	// gc runs a garbage collection on demand.
	gc func(context.Context) error
}

func NewAdminRegistryHandler(svc *registryserver.Service, ee enterprise.EE, auditLog *audit.Logger) *AdminRegistryHandler {
	return &AdminRegistryHandler{svc: svc, ee: ee, audit: auditLog}
}

// SetEnsure wires the callback that applies the settings to the running
// container (recreate on change, tear down when disabled).
func (h *AdminRegistryHandler) SetEnsure(fn func(context.Context) error) { h.ensure = fn }

// SetGC wires the on-demand garbage-collection callback.
func (h *AdminRegistryHandler) SetGC(fn func(context.Context) error) { h.gc = fn }

// RunGC triggers a registry garbage collection (read-only during the collect).
func (h *AdminRegistryHandler) RunGC(c *okapi.Context) error {
	if h.gc == nil {
		return c.AbortInternalServerError("garbage collection is unavailable", nil)
	}
	if err := h.gc(c.Request().Context()); err != nil {
		return c.AbortInternalServerError("garbage collection failed", err)
	}
	h.record(c, "registry.gc")
	return message(c, "garbage collection complete")
}

// UpdateRegistrySettingsRequest is the body for updating the registry settings.
// S3SecretKey is empty to keep the stored secret unchanged.
type UpdateRegistrySettingsRequest struct {
	Body struct {
		Enabled             bool   `json:"enabled"`
		Host                string `json:"host"`
		StorageType         string `json:"storage_type" enum:"filesystem,s3"`
		S3Endpoint          string `json:"s3_endpoint"`
		S3Bucket            string `json:"s3_bucket"`
		S3Region            string `json:"s3_region"`
		S3AccessKey         string `json:"s3_access_key"`
		S3SecretKey         string `json:"s3_secret_key"`
		S3ForcePathStyle    bool   `json:"s3_force_path_style"`
		DeleteEnabled       bool   `json:"delete_enabled"`
		PerWorkspaceQuotaMB int    `json:"per_workspace_quota_mb"`
	} `json:"body"`
}

// RegistrySettingsView is the settings response enriched with the effective host
// and whether S3 storage is licensed (so the UI can lock the option).
type RegistrySettingsView struct {
	*models.RegistrySettings
	EffectiveHost string `json:"effective_host"`
	S3Entitled    bool   `json:"s3_entitled"`
}

func (h *AdminRegistryHandler) view(st *models.RegistrySettings) RegistrySettingsView {
	return RegistrySettingsView{
		RegistrySettings: st,
		EffectiveHost:    h.svc.HostFor(st),
		S3Entitled:       h.ee.Has(enterprise.FlagRegistryS3),
	}
}

// GetSettings returns the registry settings (secret omitted).
func (h *AdminRegistryHandler) GetSettings(c *okapi.Context) error {
	st, err := h.svc.Get()
	if err != nil {
		return c.AbortInternalServerError("failed to load registry settings", err)
	}
	return ok(c, h.view(st))
}

// UpdateSettings upserts the registry settings and applies them to the container.
func (h *AdminRegistryHandler) UpdateSettings(c *okapi.Context, req *UpdateRegistrySettingsRequest) error {
	b := req.Body
	// S3/MinIO storage is an Enterprise feature; local (filesystem) storage is free.
	if b.StorageType == models.RegistryStorageS3 {
		if err := h.ee.RequireMutable(enterprise.FlagRegistryS3); err != nil {
			return entitlementAbort(c, err)
		}
	}
	if b.Enabled && b.StorageType == models.RegistryStorageS3 && b.S3Bucket == "" {
		return c.AbortBadRequest("an S3 bucket is required for the S3 storage driver")
	}
	var secret *string
	if b.S3SecretKey != "" {
		secret = &b.S3SecretKey
	}
	st, err := h.svc.Save(registryserver.SaveInput{
		Enabled:             b.Enabled,
		Host:                b.Host,
		StorageType:         b.StorageType,
		S3Endpoint:          b.S3Endpoint,
		S3Bucket:            b.S3Bucket,
		S3Region:            b.S3Region,
		S3AccessKey:         b.S3AccessKey,
		S3SecretKey:         secret,
		S3ForcePathStyle:    b.S3ForcePathStyle,
		DeleteEnabled:       b.DeleteEnabled,
		PerWorkspaceQuotaMB: b.PerWorkspaceQuotaMB,
	})
	if err != nil {
		return c.AbortInternalServerError("failed to save registry settings", err)
	}
	// Apply to the running container (recreate on change / tear down when disabled),
	// best-effort so a Docker hiccup doesn't fail the settings save.
	if h.ensure != nil {
		if err := h.ensure(c.Request().Context()); err != nil {
			logger.Warn("registry ensure after settings change failed", "error", err)
		}
	}
	h.record(c, "registry.settings_update")
	return ok(c, h.view(st))
}

func (h *AdminRegistryHandler) record(c *okapi.Context, action string) {
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{ActorID: &actor, Action: action, TargetType: "registry", IP: c.RealIP()})
}
