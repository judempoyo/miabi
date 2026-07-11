import api from './client'
import type { ApiResponse, GitRepository, GitAuthType } from './types'

export interface GitRepositoryInput {
  name: string
  url: string
  auth_type?: GitAuthType
  username?: string
  secret?: string
}

const base = (ws: number) => `/workspaces/${ws}/git-repositories`

export const gitRepositoryApi = {
  list: (ws: number) => api.get<ApiResponse<GitRepository[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<GitRepository>>(`${base(ws)}/${id}`),
  create: (ws: number, input: GitRepositoryInput) => api.post<ApiResponse<GitRepository>>(base(ws), input),
  update: (ws: number, id: number, input: GitRepositoryInput) => api.patch<ApiResponse<GitRepository>>(`${base(ws)}/${id}`, input),
  test: (ws: number, id: number) => api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/test`),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
