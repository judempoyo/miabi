import api from './client'
import type { ApiResponse, Network } from './types'

export interface NetworkInput {
  name: string
  driver?: string
  internal?: boolean
}

const base = (ws: number) => `/workspaces/${ws}/networks`

export const networkApi = {
  list: (ws: number) => api.get<ApiResponse<Network[]>>(base(ws)),
  create: (ws: number, input: NetworkInput) => api.post<ApiResponse<Network>>(base(ws), input),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
