import api from './client'
import type { ApiResponse, Registry } from './types'

export interface RegistryInput {
  name: string
  server?: string
  username?: string
  secret?: string
}

const base = (ws: number) => `/workspaces/${ws}/registries`

export const registryApi = {
  list: (ws: number) => api.get<ApiResponse<Registry[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<Registry>>(`${base(ws)}/${id}`),
  create: (ws: number, input: RegistryInput) => api.post<ApiResponse<Registry>>(base(ws), input),
  update: (ws: number, id: number, input: RegistryInput) => api.patch<ApiResponse<Registry>>(`${base(ws)}/${id}`, input),
  test: (ws: number, id: number) => api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/test`),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
