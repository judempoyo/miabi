import api from './client'
import type { ApiResponse, Image } from './types'

const base = (ws: number) => `/workspaces/${ws}/images`

export const imageApi = {
  list: (ws: number, appId?: number) =>
    api.get<ApiResponse<Image[]>>(appId ? `${base(ws)}?app=${appId}` : base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<Image>>(`${base(ws)}/${id}`),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
