import api from './client'
import type { ApiResponse, Environment } from './types'

export interface EnvironmentInput {
  name: string
  description?: string
  rank?: number
  required_approvals?: number
  git_source_id?: number | null
}

const base = (ws: number) => `/workspaces/${ws}/environments`

export const environmentApi = {
  list: (ws: number) => api.get<ApiResponse<Environment[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<Environment>>(`${base(ws)}/${id}`),
  create: (ws: number, input: EnvironmentInput) => api.post<ApiResponse<Environment>>(base(ws), input),
  update: (ws: number, id: number, input: EnvironmentInput) =>
    api.patch<ApiResponse<Environment>>(`${base(ws)}/${id}`, input),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
