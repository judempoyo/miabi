import api from './client'
import type { ApiResponse, AppInfo } from './types'

export const infoApi = {
  get: () => api.get<ApiResponse<AppInfo>>('/info'),
}
