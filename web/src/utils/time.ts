// Small, dependency-free time formatters shared across views.

// relativeTime renders an ISO timestamp as a short "2h ago" / "in 3m" string.
// Pass a `now` (e.g. a ticking ref value) to make it update live. Falls back to
// a locale date for anything older than a month, and '—' for empty/invalid input.
export function relativeTime(iso?: string | null, now: number = Date.now()): string {
  if (!iso) return '—'
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return '—'
  const diff = Math.round((now - t) / 1000) // seconds in the past (positive)
  const abs = Math.abs(diff)
  const past = diff >= 0
  if (abs < 45) return past ? 'just now' : 'soon'
  if (abs < 90) return past ? '1 min ago' : 'in 1 min'
  const mins = Math.round(abs / 60)
  if (mins < 60) return past ? `${mins} min ago` : `in ${mins} min`
  const hrs = Math.round(abs / 3600)
  if (hrs < 24) return past ? `${hrs}h ago` : `in ${hrs}h`
  const days = Math.round(abs / 86400)
  if (days < 30) return past ? `${days}d ago` : `in ${days}d`
  return new Date(iso).toLocaleDateString()
}

// formatDuration renders the span between two timestamps as "45s" / "1m 5s" /
// "1h 3m". When `finished` is missing it measures against `now` — pass a ticking
// value to show a live, counting-up duration for in-progress work.
export function formatDuration(
  started?: string | null,
  finished?: string | null,
  now: number = Date.now(),
): string {
  if (!started) return '—'
  const start = new Date(started).getTime()
  if (Number.isNaN(start)) return '—'
  const end = finished ? new Date(finished).getTime() : now
  const secs = Math.max(0, Math.round((end - start) / 1000))
  if (secs < 60) return `${secs}s`
  const m = Math.floor(secs / 60)
  const s = secs % 60
  if (m < 60) return `${m}m ${s}s`
  const h = Math.floor(m / 60)
  return `${h}h ${m % 60}m`
}
