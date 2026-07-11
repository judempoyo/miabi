// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/image"
)

// ImageHandler exposes the built-image catalog: provenance listing and deletion
// (guarded against removing an image a live deployment or pinned release uses).
type ImageHandler struct {
	svc     *image.Service
	remover image.Remover
	audit   *audit.Logger
}

// NewImageHandler wires the image catalog handler. remover deletes the physical
// image from the local node on delete (may be nil for DB-only).
func NewImageHandler(svc *image.Service, remover image.Remover, auditLog *audit.Logger) *ImageHandler {
	return &ImageHandler{svc: svc, remover: remover, audit: auditLog}
}

// List returns the workspace catalog, optionally filtered by ?app=<id>.
func (h *ImageHandler) List(c *okapi.Context) error {
	wsID := middlewares.WorkspaceID(c)
	var appID *uint
	if raw := c.Query("app"); raw != "" {
		if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
			v := uint(id)
			appID = &v
		}
	}
	out, err := h.svc.List(wsID, appID)
	if err != nil {
		return c.AbortInternalServerError("failed to list images", err)
	}
	return ok(c, out)
}

// Get returns a single catalog image.
func (h *ImageHandler) Get(c *okapi.Context) error {
	id, err := uintParam(c, "imageID")
	if err != nil {
		return c.AbortBadRequest("invalid image id")
	}
	img, err := h.svc.Get(middlewares.WorkspaceID(c), id)
	if err != nil {
		return c.AbortNotFound("image not found")
	}
	return ok(c, img)
}

// Delete removes a catalog image unless it is referenced by a live or pinned
// release.
func (h *ImageHandler) Delete(c *okapi.Context) error {
	id, err := uintParam(c, "imageID")
	if err != nil {
		return c.AbortBadRequest("invalid image id")
	}
	wsID := middlewares.WorkspaceID(c)
	switch err := h.svc.Delete(c.Request().Context(), wsID, id, h.remover); {
	case err == nil:
		actor := middlewares.UserID(c)
		h.audit.Record(audit.Entry{ActorID: &actor, WorkspaceID: &wsID, Action: "image.delete",
			TargetType: "image", TargetID: strconv.Itoa(int(id)), IP: c.RealIP()})
		return message(c, "image deleted")
	case errors.Is(err, image.ErrNotFound):
		return c.AbortNotFound("image not found")
	case errors.Is(err, image.ErrInUse):
		return c.AbortWithError(409, err)
	default:
		return c.AbortInternalServerError("failed to delete image", err)
	}
}
