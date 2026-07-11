<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { adminApi } from '@/api/admin'
import type { JobStatus, JobStats } from '@/api/types'
import { useNotificationStore } from '@/stores/notification'
import { usePagination } from '@/composables/usePagination'
import Pagination from '@/components/Pagination.vue'

const notify = useNotificationStore()

const jobs = ref<JobStatus[]>([])
const stats = ref<JobStats | null>(null)
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const { pageable, goToPage } = usePagination(async (page) => {
  loading.value = true
  try {
    const [list, summary] = await Promise.all([
      adminApi.listJobs(page, pageable.value.size),
      adminApi.jobStats(),
    ])
    jobs.value = list.data.data ?? []
    pageable.value = list.data.pageable
    stats.value = summary.data.data
  } catch (err) {
    notify.apiError(err, 'Failed to load jobs')
  } finally {
    loading.value = false
  }
})

function reload() {
  goToPage(pageable.value.current_page)
}

// byKindLabel renders the per-kind breakdown as "backup: 2 · cronjob: 5".
function byKindLabel(byKind: Record<string, number>): string {
  const entries = Object.entries(byKind)
  if (entries.length === 0) return 'no jobs'
  return entries.map(([kind, n]) => `${kind}: ${n}`).join(' · ')
}

function fmtDate(s?: string | null, fallback = 'Never'): string {
  if (!s) return fallback
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return fallback
  return d.toLocaleString()
}

function humanCron(expr: string): string {
  const e = (expr || '').trim()
  if (!e) return ''
  const parts = e.split(/\s+/)
  if (parts.length === 5) {
    const [min, hour, dom, mon, dow] = parts
    const every = (f: string) => f === '*'

    // */N * * * *  => Every N minutes
    const stepMin = /^\*\/(\d+)$/.exec(min)
    if (stepMin && every(hour) && every(dom) && every(mon) && every(dow)) {
      const n = Number(stepMin[1])
      return `Every ${n} minute${n === 1 ? '' : 's'}`
    }

    // 0 */N * * *  => Every N hours
    const stepHour = /^\*\/(\d+)$/.exec(hour)
    if (min === '0' && stepHour && every(dom) && every(mon) && every(dow)) {
      const n = Number(stepHour[1])
      return `Every ${n} hour${n === 1 ? '' : 's'}`
    }

    // M H * * *  => Daily at HH:MM
    const numMin = /^\d+$/.test(min)
    const numHour = /^\d+$/.test(hour)
    if (numMin && numHour && every(dom) && every(mon) && every(dow)) {
      const hh = String(Number(hour)).padStart(2, '0')
      const mm = String(Number(min)).padStart(2, '0')
      return `Daily at ${hh}:${mm}`
    }

    // M H * * D  => Weekly on <day> at HH:MM
    if (numMin && numHour && every(dom) && every(mon) && /^\d+$/.test(dow)) {
      const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
      const day = days[Number(dow) % 7]
      const hh = String(Number(hour)).padStart(2, '0')
      const mm = String(Number(min)).padStart(2, '0')
      return `Weekly on ${day} at ${hh}:${mm}`
    }

    // Every minute
    if (every(min) && every(hour) && every(dom) && every(mon) && every(dow)) {
      return 'Every minute'
    }

    // M * * * *  => Hourly at minute M
    if (numMin && every(hour) && every(dom) && every(mon) && every(dow)) {
      return `Hourly at minute ${Number(min)}`
    }
  }
  return expr
}

