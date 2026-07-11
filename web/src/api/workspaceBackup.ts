import api from './client'
import type { ApiResponse } from './types'

export interface BackupSettings {
  id: number
  workspace_id: number
  s3_enabled: boolean
  s3_endpoint?: string
  s3_bucket?: string
  s3_region?: string
  s3_access_key?: string
  s3_use_ssl: boolean
  s3_force_path_style: boolean
  database_backup_path?: string
  volume_backup_path?: string
  s3_secret_set: boolean
  created_at?: string
  updated_at?: string
}

// UpdateBackupSettingsInput mirrors the backend body. Leave s3_secret_key empty
// to keep the stored secret unchanged.
export interface UpdateBackupSettingsInput {
  s3_enabled: boolean
  s3_endpoint: string
  s3_bucket: string
  s3_region: string
  s3_access_key: string
  s3_secret_key: string
  s3_use_ssl: boolean
  s3_force_path_style: boolean
  database_backup_path: string
  volume_backup_path: string
}

export const workspaceBackupApi = {
  get(workspaceId: number) {
    return api.get<ApiResponse<BackupSettings>>(`/workspaces/${workspaceId}/backup-settings`)
  },
  update(workspaceId: number, input: UpdateBackupSettingsInput) {
    return api.put<ApiResponse<BackupSettings>>(`/workspaces/${workspaceId}/backup-settings`, input)
  },
  test(workspaceId: number, input: UpdateBackupSettingsInput) {
    return api.post<ApiResponse<{ message: string }>>(`/workspaces/${workspaceId}/backup-settings/test`, input)
  },
}
