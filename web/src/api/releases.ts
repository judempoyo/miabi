import api from './client'
import type { ApiResponse, PageableResponse, WorkspaceRelease, ReleaseApprovalStatus, Deployment } from './types'

export interface ApproveInput {
  environment_id?: number | null
  approved?: boolean
  comment?: string
}

const base = (ws: number) => `/workspaces/${ws}/releases`

export const releaseApi = {
  list: (ws: number, page = 0, size = 20) =>
    api.get<PageableResponse<WorkspaceRelease>>(`${base(ws)}?page=${page}&size=${size}`),
  approvals: (ws: number, id: number) =>
    api.get<ApiResponse<ReleaseApprovalStatus>>(`${base(ws)}/${id}/approvals`),
  approve: (ws: number, id: number, input: ApproveInput) =>
    api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/approve`, input),
  promote: (ws: number, id: number, environmentId: number) =>
    api.post<ApiResponse<Deployment>>(`${base(ws)}/${id}/promote`, { environment_id: environmentId }),
}
