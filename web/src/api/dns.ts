import api from './client'
import type { ApiResponse, DNSProvider, ConnectDNSProviderInput } from './types'

const base = (ws: number) => `/workspaces/${ws}/dns/providers`

export const dnsProviderApi = {
  list: (ws: number) => api.get<ApiResponse<DNSProvider[]>>(base(ws)),
  get: (ws: number, id: number) => api.get<ApiResponse<DNSProvider>>(`${base(ws)}/${id}`),
  connect: (ws: number, input: ConnectDNSProviderInput) =>
    api.post<ApiResponse<DNSProvider>>(base(ws), input),
  test: (ws: number, id: number, zone: string) =>
    api.post<ApiResponse<DNSProvider>>(`${base(ws)}/${id}/test`, { zone }),
  remove: (ws: number, id: number) =>
    api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
