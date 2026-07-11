// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/platformimage"
	"github.com/miabi-io/miabi/internal/services/settings"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

// DeploymentConfigHandler is the admin Deployment Config: the catalog of every
// image the platform runs, with per-image overrides and a global registry mirror.
type DeploymentConfigHandler struct {
	resolver *platformimage.Resolver
	repo     *repositories.SettingRepository
	provider *settings.Provider
	audit    *audit.Logger
}

func NewDeploymentConfigHandler(resolver *platformimage.Resolver, repo *repositories.SettingRepository, provider *settings.Provider, auditLog *audit.Logger) *DeploymentConfigHandler {
	return &DeploymentConfigHandler{resolver: resolver, repo: repo, provider: provider, audit: auditLog}
}

// Get returns the image catalog (default / override / effective per image) and
// the registry mirror.
func (h *DeploymentConfigHandler) Get(c *okapi.Context) error {
	return ok(c, map[string]any{
		"images": h.resolver.Catalog(),
		"mirror": h.resolver.Mirror(),
	})
}

type UpdateDeploymentConfigRequest struct {
	Body struct {
		// Mirror is the global registry mirror/prefix (empty = none).
		Mirror string `json:"mirror"`
		// Images maps catalog keys to override refs ("" clears the override).
		Images map[string]string `json:"images"`
	} `json:"body"`
}

// Update writes image overrides + the mirror. Unknown keys are ignored.
func (h *DeploymentConfigHandler) Update(c *okapi.Context, req *UpdateDeploymentConfigRequest) error {
	toSave := []models.Setting{
		{Key: platformimage.MirrorSettingKey(), Value: req.Body.Mirror, Type: models.SettingTypeString},
	}
	for key, val := range req.Body.Images {
		if !h.resolver.ValidKey(key) {
			continue
		}
		toSave = append(toSave, models.Setting{Key: platformimage.SettingKey(key), Value: val, Type: models.SettingTypeString})
	}
	if err := h.repo.BulkUpsert(toSave); err != nil {
		return c.AbortInternalServerError("failed to save deployment config", err)
	}
	h.provider.Invalidate()
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{ActorID: &actor, Action: "admin.deployment_config.update", TargetType: "deployment_config", IP: c.RealIP(), Metadata: map[string]any{"count": len(toSave)}})
	return h.Get(c)
}
