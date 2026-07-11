<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'
import { useNotificationStore } from '@/stores/notification'
import { usePagination } from '@/composables/usePagination'
import Pagination from '@/components/Pagination.vue'
import type { AdminDomain, DomainStatus } from '@/api/types'

const notify = useNotificationStore()
const router = useRouter()

const domains = ref<AdminDomain[]>([])
const loading = ref(false)
const search = ref('')
const status = ref<DomainStatus | ''>('')

let searchTimer: ReturnType<typeof setTimeout> | undefined

const { pageable, goToPage } = usePagination(async (page) => {
  loading.value = true
  try {
    const res = await adminApi.listDomains(page, pageable.value.size, search.value.trim(), status.value)
    domains.value = res.data.data
    pageable.value = res.data.pageable
  } catch (e) {
    notify.apiError(e)
  } finally {
    loading.value = false
  }
})

function reload() {
  goToPage(pageable.value.current_page)
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => goToPage(0), 300)
}

function setStatus(s: DomainStatus | '') {
  status.value = s
  goToPage(0)
}

function statusBadge(s: DomainStatus): string {
  if (s === 'verified') return 'badge-success'
  if (s === 'failed' || s === 'banned') return 'badge-danger'
  return 'badge-warning'
}

function statusIcon(s: DomainStatus): string {
  if (s === 'verified') return 'mdi-check-decagram'
  if (s === 'banned') return 'mdi-cancel'
  if (s === 'failed') return 'mdi-alert-circle-outline'
  return 'mdi-clock-alert-outline'
}

function fmtDate(s: string): string {
  return new Date(s).toLocaleDateString()
}

onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Domains</h1>
        <p class="text-muted">Every workspace's owned hostnames and their verification status.</p>
      </div>
      <button class="btn btn-secondary" :disabled="loading" @click="reload">
        <span class="mdi mdi-refresh"></span> Refresh
      </button>
    </div>

    <div class="card">
      <div class="card-body toolbar">
        <div class="search">
          <span class="mdi mdi-magnify"></span>
          <input
            v-model="search"
            class="form-input"
            type="search"
            placeholder="Search domains by name…"
            aria-label="Search domains by name"
            @input="onSearchInput"
          />
        </div>
        <div class="filters">
          <button class="chip" :class="{ active: status === '' }" @click="setStatus('')">All</button>
          <button class="chip" :class="{ active: status === 'verified' }" @click="setStatus('verified')">Verified</button>
          <button class="chip" :class="{ active: status === 'pending' }" @click="setStatus('pending')">Pending</button>
          <button class="chip" :class="{ active: status === 'failed' }" @click="setStatus('failed')">Failed</button>
          <button class="chip" :class="{ active: status === 'banned' }" @click="setStatus('banned')">Banned</button>
        </div>
        <span class="text-muted">{{ pageable.total_elements }} domain{{ pageable.total_elements === 1 ? '' : 's' }}</span>
      </div>

      <div v-if="loading && domains.length === 0" class="card-body"><span class="spinner"></span></div>

      <div v-else-if="domains.length === 0" class="empty-state">
        <span class="mdi mdi-web" style="font-size: 44px; color: var(--text-muted)"></span>
        <h3>{{ search.trim() || status ? 'No domains match your filters.' : 'No domains.' }}</h3>
      </div>

      <div v-else class="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>Domain</th>
              <th>Workspace</th>
              <th>Status</th>
              <th>TLS</th>
              <th>DNS</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in domains" :key="d.id" class="row-clickable" @click="router.push(`/admin/domains/${d.id}`)">
              <td>
                <span class="cell-text">
                  <span class="cell-title">{{ d.name }}</span>
                  <span v-if="d.wildcard" class="cell-sub">wildcard (*.{{ d.name }})</span>
                </span>
              </td>
              <td class="cell-sub">{{ d.workspace_name || ('#' + d.workspace_id) }}</td>
              <td>
                <span class="badge" :class="statusBadge(d.status)">
                  <span class="mdi" :class="statusIcon(d.status)"></span>
                  {{ d.status }}
                </span>
              </td>
              <td><span class="badge badge-neutral">{{ d.tls_mode === 'acme' ? 'automatic' : 'custom' }}</span></td>
              <td>
                <span class="badge" :class="d.automated ? 'badge-success' : 'badge-neutral'">{{ d.automated ? 'automated' : 'manual' }}</span>
              </td>
              <td class="cell-sub">{{ fmtDate(d.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Pagination :pageable="pageable" @page="goToPage" />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.search {
  position: relative;
  flex: 1;
  max-width: 360px;
}
.search .mdi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}
.search .form-input {
  padding-left: 32px;
}
.filters {
  display: flex;
  gap: 6px;
}
.chip {
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-muted);
  font-size: 13px;
  cursor: pointer;
}
.chip.active {
  background: var(--primary-500);
  border-color: var(--primary-500);
  color: #fff;
}
.badge .mdi {
  font-size: 13px;
}
</style>