// usePagination loads on mount; refresh the current page periodically.
timer = setInterval(reload, 30000)

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Jobs</h1>
      <button class="btn btn-secondary" :disabled="loading" @click="reload">
        <span class="mdi mdi-refresh"></span>
        Refresh
      </button>
    </div>

    <!-- Dashboard summary (computed across all jobs, not just this page). -->
    <div v-if="stats" class="stats-grid stats-compact">
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Total jobs</span>
          <span class="stat-icon stat-icon-primary"><span class="mdi mdi-clock-outline"></span></span>
        </div>
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-sub">{{ byKindLabel(stats.by_kind) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Running</span>
          <span class="stat-icon stat-icon-info"><span class="mdi mdi-play-circle-outline"></span></span>
        </div>
        <div class="stat-value">{{ stats.running }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Healthy</span>
          <span class="stat-icon stat-icon-success"><span class="mdi mdi-check-circle-outline"></span></span>
        </div>
        <div class="stat-value">{{ stats.ok }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-label">Failed</span>
          <span class="stat-icon" :class="stats.failed > 0 ? 'stat-icon-danger' : 'stat-icon-success'"><span class="mdi mdi-alert-circle-outline"></span></span>
        </div>
        <div class="stat-value">{{ stats.failed }}</div>
      </div>
    </div>

    <div v-if="loading && jobs.length === 0" class="card">
      <div class="card-body" style="display: flex; justify-content: center; padding: 48px 0">
        <span class="spinner"></span>
      </div>
    </div>

    <div v-else-if="jobs.length === 0" class="empty-state">
      <span class="mdi mdi-clock-outline" style="font-size: 44px; color: var(--text-muted)"></span>
      <h3>No scheduled jobs</h3>
      <p class="text-muted">
        Scheduled background jobs, like database backups, appear here once they are configured.
      </p>
    </div>

    <div v-else class="jobs-grid">
      <div v-for="job in jobs" :key="`${job.kind}-${job.id}`" class="card">
        <div class="card-body">
          <div class="job-head">
            <span class="job-name">{{ job.name }}</span>
            <span v-if="job.running" class="badge badge-dot badge-warning">Running</span>
            <span v-else-if="job.last_error" class="badge badge-dot badge-danger">Failed</span>
            <span v-else class="badge badge-dot badge-success">OK</span>
          </div>

          <div class="job-schedule mt-4">
            <span class="cron-chip" :title="job.schedule">{{ job.schedule }}</span>
            <span v-if="humanCron(job.schedule) !== job.schedule" class="text-muted schedule-human">
              {{ humanCron(job.schedule) }}
            </span>
          </div>

          <div class="job-meta mt-4">
            <span class="text-muted">Last run</span>
            <span>{{ fmtDate(job.last_run_at, 'Never') }}</span>
          </div>
          <div class="job-meta">
            <span class="text-muted">Next run</span>
            <span>{{ fmtDate(job.next_run_at, '—') }}</span>
          </div>

          <div v-if="job.last_error" class="job-error" :title="job.last_error">
            <span class="mdi mdi-alert-circle-outline"></span>
            {{ job.last_error }}
          </div>
        </div>
      </div>
    </div>

    <Pagination :pageable="pageable" @page="goToPage" />
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* Compact summary cards — smaller than the global stat-card. */
.stats-compact {
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}
.stats-compact :deep(.stat-card) {
  padding: 12px 14px;
  border-radius: var(--radius);
}
.stats-compact :deep(.stat-card:hover) {
  transform: none;
  box-shadow: var(--shadow-sm);
}
.stats-compact :deep(.stat-header) {
  margin-bottom: 4px;
}
.stats-compact :deep(.stat-value) {
  font-size: 20px;
}
.stats-compact :deep(.stat-icon) {
  width: 26px;
  height: 26px;
  font-size: 15px;
}

.jobs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.job-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.job-name {
  font-weight: 600;
  color: var(--text-primary);
}

.job-schedule {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.cron-chip {
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  background: var(--bg-tertiary);
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--text-primary);
}

.schedule-human {
  font-size: 12px;
}

.job-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  margin-top: 6px;
}

.job-meta span:last-child {
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.job-error {
  margin-top: 10px;
  font-size: 12px;
  color: var(--danger-500, #ef4444);
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.job-error .mdi {
  flex-shrink: 0;
}
</style>
