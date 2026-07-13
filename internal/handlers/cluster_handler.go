// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/nodes"
	"github.com/miabi-io/miabi/internal/services/audit"
	"github.com/miabi-io/miabi/internal/services/cluster"
	"github.com/miabi-io/miabi/internal/services/node"
)

// ClusterHandler exposes platform-admin cluster (Docker Swarm) management:
// status, enable/adopt/disable, and per-node join/leave. Cluster mode is opt-in
// and auto-detected — these endpoints work on plain Docker too, simply reporting
// "not enabled" and refusing mutations until swarm mode is on.
type ClusterHandler struct {
	cluster *cluster.Service
	nodes   *node.Service
	audit   *audit.Logger
}

func NewClusterHandler(c *cluster.Service, n *node.Service, auditLog *audit.Logger) *ClusterHandler {
	return &ClusterHandler{cluster: c, nodes: n, audit: auditLog}
}

// Status returns the manager's current cluster (swarm) status.
func (h *ClusterHandler) Status(c *okapi.Context) error {
	return ok(c, h.cluster.Status())
}

// EnableClusterRequest enables (or adopts) cluster mode.
type EnableClusterRequest struct {
	Body struct {
		// AdvertiseAddr is the address swarm peers reach this manager on (its
		// private/WG address, host or host:port). Required when initializing a new
		// swarm; ignored when adopting one Docker is already in.
		AdvertiseAddr string `json:"advertise_addr"`
	} `json:"body"`
}

// Enable puts the manager into swarm mode (or adopts an existing swarm).
func (h *ClusterHandler) Enable(c *okapi.Context, req *EnableClusterRequest) error {
	status, err := h.cluster.Enable(c.Request().Context(), req.Body.AdvertiseAddr)
	if err != nil {
		if errors.Is(err, cluster.ErrAdvertiseAddrRequired) {
			return c.AbortBadRequest("an advertise address is required to enable cluster mode")
		}
		return c.AbortInternalServerError("failed to enable cluster mode", err)
	}
	h.record(c, "cluster.enable", 0)
	return ok(c, status)
}

// Disable removes the manager (and member nodes) from the swarm.
func (h *ClusterHandler) Disable(c *okapi.Context) error {
	if err := h.cluster.Disable(c.Request().Context()); err != nil {
		if errors.Is(err, cluster.ErrNotEnabled) {
			return c.AbortBadRequest("cluster mode is not enabled")
		}
		return c.AbortInternalServerError("failed to disable cluster mode", err)
	}
	h.record(c, "cluster.disable", 0)
	return message(c, "cluster mode disabled")
}

// ApplyNetworking converts the workspace networks still on node-local bridges into
// swarm overlays, so apps and databases reach each other across nodes.
//
// Enable already does this on the transition into cluster mode. This is the
// explicit action for an install that was ALREADY clustered when it upgraded (it
// never saw that transition, so its workspaces are still on per-node islands), and
// for re-running the conversion after a node that was offline comes back.
//
// It briefly drops in-flight connections inside each workspace; containers are not
// restarted.
func (h *ClusterHandler) ApplyNetworking(c *okapi.Context) error {
	if err := h.cluster.ApplyNetworking(c.Request().Context()); err != nil {
		if errors.Is(err, cluster.ErrNotEnabled) {
			return c.AbortBadRequest("cluster mode is not enabled")
		}
		return c.AbortInternalServerError("failed to apply cluster networking", err)
	}
	h.record(c, "cluster.network.apply", 0)
	return ok(c, h.cluster.Status())
}

// Preflight reports what this host can and cannot do before cluster mode is turned
// on: whether its Docker engine can carry the overlay data plane to other hosts at
// all, and the ports that must be open between nodes. Read-only.
func (h *ClusterHandler) Preflight(c *okapi.Context) error {
	p, err := h.cluster.Preflight(c.Request().Context())
	if err != nil {
		return c.AbortInternalServerError("failed to inspect the Docker engine", err)
	}
	return ok(c, p)
}

// NetCheck probes the cluster's overlay data plane between every pair of nodes,
// separating the three failures that are indistinguishable from inside an app: a
// name that will not resolve, a connection that never completes, and a payload that
// silently dies at the MTU.
//
// It starts and removes probe containers, so it is a mutation, not a read.
func (h *ClusterHandler) NetCheck(c *okapi.Context) error {
	res, err := h.cluster.NetCheck(c.Request().Context())
	if err != nil {
		if errors.Is(err, cluster.ErrNotEnabled) {
			return c.AbortBadRequest("cluster mode is not enabled")
		}
		return c.AbortInternalServerError("failed to run the network check", err)
	}
	h.record(c, "cluster.netcheck", 0)
	return ok(c, res)
}

// SetAvailabilityRequest changes a swarm node's scheduling availability.
type SetAvailabilityRequest struct {
	Body struct {
		// Availability is active | pause | drain. Drain reschedules the node's tasks
		// away, which is what makes it safe to reboot.
		Availability string `json:"availability"`
	} `json:"body"`
}

