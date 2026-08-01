import { defineStore } from 'pinia'
import { ref } from 'vue'
import { analyticsApi, type AnalyticsReport } from '@/api/analytics'
import { appApi } from '@/api/apps'
import { useWorkspaceStore } from './workspace'

// Range options shared by every analytics page (Cloudflare-style windows).
export const ANALYTICS_RANGES = [
  { key: '30m', label: 'Last 30 min' },
  { key: '1h', label: 'Last hour' },
  { key: '24h', label: 'Last 24 hours' },
  { key: '7d', label: 'Last 7 days' },
  { key: '30d', label: 'Last 30 days' },
] as const

// Shared analytics state: range + app filter drive one report used by all four
// sub-pages (Overview, HTTP Traffic, Performance, Web Analytics), so switching
// pages doesn't refetch. A key guard skips redundant loads.
export const useAnalyticsStore = defineStore('analytics', () => {
  const range = ref<string>('24h')
  const appFilter = ref<number | null>(null)

  const report = ref<AnalyticsReport | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const appIds = ref<number[]>([])
  const appNames = ref<Record<number, string>>({})

  // Live visitors is a "right now" number, so it polls on its own cadence
  // instead of riding the report load. null = not loaded yet (the pill hides).
  const live = ref<number | null>(null)
  const liveWindowSeconds = ref(300)
  let liveTimer: ReturnType<typeof setInterval> | null = null
  let liveWatchers = 0

  let lastKey = ''

  function key(ws: number) {
    return `${ws}|${range.value}|${appFilter.value ?? 0}`
  }

  async function loadApps(ws: number) {
    try {
      const [all, withData] = await Promise.all([
        appApi.list(ws),
        analyticsApi.apps(ws, range.value),
      ])
      const names: Record<number, string> = {}
      for (const a of all.data.data ?? []) names[a.id] = a.display_name || a.name
      appNames.value = names
      appIds.value = withData.data.data?.application_ids ?? []
      if (appFilter.value && !appIds.value.includes(appFilter.value)) appFilter.value = null
    } catch {
      // Non-fatal — the report still renders workspace-wide.
    }
  }

  // load fetches the report for the current (workspace, range, app). force
  // bypasses the key guard (e.g. a manual refresh).
  async function load(force = false) {
    const ws = useWorkspaceStore().currentWorkspaceId
    if (!ws) {
      report.value = null
      return
    }
    const k = key(ws)
    if (!force && k === lastKey && report.value) return
    lastKey = k
    loading.value = true
    error.value = null
    try {
      const res = await analyticsApi.report(ws, range.value, appFilter.value ?? undefined)
      report.value = res.data.data
      await loadApps(ws)
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: { message?: string } } } }
      error.value = err.response?.data?.error?.message || 'Failed to load analytics'
      report.value = null
    } finally {
      loading.value = false
    }
  }

  // LIVE_POLL_MS is the refresh cadence of the live pill. The number is a count
  // over a multi-minute window, so polling faster only adds requests.
  const LIVE_POLL_MS = 10_000

  async function loadLive() {
    const ws = useWorkspaceStore().currentWorkspaceId
    if (!ws) return
    try {
      const res = await analyticsApi.live(ws, appFilter.value ?? undefined)
      live.value = res.data.data?.visitors ?? 0
      if (res.data.data?.window_seconds) liveWindowSeconds.value = res.data.data.window_seconds
    } catch {
      // Best-effort: a failed poll leaves the last known value in place rather
      // than flashing the pill away.
    }
  }

  // watchLive starts polling for the first watcher and stops when the last one
  // leaves, so a backgrounded dashboard isn't polling forever. Returns the
  // matching stop function. Polling pauses while the tab is hidden.
  function watchLive(): () => void {
    liveWatchers++
    if (liveWatchers === 1) {
      loadLive()
      liveTimer = setInterval(() => {
        if (!document.hidden) loadLive()
      }, LIVE_POLL_MS)
      document.addEventListener('visibilitychange', onVisible)
    }
    let stopped = false
    return () => {
      if (stopped) return
      stopped = true
      liveWatchers--
      if (liveWatchers === 0) {
        if (liveTimer) clearInterval(liveTimer)
        liveTimer = null
        document.removeEventListener('visibilitychange', onVisible)
      }
    }
  }

  // Refresh straight away when the tab comes back, so the pill isn't showing a
  // number from before the user switched away.
  function onVisible() {
    if (!document.hidden) loadLive()
  }

  function setRange(r: string) {
    if (r === range.value) return
    range.value = r
    load()
  }

  function setApp(id: number | null) {
    if (id === appFilter.value) return
    appFilter.value = id
    load()
    loadLive() // the pill is app-scoped too
  }

  return {
    range, appFilter, report, loading, error, appIds, appNames,
    live, liveWindowSeconds,
    load, setRange, setApp, loadLive, watchLive,
  }
})
