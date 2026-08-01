// Time-axis helpers for the analytics charts.
//
// The report's granularity (minute | hour | day) decides both how a bucket is
// labelled and which tick spacings are legible: an hourly series wants "14:00",
// a daily one wants "Aug 2". Ticks land on round wall-clock boundaries (:00,
// :15, …) rather than every Nth bucket, so the labels read like a clock instead
// of like array indices.
//
// Sub-daily buckets render in the viewer's own zone. Daily buckets are cut on
// UTC midnight server-side, so they are labelled in UTC too — labelling a UTC
// day with a local date shifts it by one for anyone west of Greenwich.

import type { AnalyticsReport } from '@/api/analytics'

export type Granularity = AnalyticsReport['granularity']

// Candidate tick spacings, in buckets, coarsest-last. Every candidate divides
// the day evenly so ticks always land on the same wall-clock marks.
const STEPS: Record<Granularity, number[]> = {
  minute: [1, 2, 5, 10, 15, 30, 60, 120, 180],
  hour: [1, 2, 3, 4, 6, 12, 24],
  day: [1, 2, 5, 7, 14, 30],
}

// Roughly how much room one tick label needs, used to pick the spacing. Sized
// for the widest label a locale produces ("11:45 PM").
const TICK_PX = 76

const DAY_MS = 86_400_000

// aligned reports whether a bucket start falls on a `step`-sized boundary.
// Hour and day boundaries are counted from the epoch rather than matched against
// wall-clock components: buckets are cut in UTC, so in a half-hour-offset zone
// (India, Nepal, Newfoundland) no hourly bucket ever lands on a local :00, and
// requiring one would leave the axis with no ticks at all.
function aligned(d: Date, gran: Granularity, step: number): boolean {
  if (gran === 'day') return Math.floor(d.getTime() / DAY_MS) % step === 0
  if (gran === 'hour') {
    const hours = Math.round((d.getTime() - d.getTimezoneOffset() * 60_000) / 3_600_000)
    return hours % step === 0
  }
  return (d.getHours() * 60 + d.getMinutes()) % step === 0
}

export interface Tick {
  i: number // index into the series
  text: string
  major: boolean // carries a date rather than a time
}

// dayKey identifies the local calendar day a bucket falls in — the axis switches
// from a time label to a date label on the first tick of each new day.
function dayKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

export function timeLabel(d: Date): string {
  // `numeric` hours keep 12-hour locales short ("4:10 PM"); 24-hour locales still
  // render zero-padded ("16:10").
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
}

export function dateLabel(d: Date, withYear = false, utc = false): string {
  return d.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    ...(withYear ? { year: 'numeric' } : {}),
    ...(utc ? { timeZone: 'UTC' } : {}),
  })
}

// buildTicks picks a spacing that fits `width` and returns the labelled ticks.
// A daily axis is all dates; a sub-daily one is times, except that the first
// tick of each day carries the date instead — so a window spanning midnight
// always says which day each side of it belongs to.
export function buildTicks(times: Date[], gran: Granularity, width: number): Tick[] {
  if (!times.length) return []
  const maxTicks = Math.max(2, Math.floor(width / TICK_PX))
  const steps = STEPS[gran]

  let step = steps[steps.length - 1]
  for (const s of steps) {
    if (times.filter((d) => aligned(d, gran, s)).length <= maxTicks) {
      step = s
      break
    }
  }

  let idx = times.map((_, i) => i).filter((i) => aligned(times[i], gran, step))
  // A window too short to contain a single boundary (e.g. 30 min at a 60-min
  // spacing) still gets its ends labelled — an unlabelled axis is worse.
  if (idx.length < 2 && times.length > 1) idx = [0, times.length - 1]

  const last = times[times.length - 1]
  if (gran === 'day') {
    const withYear = times[0].getUTCFullYear() !== new Date().getUTCFullYear()
    return idx.map((i) => ({
      i,
      text: dateLabel(times[i], withYear, true),
      major: times[i].getUTCDate() === 1,
    }))
  }

  const multiDay = dayKey(times[0]) !== dayKey(last)
  let prevDay = ''
  return idx.map((i) => {
    const d = times[i]
    const newDay = dayKey(d) !== prevDay
    prevDay = dayKey(d)
    const asDate = multiDay && newDay
    return { i, text: asDate ? dateLabel(d) : timeLabel(d), major: asDate }
  })
}

// bucketLabel is the full, unambiguous name of one bucket, for tooltips: a date
// plus the span the bucket covers.
export function bucketLabel(d: Date, gran: Granularity): string {
  const utc = gran === 'day'
  const withYear = (utc ? d.getUTCFullYear() : d.getFullYear()) !== new Date().getFullYear()
  const day = d.toLocaleDateString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    ...(withYear ? { year: 'numeric' } : {}),
    ...(utc ? { timeZone: 'UTC' } : {}),
  })
  if (gran === 'day') return `${day} (UTC)`
  if (gran === 'hour') {
    const end = new Date(d.getTime() + 3_600_000)
    return `${day} · ${timeLabel(d)} – ${timeLabel(end)}`
  }
  return `${day} · ${timeLabel(d)}`
}

// windowLabel spells out the absolute window behind a relative range ("24h"),
// collapsing the date when both ends fall on the same day.
export function windowLabel(since: string, until: string): string {
  const a = new Date(since)
  const b = new Date(until)
  if (isNaN(a.getTime()) || isNaN(b.getTime())) return ''
  const sameDay = a.toDateString() === b.toDateString()
  const withYear = a.getFullYear() !== new Date().getFullYear()
  const left = `${dateLabel(a, withYear)}, ${timeLabel(a)}`
  const right = sameDay ? timeLabel(b) : `${dateLabel(b, withYear)}, ${timeLabel(b)}`
  return `${left} → ${right}`
}
