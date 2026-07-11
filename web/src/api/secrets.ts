import api from './client'
import type { ApiResponse, PageableResponse, Secret } from './types'

export interface SecretInput {
  name?: string
  value?: string
  description?: string
}

const base = (ws: number) => `/workspaces/${ws}/secrets`

export const secretApi = {
  list: (ws: number, search = '', page = 0, size = 20) =>
    api.get<PageableResponse<Secret>>(base(ws), { params: { search: search || undefined, page, size } }),
  create: (ws: number, input: SecretInput) => api.post<ApiResponse<Secret>>(base(ws), input),
  update: (ws: number, id: number, input: SecretInput) => api.put<ApiResponse<Secret>>(`${base(ws)}/${id}`, input),
  reveal: (ws: number, id: number) => api.get<ApiResponse<{ value: string }>>(`${base(ws)}/${id}/reveal`),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
