// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/webhook"
)

type WebhookHandler struct {
	svc   *webhook.Service
	audit *audit.Logger
}

func NewWebhookHandler(svc *webhook.Service, auditLog *audit.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, audit: auditLog}
}

type CreateWebhookRequest struct {
	Body struct {
		Name    string            `json:"name"`
		URL     string            `json:"url" required:"true" format:"uri"`
		Events  []string          `json:"events" required:"true" minItems:"1"`
		Headers map[string]string `json:"headers"`
		Enabled bool              `json:"enabled"`
	} `json:"body"`
}

type UpdateWebhookRequest struct {
	Body struct {
		Name    string            `json:"name"`
		URL     string            `json:"url"`
		Events  []string          `json:"events"`
		Headers map[string]string `json:"headers"`
		Enabled bool              `json:"enabled"`
	} `json:"body"`
}

func (h *WebhookHandler) List(c *okapi.Context) error {
	ws, err := h.svc.List(middlewares.WorkspaceID(c))
	if err != nil {
		return c.AbortInternalServerError("failed to list webhooks", err)
	}
	return ok(c, ws)
}

func (h *WebhookHandler) Create(c *okapi.Context, req *CreateWebhookRequest) error {
	wsID := middlewares.WorkspaceID(c)
	w, err := h.svc.Create(wsID, webhook.Input{
		Name: req.Body.Name, URL: req.Body.URL, Events: req.Body.Events,
		Headers: req.Body.Headers, Enabled: req.Body.Enabled,
	})
	if err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "webhook.create", w.ID)
	return created(c, w)
}

func (h *WebhookHandler) Get(c *okapi.Context) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	w, err := h.svc.Get(middlewares.WorkspaceID(c), id)
	if err != nil {
		return c.AbortNotFound("webhook not found")
	}
	return ok(c, w)
}

func (h *WebhookHandler) Update(c *okapi.Context, req *UpdateWebhookRequest) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	wsID := middlewares.WorkspaceID(c)
	w, err := h.svc.Update(wsID, id, webhook.Input{
		Name: req.Body.Name, URL: req.Body.URL, Events: req.Body.Events,
		Headers: req.Body.Headers, Enabled: req.Body.Enabled,
	})
	if err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "webhook.update", w.ID)
	return ok(c, w)
}

func (h *WebhookHandler) Delete(c *okapi.Context) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	wsID := middlewares.WorkspaceID(c)
	if err := h.svc.Delete(wsID, id); err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, wsID, "webhook.delete", id)
	return message(c, "webhook deleted")
}

func (h *WebhookHandler) Test(c *okapi.Context) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	if err := h.svc.Test(c.Request().Context(), middlewares.WorkspaceID(c), id); err != nil {
		if errors.Is(err, webhook.ErrNotFound) {
			return c.AbortNotFound("webhook not found")
		}
		return c.AbortWithError(400, err)
	}
	return message(c, "test delivery succeeded")
}

func (h *WebhookHandler) Deliveries(c *okapi.Context) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	limit, _ := pageParams(c)
	ds, err := h.svc.Deliveries(middlewares.WorkspaceID(c), id, limit)
	if err != nil {
		if errors.Is(err, webhook.ErrNotFound) {
			return c.AbortNotFound("webhook not found")
		}
		return c.AbortInternalServerError("failed to list deliveries", err)
	}
	return ok(c, ds)
}

// Redeliver re-sends a past delivery's source event to the webhook.
func (h *WebhookHandler) Redeliver(c *okapi.Context) error {
	id, err := h.id(c)
	if err != nil {
		return c.AbortBadRequest("invalid webhook id")
	}
	deliveryID, err := strconv.Atoi(c.Param("deliveryID"))
	if err != nil || deliveryID <= 0 {
		return c.AbortBadRequest("invalid delivery id")
	}
	wsID := middlewares.WorkspaceID(c)
	if err := h.svc.Redeliver(wsID, id, uint(deliveryID)); err != nil {
		if errors.Is(err, webhook.ErrNotFound) {
			return c.AbortNotFound("delivery not found")
		}
		return c.AbortInternalServerError("failed to redeliver", err)
	}
	h.record(c, wsID, "webhook.redeliver", id)
	return message(c, "delivery re-queued")
}

func (h *WebhookHandler) id(c *okapi.Context) (uint, error) {
	id, err := strconv.Atoi(c.Param("webhookID"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid webhook id")
	}
	return uint(id), nil
}

func (h *WebhookHandler) record(c *okapi.Context, wsID uint, action string, id uint) {
	actor := middlewares.UserID(c)
	h.audit.Record(audit.Entry{ActorID: &actor, WorkspaceID: &wsID, Action: action, TargetType: "webhook", TargetID: strconv.Itoa(int(id)), IP: c.RealIP()})
}

func (h *WebhookHandler) mapErr(c *okapi.Context, err error) error {
	switch {
	case errors.Is(err, webhook.ErrURLRequired), errors.Is(err, webhook.ErrURLInvalid),
		errors.Is(err, webhook.ErrURLBlocked), errors.Is(err, webhook.ErrInvalidEvent):
		return c.AbortBadRequest(err.Error())
	case errors.Is(err, webhook.ErrNotFound):
		return c.AbortNotFound("webhook not found")
	default:
		return c.AbortInternalServerError("webhook operation failed", err)
	}
}