// SetAvailability changes a swarm node's scheduling availability. Keyed by SWARM
// node id, so an unmanaged member (no Miabi agent) can be drained too.
func (h *ClusterHandler) SetAvailability(c *okapi.Context, req *SetAvailabilityRequest) error {
	swarmNodeID := c.Param("swarmNodeID")
	if swarmNodeID == "" {
		return c.AbortBadRequest("swarm node id is required")
	}
	err := h.cluster.SetAvailability(c.Request().Context(), swarmNodeID, req.Body.Availability)
	switch {
	case errors.Is(err, cluster.ErrNotEnabled):
		return c.AbortBadRequest("cluster mode is not enabled")
	case errors.Is(err, cluster.ErrInvalidAvailability):
		return c.AbortBadRequest(err.Error())
	case err != nil:
		return c.AbortInternalServerError("failed to change node availability", err)
	}
	h.record(c, "cluster.node.availability", 0)
	return message(c, "node availability set to "+req.Body.Availability)
}

// NodeTasks lists the service tasks the scheduler placed on a swarm node. This is
// the only way to see the workload of an unmanaged member — the containers live on
// the node, which Miabi has no Docker client for.
func (h *ClusterHandler) NodeTasks(c *okapi.Context) error {
	tasks, err := h.cluster.Tasks(c.Request().Context(), c.Param("swarmNodeID"))
	if err != nil {
		return c.AbortInternalServerError("failed to list the node's tasks", err)
	}
	return ok(c, tasks)
}

// Members lists the swarm's nodes (docker node ls), annotated with whether each
// maps to a managed Miabi node. Drives the manager detail page's cluster view.
func (h *ClusterHandler) Members(c *okapi.Context) error {
	members, err := h.cluster.Members(c.Request().Context())
	if err != nil {
		return c.AbortInternalServerError("failed to list cluster nodes", err)
	}
	return ok(c, members)
}

// JoinToken returns the manual join command + worker token for joining a host
// that is not connected to the manager over the agent tunnel.
func (h *ClusterHandler) JoinToken(c *okapi.Context) error {
	inst, err := h.cluster.JoinInstructions(c.Request().Context())
	if err != nil {
		switch {
		case errors.Is(err, cluster.ErrNotEnabled):
			return c.AbortBadRequest("cluster mode is not enabled")
		case errors.Is(err, cluster.ErrManagerAddrUnknown):
			return c.AbortWithError(http.StatusConflict, err)
		default:
			return c.AbortInternalServerError("failed to get cluster join command", err)
		}
	}
	return ok(c, inst)
}

// JoinNode joins a worker node to the swarm.
func (h *ClusterHandler) JoinNode(c *okapi.Context) error {
	id, err := h.nodeID(c)
	if err != nil {
		return c.AbortBadRequest("invalid node id")
	}
	if err := h.cluster.JoinNode(c.Request().Context(), id); err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, "cluster.node_join", id)
	return message(c, "node joined the cluster")
}

// LeaveNode removes a worker node from the swarm.
func (h *ClusterHandler) LeaveNode(c *okapi.Context) error {
	id, err := h.nodeID(c)
	if err != nil {
		return c.AbortBadRequest("invalid node id")
	}
	if err := h.cluster.LeaveNode(c.Request().Context(), id, true); err != nil {
		return h.mapErr(c, err)
	}
	h.record(c, "cluster.node_leave", id)
	return message(c, "node removed from the cluster")
}

func (h *ClusterHandler) nodeID(c *okapi.Context) (uint, error) {
	return resolveID(c.Param("nodeID"), h.nodes.IDByUID)
}

func (h *ClusterHandler) mapErr(c *okapi.Context, err error) error {
	switch {
	case errors.Is(err, cluster.ErrNotEnabled):
		return c.AbortBadRequest("cluster mode is not enabled")
	case errors.Is(err, cluster.ErrManagerNode):
		return c.AbortBadRequest("the manager node cannot be used for this operation")
	case errors.Is(err, cluster.ErrManagerAddrUnknown):
		return c.AbortWithError(http.StatusConflict, err)
	case errors.Is(err, node.ErrNodeNotFound):
		return c.AbortNotFound("node not found")
	case errors.Is(err, nodes.ErrNodeOffline):
		return c.AbortWithError(http.StatusServiceUnavailable, err)
	default:
		return c.AbortInternalServerError("cluster operation failed", err)
	}
}

func (h *ClusterHandler) record(c *okapi.Context, action string, id uint) {
	actor := middlewares.UserID(c)
	target := ""
	if id != 0 {
		target = strconv.Itoa(int(id))
	}
	h.audit.Record(audit.Entry{ActorID: &actor, Action: action, TargetType: "cluster", TargetID: target, IP: c.RealIP()})
}
