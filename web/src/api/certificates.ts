import api from './client'
import type { ApiResponse, Certificate, IssueCertificateInput } from './types'

export interface CertificateInput {
  name?: string
  cert_pem: string
  key_pem: string
}

const base = (ws: number) => `/workspaces/${ws}/certificates`

export const certificateApi = {
  list: (ws: number) => api.get<ApiResponse<Certificate[]>>(base(ws)),
  // Certificates whose SANs cover the given host (route-form auto-select).
  matchHost: (ws: number, host: string) =>
    api.get<ApiResponse<Certificate[]>>(`${base(ws)}?host=${encodeURIComponent(host)}`),
  get: (ws: number, id: number) => api.get<ApiResponse<Certificate>>(`${base(ws)}/${id}`),
  usage: (ws: number, id: number) =>
    api.get<ApiResponse<{ id: number; name: string }[]>>(`${base(ws)}/${id}/usage`),
  import: (ws: number, input: CertificateInput) => api.post<ApiResponse<Certificate>>(base(ws), input),
  // issue starts a managed (ACME DNS-01) certificate; returns the issuing row.
  issue: (ws: number, input: IssueCertificateInput) =>
    api.post<ApiResponse<Certificate>>(`${base(ws)}/issue`, input),
  replace: (ws: number, id: number, input: CertificateInput) =>
    api.put<ApiResponse<Certificate>>(`${base(ws)}/${id}`, input),
  remove: (ws: number, id: number) => api.delete<ApiResponse<{ message: string }>>(`${base(ws)}/${id}`),
}
