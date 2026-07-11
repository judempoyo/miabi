<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'
import type { AdminEvent, PlatformMetrics } from '@/api/types'
import { sseUrl } from '@/api/client'
import { useNotificationStore } from '@/stores/notification'

const notify = useNotificationStore()
const router = useRouter()

const metrics = ref<PlatformMetrics | null>(null)
const events = ref<AdminEvent[]>([])
let es: EventSource | null = null

function fmtBytes(n: number): string {
  if (!n || n < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = n
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  const rounded = value >= 100 || i === 0 ? Math.round(value) : Math.round(value * 10) / 10
  return `${rounded} ${units[i]}`
}

function fmtUptime(seconds: number): string {
  if (!seconds || seconds < 0) return '0m'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d > 0) parts.push(`${d}d`)
  if (h > 0) parts.push(`${h}h`)
  parts.push(`${m}m`)
  return parts.join(' ')
}

function relTime(ts: string): string {
  const then = new Date(ts).getTime()
  if (Number.isNaN(then)) return ''
  const diff = Math.floor((Date.now() - then) / 1000)
  if (diff < 60) return `${Math.max(0, diff)}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

// Turn an audit action like "admin.user.update" into "User update".
function prettyAction(action: string): string {
  const s = action.replace(/^admin\./, '').replace(/[._]/g, ' ').trim()
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function eventIcon(action: string): string {
  if (action.includes('delete') || action.includes('revoke')) return 'mdi-delete-outline'
  if (action.includes('create') || action.includes('invite')) return 'mdi-plus-circle-outline'
  if (action.includes('login')) return 'mdi-login-variant'
  if (action.includes('update') || action.includes('settings')) return 'mdi-pencil-outline'
  if (action.includes('2fa') || action.includes('password')) return 'mdi-shield-key-outline'
  return 'mdi-circle-small'
}

function eventSeverity(action: string): string {
  const a = action.toLowerCase()
  if (a.includes('delete')) return 'sev-error'
  if (a.includes('fail') || a.includes('revoke') || a.includes('disable')) return 'sev-warning'
  return ''
}

const containerPct = computed(() => {
  const m = metrics.value
  if (!m || !m.total_containers) return 0
  return Math.round((m.running_containers / m.total_containers) * 100)
})

const poolPct = computed(() => {
  const p = metrics.value?.network_pool
  if (!p || !p.total) return 0
  return Math.round((p.used / p.total) * 100)
})

const quickLinks = [
  { label: 'Users', icon: 'mdi-account-group-outline', to: '/admin/users' },
  { label: 'Workspaces', icon: 'mdi-briefcase-outline', to: '/admin/workspaces' },
  { label: 'Nodes', icon: 'mdi-server-network', to: '/admin/nodes' },
  { label: 'Events', icon: 'mdi-timeline-text-outline', to: '/admin/events' },
  { label: 'Settings', icon: 'mdi-cog-outline', to: '/admin/settings' },
]

async function loadInitial() {
  try {
    metrics.value = (await adminApi.metrics()).data.data
  } catch (err) {
    notify.apiError(err, 'Failed to load metrics')
  }
}

async function loadEvents() {
  try {
    events.value = (await adminApi.listEvents('', '', 0, 8)).data.data ?? []
  } catch {
    // best-effort
  }
}

function openStream() {
  es = new EventSource(sseUrl('/admin/metrics/stream'))
  es.onmessage = (e) => {
    try {
      metrics.value = JSON.parse(e.data) as PlatformMetrics
    } catch {
      // ignore malformed payloads
    }
  }
}

onMounted(() => {
  loadInitial()
  loadEvents()
  openStream()
})

onBeforeUnmount(() => {
  es?.close()
  es = null
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Platform Admin Dashboard</h1>
        <p class="subtitle">Platform health and activity at a glance</p>
      </div>
      <div class="live-indicator">
        <span class="live-dot"></span>
        <span class="text-muted">Live</span>
      </div>
    </div>

    <div v-if="!metrics" class="card">
      <div class="card-body" style="display: flex; justify-content: center; padding: 48px 0">
        <span class="spinner"></span>
      </div>
    </div>

    <template v-else>
      <!-- Status hero -->
      <div class="hero card">
        <div class="hero-status">
          <span class="hero-badge"><span class="mdi mdi-check-circle"></span></span>
          <div>
            <div class="hero-title">All systems operational</div>
            <div class="hero-sub">
              {{ metrics.running_containers }} of {{ metrics.total_containers }} containers running
            </div>
          </div>
        </div>
        <div class="hero-meta">
          <div class="hero-stat">
            <span class="hero-stat-label">Uptime</span>
            <span class="hero-stat-value">{{ fmtUptime(metrics.uptime_seconds) }}</span>
          </div>
          <div class="hero-stat">
            <span class="hero-stat-label">Version</span>
            <span class="hero-stat-value">{{ metrics.version || 'dev' }}</span>
          </div>
          <div class="hero-stat">
            <span class="hero-stat-label">Memory</span>
            <span class="hero-stat-value">{{ fmtBytes(metrics.memory_alloc_bytes) }}</span>
          </div>
        </div>
      </div>

      <!-- Quick links -->
      <div class="quick-links">
        <button v-for="l in quickLinks" :key="l.to" class="quick-link card" @click="router.push(l.to)">
          <span class="mdi" :class="l.icon"></span>
          <span>{{ l.label }}</span>
        </button>
      </div>

      <!-- Platform -->
      <h2 class="section-title">Platform</h2>
      <div class="stats-grid">
        <div class="stat-card stat-card-clickable" @click="router.push('/admin/users')">
          <div class="stat-header">
            <span class="stat-label">Users</span>
            <span class="stat-icon stat-icon-primary"><span class="mdi mdi-account-group"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_users }}</div>
          <div class="stat-sub">{{ metrics.active_users }} active · {{ metrics.admin_users }} admin</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Active sessions</span>
            <span class="stat-icon stat-icon-info"><span class="mdi mdi-key-outline"></span></span>
          </div>
          <div class="stat-value">{{ metrics.active_sessions }}</div>
        </div>
        <div class="stat-card stat-card-clickable" @click="router.push('/admin/workspaces')">
          <div class="stat-header">
            <span class="stat-label">Workspaces</span>
            <span class="stat-icon stat-icon-primary"><span class="mdi mdi-briefcase"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_workspaces }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Connected workers</span>
            <span class="stat-icon" :class="metrics.connected_workers > 0 ? 'stat-icon-success' : 'stat-icon-danger'">
              <span class="mdi mdi-server-network"></span>
            </span>
          </div>
          <div class="stat-value">{{ metrics.connected_workers }}</div>
          <div class="stat-sub">{{ metrics.connected_workers > 0 ? 'Processing jobs' : 'No workers running' }}</div>
        </div>
        <div class="stat-card stat-card-clickable" @click="router.push('/admin/runners')">
          <div class="stat-header">
            <span class="stat-label">Shared runners</span>
            <span class="stat-icon" :class="metrics.shared_runners_online > 0 ? 'stat-icon-success' : 'stat-icon-secondary'">
              <span class="mdi mdi-cog-transfer-outline"></span>
            </span>
          </div>
          <div class="stat-value">{{ metrics.shared_runners }}</div>
          <div class="stat-sub">{{ metrics.shared_runners_online }} online</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Workspace runners</span>
            <span class="stat-icon" :class="metrics.workspace_runners_online > 0 ? 'stat-icon-success' : 'stat-icon-secondary'">
              <span class="mdi mdi-cog-outline"></span>
            </span>
          </div>
          <div class="stat-value">{{ metrics.workspace_runners }}</div>
          <div class="stat-sub">{{ metrics.workspace_runners_online }} online</div>
        </div>
      </div>

      <!-- Tenancy -->
      <h2 class="section-title">Resources</h2>
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Applications</span>
            <span class="stat-icon stat-icon-primary"><span class="mdi mdi-cube-outline"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_applications }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Databases</span>
            <span class="stat-icon stat-icon-info"><span class="mdi mdi-database"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_databases }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Stacks</span>
            <span class="stat-icon stat-icon-primary"><span class="mdi mdi-layers"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_stacks }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Volumes</span>
            <span class="stat-icon stat-icon-secondary"><span class="mdi mdi-harddisk"></span></span>
          </div>
          <div class="stat-value">{{ metrics.total_volumes }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">Storage used</span>
            <span class="stat-icon stat-icon-secondary"><span class="mdi mdi-database-arrow-up-outline"></span></span>
          </div>
          <div class="stat-value">{{ fmtBytes(metrics.storage_used_bytes) }}</div>
          <div class="stat-sub">{{ fmtBytes(metrics.storage_declared_bytes) }} declared</div>
        </div>
      </div>

      <!-- Runtime + activity -->
      <h2 class="section-title">Runtime</h2>
      <div class="runtime-row">
        <div class="card runtime-card">
          <div class="card-body">
            <div class="metric-card-head">
              <span class="mdi mdi-docker"></span>
              <span class="metric-label">Container utilization</span>
            </div>
            <div class="metric-value">{{ metrics.running_containers }}<span class="metric-total">/{{ metrics.total_containers }}</span></div>
            <div class="progress">
              <div class="progress-bar" :style="{ width: containerPct + '%' }"></div>
            </div>
            <div class="stat-sub">{{ containerPct }}% running</div>
          </div>
        </div>

        <div class="card runtime-card">
          <div class="card-body runtime-mini">
            <div class="mini-stat">
              <span class="mdi mdi-sync stat-icon stat-icon-info"></span>
              <div>
                <div class="metric-label">Goroutines</div>
                <div class="mini-value">{{ metrics.goroutines }}</div>
              </div>
            </div>
            <div class="mini-stat">
              <span class="mdi mdi-memory stat-icon stat-icon-primary"></span>
              <div>
                <div class="metric-label">Memory</div>
                <div class="mini-value">{{ fmtBytes(metrics.memory_alloc_bytes) }}</div>
              </div>
            </div>
            <div class="mini-stat">
              <span class="mdi mdi-clock-outline stat-icon stat-icon-secondary"></span>
              <div>
                <div class="metric-label">Uptime</div>
                <div class="mini-value">{{ fmtUptime(metrics.uptime_seconds) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Network subnet pool -->
      <template v-if="metrics.network_pool">
        <h2 class="section-title">Network pool</h2>
        <div class="card runtime-card">
          <div class="card-body">
            <div class="metric-card-head">
              <span class="mdi mdi-ip-network-outline"></span>
              <span class="metric-label">Subnet pool utilization</span>
            </div>
            <div class="metric-value">{{ metrics.network_pool.used }}<span class="metric-total">/{{ metrics.network_pool.total }}</span></div>
            <div class="progress">
              <div class="progress-bar" :class="{ 'progress-warn': poolPct >= 85 }" :style="{ width: poolPct + '%' }"></div>
            </div>
            <div class="stat-sub">
              {{ poolPct }}% used · {{ metrics.network_pool.available }} subnets free
              <span v-if="poolPct >= 85" class="pool-warn"> · nearing capacity — enlarge MIABI_NETWORK_POOL_CIDR</span>
            </div>
          </div>
        </div>
      </template>

      <!-- Connected workers -->
      <template v-if="metrics.workers && metrics.workers.length">
        <h2 class="section-title">Workers</h2>
        <div class="card worker-card">
          <div class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Host</th>
                  <th>Type</th>
                  <th>PID</th>
                  <th>Queues</th>
                  <th>Active</th>
                  <th>Status</th>
                  <th>Started</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="w in metrics.workers" :key="`${w.host}:${w.pid}`">
                  <td class="mono">{{ w.host }}</td>
                  <td>
                    <span class="badge" :class="w.type === 'embedded' ? 'badge-info' : 'badge-success'">{{ w.type }}</span>
                  </td>
                  <td class="mono">{{ w.pid }}</td>
                  <td>
                    <span v-for="(c, q) in w.queues" :key="q" class="badge badge-neutral queue-badge">{{ q }}: {{ c }}</span>
                  </td>
                  <td class="mono">{{ w.active_tasks }} / {{ w.concurrency }}</td>
                  <td>
                    <span class="badge" :class="w.status === 'active' ? 'badge-success' : 'badge-warning'">{{ w.status }}</span>
                  </td>
                  <td class="text-muted">{{ relTime(w.started) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>

      <!-- Recent platform activity -->
      <div class="card" style="margin-top: 20px">
        <div class="card-header">
          <h2>Recent activity</h2>
          <button class="btn btn-ghost btn-sm" @click="router.push('/admin/events')">View all</button>
        </div>
        <div v-if="events.length === 0" class="empty-state" style="padding: 28px">
          <span class="mdi mdi-timeline-text-outline" style="font-size: 32px; color: var(--text-muted)"></span>
          <p>No recorded activity yet.</p>
        </div>
        <ul v-else class="timeline">
          <li v-for="e in events" :key="e.id" class="event">
            <span class="event-icon" :class="eventSeverity(e.action)"><span class="mdi" :class="eventIcon(e.action)"></span></span>
            <div class="event-body">
              <div class="event-row">
                <span class="event-msg">{{ prettyAction(e.action) }}</span>
                <span class="event-time">{{ relTime(e.created_at) }}</span>
              </div>
              <span class="event-type">{{ e.target_type }}{{ e.target_id ? ' · ' + e.target_id : '' }}{{ e.ip_address ? ' · ' + e.ip_address : '' }}</span>
            </div>
          </li>
        </ul>
      </div>

      <div class="mt-4 text-muted version-line">
        Miabi {{ metrics.version }} · {{ metrics.commit }}
      </div>
    </template>
  </div>
</template>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; }
.subtitle { font-size: 13px; color: var(--text-muted); margin-top: 2px; }

.live-indicator { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.live-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: var(--success-500, #22c55e);
  animation: live-pulse 1.6s ease-out infinite;
}
@keyframes live-pulse {
  0% { box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.5); }
  70% { box-shadow: 0 0 0 6px rgba(34, 197, 94, 0); }
  100% { box-shadow: 0 0 0 0 rgba(34, 197, 94, 0); }
}

/* Hero */
.hero {
  display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap;
  gap: 20px; padding: 20px 24px; margin-bottom: 20px;
}
.hero-status { display: flex; align-items: center; gap: 14px; }
.hero-badge {
  width: 44px; height: 44px; border-radius: 50%;
  display: inline-flex; align-items: center; justify-content: center;
  background: var(--success-50); color: var(--success-600); font-size: 24px;
}
.hero-title { font-size: 17px; font-weight: 700; color: var(--text-primary); }
.hero-sub { font-size: 13px; color: var(--text-muted); margin-top: 2px; }
.hero-meta { display: flex; gap: 28px; flex-wrap: wrap; }
.hero-stat { display: flex; flex-direction: column; }
.hero-stat-label { font-size: 12px; color: var(--text-muted); }
.hero-stat-value { font-size: 18px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }

/* Quick links */
.quick-links {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px; margin-bottom: 24px;
}
.quick-link {
  display: flex; align-items: center; gap: 10px; padding: 14px 16px;
  cursor: pointer; font-size: 14px; font-weight: 500; color: var(--text-secondary);
  background: var(--bg-primary); border: 1px solid var(--border-primary);
  transition: border-color 0.15s, transform 0.15s;
}
.quick-link:hover { border-color: var(--primary-400); transform: translateY(-1px); color: var(--text-primary); }
.quick-link .mdi { font-size: 20px; color: var(--primary-500); }

.section-title {
  font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em;
  color: var(--text-muted); margin: 0 0 12px; padding-top: 4px;
}
.stats-grid { margin-bottom: 24px; }
.stat-card-clickable { cursor: pointer; }

/* Runtime */
.runtime-row { display: grid; grid-template-columns: 1fr 2fr; gap: 16px; }
@media (max-width: 720px) { .runtime-row { grid-template-columns: 1fr; } }
.metric-card-head { display: flex; align-items: center; gap: 6px; color: var(--text-muted); }
.metric-card-head .mdi { font-size: 18px; color: var(--primary-500); }
.metric-label { color: var(--text-muted); font-size: 13px; }
.metric-value { font-size: 28px; font-weight: 700; color: var(--text-primary); margin-top: 8px; font-variant-numeric: tabular-nums; }
.metric-total { font-size: 18px; font-weight: 600; color: var(--text-muted); }
.progress { height: 8px; border-radius: 9999px; background: var(--bg-tertiary); overflow: hidden; margin: 12px 0 6px; }
.progress-bar { height: 100%; border-radius: 9999px; background: var(--success-500); transition: width 0.4s ease; }
.progress-bar.progress-warn { background: var(--warning-500, #f59e0b); }
.pool-warn { color: var(--warning-600, #d97706); font-weight: 600; }
.runtime-mini { display: flex; justify-content: space-around; gap: 16px; flex-wrap: wrap; }
.mini-stat { display: flex; align-items: center; gap: 10px; }
.mini-stat .stat-icon { width: 36px; height: 36px; border-radius: var(--radius); display: inline-flex; align-items: center; justify-content: center; font-size: 18px; }
.mini-value { font-size: 20px; font-weight: 700; color: var(--text-primary); font-variant-numeric: tabular-nums; }

/* Timeline (matches dashboard) */
.timeline { list-style: none; margin: 0; padding: 8px 0; }
.event { display: flex; gap: 12px; padding: 10px 20px; }
.event + .event { border-top: 1px solid var(--border-secondary); }
.event-icon {
  flex-shrink: 0; width: 30px; height: 30px; border-radius: 50%;
  display: inline-flex; align-items: center; justify-content: center; font-size: 16px;
  background: var(--bg-tertiary); color: var(--text-secondary);
}
.event-icon.sev-warning { background: var(--warning-50); color: var(--warning-600); }
.event-icon.sev-error { background: var(--danger-50); color: var(--danger-600); }
.event-body { flex: 1; min-width: 0; }
.event-row { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }
.event-msg { font-size: 14px; color: var(--text-primary); }
.event-time { flex-shrink: 0; font-size: 12px; color: var(--text-muted); font-variant-numeric: tabular-nums; }
.event-type { font-size: 11px; color: var(--text-muted); font-family: 'JetBrains Mono', monospace; }

.version-line { font-size: 12px; }

/* Workers table */
.worker-card { margin-bottom: 24px; }
.worker-card .mono { font-family: 'JetBrains Mono', monospace; font-size: 12.5px; }
.queue-badge { margin-right: 6px; }
.queue-badge:last-child { margin-right: 0; }
</style>
