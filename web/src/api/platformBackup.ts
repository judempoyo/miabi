import api from './client'
import type { ApiResponse, PageableResponse } from './types'

// PlatformBackupSubject mirrors the backend: the control-plane database or a
// platform/system volume.
export type PlatformBackupSubject = 'database' | 'volume'
export type PlatformBackupStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface PlatformBackup {
  id: number
  subject: PlatformBackupSubject
  volume_name?: string
  status: PlatformBackupStatus
  trigger: string
  destination: string
  s3_bucket?: string
  s3_path?: string
  filename?: string
  size_bytes: number
  logs?: string
  error?: string
  started_at?: string | null
  finished_at?: string | null
  created_at: string
}

export interface PlatformBackupSettings {
  id: number
  s3_enabled: boolean
  s3_endpoint?: string
  s3_bucket?: string
  s3_region?: string
  s3_access_key?: string
  s3_use_ssl: boolean
  s3_force_path_style: boolean
  database_backup_path?: string
  volume_backup_path?: string
  schedule_enabled: boolean
  schedule_cron?: string
  max_backups: number
  retention_days: number
  volumes: string[]
  created_at: string
  updated_at: string
  s3_secret_set: boolean
}

export interface PlatformBackupSettingsPayload {
  s3_enabled: boolean
  s3_endpoint: string
  s3_bucket: string
  s3_region: string
  s3_access_key: string
  s3_secret_key?: string
  s3_use_ssl: boolean
  s3_force_path_style: boolean
  database_backup_path: string
  volume_backup_path: string
  schedule_enabled: boolean
  schedule_cron: string
  max_backups: number
  retention_days: number
  volumes: string[]
}

export interface PlatformVolume {
  name: string
  role?: string
}

export const platformBackupApi = {
  getSettings: () => api.get<ApiResponse<PlatformBackupSettings>>('/admin/platform-backup/settings'),
  updateSettings: (payload: PlatformBackupSettingsPayload) =>
    api.put<ApiResponse<PlatformBackupSettings>>('/admin/platform-backup/settings', payload),
  testSettings: (payload: PlatformBackupSettingsPayload) =>
    api.post<ApiResponse<{ message: string }>>('/admin/platform-backup/settings/test', payload),

  list: (page = 0, size = 20) =>
    api.get<PageableResponse<PlatformBackup>>('/admin/platform-backup/backups', { params: { page, size } }),
  create: (payload: { database: boolean; volumes: string[] }) =>
    api.post<ApiResponse<PlatformBackup[]>>('/admin/platform-backup/backups', payload),
  restore: (id: number) =>
    api.post<ApiResponse<{ message: string }>>(`/admin/platform-backup/backups/${id}/restore`, { confirm: true }),
  remove: (id: number) =>
    api.delete<ApiResponse<{ message: string }>>(`/admin/platform-backup/backups/${id}`),
  download: (id: number) =>
    api.get<Blob>(`/admin/platform-backup/backups/${id}/download`, { responseType: 'blob' }),

  volumes: () => api.get<ApiResponse<PlatformVolume[]>>('/admin/platform-backup/volumes'),
}
