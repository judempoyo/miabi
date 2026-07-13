import api from './client'
import type { ApiResponse, ClusterStatus, ClusterJoinInstructions, ClusterMember } from './types'

// clusterApi drives Miabi's optional cluster mode (Docker Swarm under the hood).
// Status is always available; the mutations require platform-admin rights and
// only take effect on a swarm-capable manager.
export const clusterApi = {
  status: () => api.get<ApiResponse<ClusterStatus>>('/admin/cluster'),
  // Swarm membership (docker node ls), including unmanaged members.
  members: () => api.get<ApiResponse<ClusterMember[]>>('/admin/cluster/nodes'),
  // Manual join command/token for hosts not connected to the manager.
  joinToken: () => api.get<ApiResponse<ClusterJoinInstructions>>('/admin/cluster/join-token'),
  // advertiseAddr is the address swarm peers reach the manager on; required when
  // initializing a new swarm, ignored when adopting an existing one.
  enable: (advertiseAddr: string) =>
    api.post<ApiResponse<ClusterStatus>>('/admin/cluster/enable', { advertise_addr: advertiseAddr }),
  disable: () => api.post<ApiResponse<{ message: string }>>('/admin/cluster/disable'),
  // Convert workspace networks still on node-local bridges into cluster overlays,
  // so apps and databases reach each other across nodes. Enable already does this
  // on the transition into cluster mode; this is the explicit action for an install
  // that was already clustered when it upgraded, or to re-run it after a node that
  // was offline comes back. Containers are not restarted, but in-flight connections
  // inside each workspace drop briefly.
  applyNetworking: () => api.post<ApiResponse<ClusterStatus>>('/admin/cluster/network/apply'),
  joinNode: (nodeId: number) =>
    api.post<ApiResponse<{ message: string }>>(`/admin/cluster/nodes/${nodeId}/join`),
  leaveNode: (nodeId: number) =>
    api.post<ApiResponse<{ message: string }>>(`/admin/cluster/nodes/${nodeId}/leave`),
}
