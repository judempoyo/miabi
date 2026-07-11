// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/environment"
)

// EnvironmentHandler exposes promotion-environment CRUD.
type EnvironmentHandler struct {
	svc   *environment.Service
	audit *audit.Logger
}

func NewEnvironmentHandler(svc *environment.Service, auditLog *audit.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{svc: svc, audit: auditLog}
}

type EnvironmentRequest struct {
	Body struct {
		Name              string `json:"name" required:"true"`
		DisplayName       string `json:"display_name"`
		Description       string `json:"description"`
		Rank              int    `json:"rank"`
		RequiredApprovals int    `json:"required_approvals"`
		GitSourceID       *uint  `json:"git_source_id"`
	} `json:"body"`
}

func (h *EnvironmentHandler) toInput(req *EnvironmentRequest) environment.Input {
	return environment.Input{
		Name: req.Body.Name, DisplayName: req.Body.DisplayName, Description: req.Body.Description, Rank: req.Body.Rank,
		RequiredApprovals: req.Body.RequiredApprovals, GitSourceID: req.Body.GitSourceID,
	}
}

func (h *EnvironmentHandler) List(c *okapi.Context) error {
	out, err := h.svc.List(middlewares.WorkspaceID(c))
	if err != nil {
		return c.AbortInternalServerError("failed to list environments", err)
	}
	return ok(c, out)
}

func (h *EnvironmentHandler) Create(c *okapi.Context, req *EnvironmentRequest) error {
	wsID := middlewares.WorkspaceID(c)
	env, err := h.svc.Create(wsID, h.toInput(req))
	if err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "environment.create", env.ID)
	return created(c, env)
}

func (h *EnvironmentHandler) Get(c *okapi.Context) error {
	id, err := uintParam(c, "environmentID")
	if err != nil {
		return c.AbortBadRequest("invalid environment id")
	}
	env, err := h.svc.Get(middlewares.WorkspaceID(c), id)
	if err != nil {
		return c.AbortNotFound("environment not found")
	}
	return ok(c, env)
}

func (h *EnvironmentHandler) Update(c *okapi.Context, req *EnvironmentRequest) error {
	id, err := uintParam(c, "environmentID")
	if err != nil {
		return c.AbortBadRequest("invalid environment id")
	}
	wsID := middlewares.WorkspaceID(c)
	env, err := h.svc.Update(wsID, id, h.toInput(req))
	if err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "environment.update", env.ID)
	return ok(c, env)
}

func (h *EnvironmentHandler) Delete(c *okapi.Context) error {
	id, err := uintParam(c, "environmentID")
	if err != nil {
		return c.AbortBadRequest("invalid environment id")
	}
	wsID := middlewares.WorkspaceID(c)
	if err := h.svc.Delete(wsID, id); err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "environment.delete", id)
	return message(c, "environment deleted")
}

func (h *EnvironmentHandler) record(c *okapi.Context, wsID uint, action string, id uint) {
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{ActorID: &actor, WorkspaceID: &wsID, Action: action,
		TargetType: "environment", TargetID: strconv.Itoa(int(id)), IP: c.RealIP()})
}

func (h *EnvironmentHandler) mapErr(c *okapi.Context, err error) error {
	switch {
	case errors.Is(err, environment.ErrNotFound):
		return c.AbortNotFound("environment not found")
	case errors.Is(err, environment.ErrNameTaken):
		return c.AbortWithError(409, err)
	case errors.Is(err, environment.ErrNameRequired):
		return c.AbortBadRequest(err.Error())
	default:
		return c.AbortInternalServerError("environment operation failed", err)
	}
}
