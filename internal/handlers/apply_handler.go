// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/apply"
	"github.com/miabi-io/miabi/internal/services/audit"
)

// ApplyHandler exposes the declarative apply API: preview (dry run) or converge
// a workspace to a bundle of miabi.io/v1 manifests.
type ApplyHandler struct {
	svc   *apply.Service
	audit *audit.Logger
}

func NewApplyHandler(svc *apply.Service, auditLog *audit.Logger) *ApplyHandler {
	return &ApplyHandler{svc: svc, audit: auditLog}
}

// ApplyRequest carries the manifest bundle and apply options.
type ApplyRequest struct {
	Body struct {
		// Manifests is a single- or multi-document miabi.io/v1 YAML bundle.
		Manifests string `json:"manifests" required:"true"`
		// Prune deletes managed resources absent from the bundle (opt-in).
		Prune bool `json:"prune"`
		// DryRun returns the plan without applying it.
		DryRun bool `json:"dry_run"`
		// Delete removes exactly the resources the bundle names (the inverse of an
		// apply) instead of converging to them. Honors DryRun.
		Delete bool `json:"delete"`
	} `json:"body"`
}

// Apply previews or executes a declarative bundle.
func (h *ApplyHandler) Apply(c *okapi.Context, req *ApplyRequest) error {
	wsID := middlewares.WorkspaceID(c)
	ctx := c.Request().Context()

	// Delete mode: remove exactly the resources the bundle names (inverse of apply).
	if req.Body.Delete {
		res, err := h.svc.Delete(ctx, wsID, []byte(req.Body.Manifests), req.Body.DryRun)
		if err != nil {
			return h.mapErr(c, err)
		}
		if req.Body.DryRun {
			return ok(c, res.Plan)
		}
		actor := middlewares.UserID(c)
		h.audit.Record(audit.Entry{ActorID: &actor, WorkspaceID: &wsID, Action: "apply.delete",
			TargetType: "workspace", TargetID: strconv.Itoa(int(wsID)), IP: c.RealIP()})
		return ok(c, res)
	}

	opts := apply.Options{Prune: req.Body.Prune}
	if req.Body.DryRun {
		plan, _, err := h.svc.Plan(ctx, wsID, []byte(req.Body.Manifests), opts)
		if err != nil {
			return h.mapErr(c, err)
		}
		return ok(c, plan)
	}

	res, err := h.svc.Apply(ctx, wsID, []byte(req.Body.Manifests), opts)
	if err != nil {
		return h.mapErr(c, err)
	}
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{ActorID: &actor, WorkspaceID: &wsID, Action: "apply.run",
		TargetType: "workspace", TargetID: strconv.Itoa(int(wsID)), IP: c.RealIP()})
	return ok(c, res)
}

func (h *ApplyHandler) mapErr(c *okapi.Context, err error) error {
	switch {
	case errors.Is(err, apply.ErrInvalidManifest):
		return c.AbortBadRequest(err.Error())
	default:
		return c.AbortInternalServerError("apply failed", err)
	}
}
