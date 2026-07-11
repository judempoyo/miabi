import api from './client'
import type { ApiResponse, Webhook, WebhookDelivery } from './types'

export interface WebhookInput {
  name?: string
  url: string
  events: string[]
  headers?: Record<string, string>
  enabled: boolean
}

const base = (ws: number) => `/workspaces/${ws}/webhooks`

export const webhookApi = {
  list: (ws: number) => api.get<ApiResponse<Webhook[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<Webhook>>(`${base(ws)}/${id}`),
  create: (ws: number, input: WebhookInput) => api.post<ApiResponse<Webhook>>(base(ws), input),
  update: (ws: number, id: number, input: WebhookInput) =>
    api.patch<ApiResponse<Webhook>>(`${base(ws)}/${id}`, input),
  test: (ws: number, id: number) =>
    api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/test`),
  remove: (ws: number, id: number) =>
    api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
  deliveries: (ws: number, id: number) =>
    api.get<ApiResponse<WebhookDelivery[]>>(`${base(ws)}/${id}/deliveries`),
  redeliver: (ws: number, id: number, deliveryId: number) =>
    api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/deliveries/${deliveryId}/redeliver`),
}
