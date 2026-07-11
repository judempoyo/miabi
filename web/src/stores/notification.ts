import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiError as decodeApiError } from '@/api/client'

type NotificationType = 'success' | 'error' | 'info'

interface Notification {
  id: number
  title: string
  message: string
  // detail is an optional secondary line shown muted under the message (e.g. the
  // envelope's raw `error` when it adds detail beyond `message`).
  detail?: string
  type: NotificationType
}

// NotifyOptions configures a toast beyond its message. Passing a bare string as
// the options arg is shorthand for { title } (back-compat with the old signature).
interface NotifyOptions {
  title?: string
  detail?: string
}

// Default heading shown above the message for each toast type.
const DEFAULT_TITLES: Record<NotificationType, string> = {
  success: 'Success',
  error: 'Error',
  info: 'Info',
}

// normalizeOpts accepts the legacy `title` string or a NotifyOptions object.
function normalizeOpts(opts?: string | NotifyOptions): NotifyOptions {
  return typeof opts === 'string' ? { title: opts } : (opts ?? {})
}

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<Notification[]>([])
  let nextId = 0

  function show(message: string, type: NotificationType = 'info', duration = 4000, opts?: string | NotifyOptions) {
    const { title, detail } = normalizeOpts(opts)
    const id = nextId++
    notifications.value.push({ id, title: title ?? DEFAULT_TITLES[type], message, detail, type })
    if (duration > 0) {
      setTimeout(() => dismiss(id), duration)
    }
  }

  function success(message: string, opts?: string | NotifyOptions) { show(message, 'success', 4000, opts) }
  function error(message: string, opts?: string | NotifyOptions) { show(message, 'error', 6000, opts) }
  function info(message: string, opts?: string | NotifyOptions) { show(message, 'info', 4000, opts) }

  // apiError shows an error toast for a rejected request: a status-derived title,
  // the envelope's human `message`, and — when it adds detail — the raw `error` as
  // a muted secondary line.
  function apiError(err: unknown, fallback?: string) {
    const { title, message, detail } = decodeApiError(err, fallback)
    error(message, { title, detail })
  }

  function dismiss(id: number) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  return { notifications, show, success, error, info, apiError, dismiss }
})
