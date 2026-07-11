import api from './client'
import type { ApiResponse, NotificationChannel } from './types'

export interface ChannelInput {
  type?: 'telegram' | 'slack' | 'discord'
  name?: string
  bot_token?: string
  chat_id?: string
  webhook_url?: string
  events: string[]
  enabled: boolean
}

const base = (ws: number) => `/workspaces/${ws}/notifications/channels`

export const channelApi = {
  list: (ws: number) => api.get<ApiResponse<NotificationChannel[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<NotificationChannel>>(`${base(ws)}/${id}`),
  create: (ws: number, input: ChannelInput) =>
    api.post<ApiResponse<NotificationChannel>>(base(ws), input),
  update: (ws: number, id: number, input: ChannelInput) =>
    api.patch<ApiResponse<NotificationChannel>>(`${base(ws)}/${id}`, input),
  test: (ws: number, id: number) =>
    api.post<ApiResponse<{ message: string }>>(`${base(ws)}/${id}/test`),
  remove: (ws: number, id: number) =>
    api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
