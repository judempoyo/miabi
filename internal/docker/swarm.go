// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

// defaultSwarmListenAddr is the management-plane bind used when a caller does
// not specify one (the standard Docker Swarm port on all interfaces).
const defaultSwarmListenAddr = "0.0.0.0:2377"

// SwarmInfo summarizes an engine's participation in a Docker Swarm, as reported
// by `docker info`. It is available on any engine (no manager role required), so
// it is the cheap signal Miabi uses to auto-detect cluster mode.
type SwarmInfo struct {
	// LocalNodeState is the engine's swarm state: inactive | pending | active |
	// error | locked. "inactive" means plain (non-swarm) Docker.
	LocalNodeState string `json:"local_node_state"`
	// ControlAvailable is true when this engine is a reachable swarm manager
	// (i.e. it can drive cluster operations).
	ControlAvailable bool `json:"control_available"`
	// NodeID / NodeAddr identify this engine within the swarm; NodeAddr is the
	// address it advertises to peers (used as the join remote-address).
	NodeID   string `json:"node_id"`
	NodeAddr string `json:"node_addr"`
	// Managers / Nodes are cluster-wide counts (manager-reported; 0 elsewhere).
	Managers int `json:"managers"`
	Nodes    int `json:"nodes"`
	// RemoteManagers are the advertised addresses of the swarm's managers.
	RemoteManagers []string `json:"remote_managers,omitempty"`
	// Error carries the engine's swarm error string when LocalNodeState == error.
	Error string `json:"error,omitempty"`
}

// SwarmNode is one node as seen from a swarm manager (`docker node ls`).
type SwarmNode struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Role          string `json:"role"`         // manager | worker
	Availability  string `json:"availability"` // active | pause | drain
	State         string `json:"state"`        // ready | down | unknown | disconnected
	Leader        bool   `json:"leader"`
	Reachability  string `json:"reachability,omitempty"` // reachable | unreachable (managers)
	Addr          string `json:"addr,omitempty"`
	EngineVersion string `json:"engine_version,omitempty"`
}

// SwarmJoinTokens are the secrets a node uses to join a swarm in either role.
type SwarmJoinTokens struct {
	Worker  string `json:"worker"`
	Manager string `json:"manager"`
}

// SwarmInitRequest configures putting an engine into swarm mode as a manager.
type SwarmInitRequest struct {
	// AdvertiseAddr is the address managers/workers reach this manager on
	// (host or host:port). Required.
	AdvertiseAddr string
	// ListenAddr is the management-plane bind; blank defaults to 0.0.0.0:2377.
	ListenAddr string
	// DataPathAddr is the address for overlay (VXLAN) data-plane traffic; blank
	// falls back to AdvertiseAddr.
	DataPathAddr string
}

// SwarmJoinRequest configures joining an engine to an existing swarm.
type SwarmJoinRequest struct {
	// RemoteAddrs are the manager addresses to dial (host:port).
	RemoteAddrs []string
	// JoinToken is the worker or manager join token.
	JoinToken string
	// AdvertiseAddr is the address this node advertises to peers; blank lets the
	// engine auto-detect.
	AdvertiseAddr string
	// ListenAddr is the management-plane bind; blank defaults to 0.0.0.0:2377.
	ListenAddr string
}

// Swarm reports this engine's swarm participation, derived from `docker info`.
// It never requires a manager role, so it is safe to call on any node.
func (e *engineClient) Swarm(ctx context.Context) (SwarmInfo, error) {
	info, err := e.cli.Info(ctx)
	if err != nil {
		return SwarmInfo{}, err
	}
	s := info.Swarm
	out := SwarmInfo{
		LocalNodeState:   string(s.LocalNodeState),
		ControlAvailable: s.ControlAvailable,
		NodeID:           s.NodeID,
		NodeAddr:         s.NodeAddr,
		Managers:         s.Managers,
		Nodes:            s.Nodes,
		Error:            s.Error,
	}
	for _, rm := range s.RemoteManagers {
		if rm.Addr != "" {
			out.RemoteManagers = append(out.RemoteManagers, rm.Addr)
		}
	}
	return out, nil
}

// SwarmInit puts this engine into swarm mode as the first manager and returns
// its swarm node ID.
func (e *engineClient) SwarmInit(ctx context.Context, req SwarmInitRequest) (string, error) {
	listen := req.ListenAddr
	if listen == "" {
		listen = defaultSwarmListenAddr
	}
	return e.cli.SwarmInit(ctx, swarm.InitRequest{
		AdvertiseAddr: req.AdvertiseAddr,
		ListenAddr:    listen,
		DataPathAddr:  req.DataPathAddr,
	})
}

// SwarmJoin joins this engine to an existing swarm using the given token and
// manager remote address(es).
func (e *engineClient) SwarmJoin(ctx context.Context, req SwarmJoinRequest) error {
	listen := req.ListenAddr
	if listen == "" {
		listen = defaultSwarmListenAddr
	}
	return e.cli.SwarmJoin(ctx, swarm.JoinRequest{
		RemoteAddrs:   req.RemoteAddrs,
		JoinToken:     req.JoinToken,
		AdvertiseAddr: req.AdvertiseAddr,
		ListenAddr:    listen,
	})
}

// SwarmLeave removes this engine from its swarm. Leaving as the last manager
// requires force.
func (e *engineClient) SwarmLeave(ctx context.Context, force bool) error {
	return e.cli.SwarmLeave(ctx, force)
}

// SwarmJoinTokens returns the swarm's worker and manager join tokens. Requires
// this engine to be a reachable manager.
func (e *engineClient) SwarmJoinTokens(ctx context.Context) (SwarmJoinTokens, error) {
	sw, err := e.cli.SwarmInspect(ctx)
	if err != nil {
		return SwarmJoinTokens{}, err
	}
	return SwarmJoinTokens{Worker: sw.JoinTokens.Worker, Manager: sw.JoinTokens.Manager}, nil
}

// SwarmNodes lists the swarm's nodes. Requires this engine to be a reachable
// manager.
func (e *engineClient) SwarmNodes(ctx context.Context) ([]SwarmNode, error) {
	list, err := e.cli.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]SwarmNode, 0, len(list))
	for _, n := range list {
		sn := SwarmNode{
			ID:            n.ID,
			Hostname:      n.Description.Hostname,
			Role:          string(n.Spec.Role),
			Availability:  string(n.Spec.Availability),
			State:         string(n.Status.State),
			Addr:          n.Status.Addr,
			EngineVersion: n.Description.Engine.EngineVersion,
		}
		if n.ManagerStatus != nil {
			sn.Leader = n.ManagerStatus.Leader
			sn.Reachability = string(n.ManagerStatus.Reachability)
			if sn.Role == "" {
				sn.Role = string(swarm.NodeRoleManager)
			}
		}
		out = append(out, sn)
	}
	return out, nil
}

// SwarmNodeRemove removes a node from the swarm's node list on the manager
// (after the node has left). force is needed for nodes that have not gracefully
// left. Used to keep the Nodes page free of stale "down" entries.
func (e *engineClient) SwarmNodeRemove(ctx context.Context, nodeID string, force bool) error {
	return e.cli.NodeRemove(ctx, nodeID, types.NodeRemoveOptions{Force: force})
}
