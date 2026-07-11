import api from './client'
import type { ApiResponse, Middleware, MiddlewareCatalog } from './types'

export interface MiddlewareInput {
  name: string
  type: string
  paths?: string[]
  rule?: Record<string, unknown>
}

const base = (ws: number) => `/workspaces/${ws}/middlewares`

export const middlewareApi = {
  list: (ws: number) => api.get<ApiResponse<Middleware[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<Middleware>>(`${base(ws)}/${id}`),
  create: (ws: number, input: MiddlewareInput) => api.post<ApiResponse<Middleware>>(base(ws), input),
  update: (ws: number, id: number, input: MiddlewareInput) => api.patch<ApiResponse<Middleware>>(`${base(ws)}/${id}`, input),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
  catalog: (ws: number) => api.get<ApiResponse<MiddlewareCatalog>>(`/workspaces/${ws}/middleware-catalog`),
}
