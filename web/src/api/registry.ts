import api from './client'
import type { ApiResponse } from './types'

// RegistrySettings is the platform's built-in Docker registry config (admin).
export interface RegistrySettings {
  id: number
  enabled: boolean
  host?: string
  storage_type: 'filesystem' | 's3'
  volume_name?: string
  s3_endpoint?: string
  s3_bucket?: string
  s3_region?: string
  s3_access_key?: string
  s3_force_path_style: boolean
  delete_enabled: boolean
  per_workspace_quota_mb: number
  s3_secret_set: boolean
  effective_host: string
  /** Whether S3/MinIO storage is licensed (Enterprise); local storage is free. */
  s3_entitled: boolean
}

export interface RegistrySettingsPayload {
  enabled: boolean
  host: string
  storage_type: 'filesystem' | 's3'
  s3_endpoint: string
  s3_bucket: string
  s3_region: string
  s3_access_key: string
  s3_secret_key?: string
  s3_force_path_style: boolean
  delete_enabled: boolean
  per_workspace_quota_mb: number
}

// RegistryInfo is the per-workspace docker-login guidance.
export interface RegistryInfo {
  enabled: boolean
  host: string
  namespace: string
  image_prefix: string
  login_example: string
}

export interface RegistryRepository {
  name: string
  tags: string[]
}

export const registryApi = {
  // Admin
  getSettings: () => api.get<ApiResponse<RegistrySettings>>('/admin/registry/settings'),
  updateSettings: (payload: RegistrySettingsPayload) =>
    api.put<ApiResponse<RegistrySettings>>('/admin/registry/settings', payload),
  runGc: () => api.post<ApiResponse<{ message: string }>>('/admin/registry/gc'),

  // Workspace
  info: (workspaceId: number) =>
    api.get<ApiResponse<RegistryInfo>>(`/workspaces/${workspaceId}/registry`),
  repositories: (workspaceId: number) =>
    api.get<ApiResponse<RegistryRepository[]>>(`/workspaces/${workspaceId}/registry/repositories`),
  deleteTag: (workspaceId: number, repo: string, tag: string) =>
    api.delete<ApiResponse<{ message: string }>>(
      `/workspaces/${workspaceId}/registry/repositories/${encodeURIComponent(repo)}/tags/${encodeURIComponent(tag)}`,
    ),
}
