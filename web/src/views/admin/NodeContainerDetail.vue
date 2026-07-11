<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import { nodesApi } from '@/api/nodes'
import type { Container } from '@/api/types'

const route = useRoute()
const notify = useNotificationStore()
const id = Number(route.params.id)
const cid = String(route.params.cid)

const cont = ref<Container | null>(null)
const loading = ref(false)
const logs = ref<string[]>([])
const connected = ref(false)
let es: EventSource | null = null

async function load() {
  loading.value = true
  try {
    cont.value = (await nodesApi.inspectContainer(id, cid)).data.data
  } catch (e) { notify.apiError(e) } finally { loading.value = false }
}

function startLogs() {
  logs.value = []
  es = new EventSource(nodesApi.containerLogsUrl(id, cid))
  es.onopen = () => { connected.value = true }
  es.onmessage = (ev) => {
    try {
      const l = JSON.parse(ev.data) as { text?: string }
      if (l.text != null) logs.value.push(l.text)
    } catch { /* ignore */ }
  }
  es.onerror = () => { connected.value = false; es?.close() }
}

onMounted(() => { load(); startLogs() })
onBeforeUnmount(() => es?.close())

function cname(c: Container) { return ((c.names && c.names[0]) || c.id).replace(/^\//, '') }
function shortId(idv: string) { return idv.replace(/^sha256:/, '').slice(0, 12) }
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <router-link :to="`/admin/nodes/${id}/containers`" class="back-link"><span class="mdi mdi-arrow-left"></span> Containers</router-link>
        <h1>{{ cont ? cname(cont) : 'Container' }}</h1>
      </div>
    </div>

    <div v-if="loading && !cont" class="card"><div class="card-body"><span class="spinner"></span></div></div>

    <template v-else-if="cont">
      <div class="card mb-4">
        <div class="card-header"><h2>Details</h2><span class="badge badge-dot" :class="cont.state === 'running' ? 'badge-success' : 'badge-neutral'">{{ cont.state }}</span></div>
        <div class="card-body">
          <div v-for="f in [
            { label: 'ID', value: shortId(cont.id) },
            { label: 'Image', value: cont.image },
            { label: 'Status', value: cont.status },
          ]" :key="f.label" class="kv">
            <span class="kv-label">{{ f.label }}</span>
            <span class="kv-value mono">{{ f.value }}</span>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-header">
          <h2>Logs</h2>
          <span class="badge badge-dot" :class="connected ? 'badge-success' : 'badge-neutral'">{{ connected ? 'live' : 'disconnected' }}</span>
        </div>
        <pre class="logbox">{{ logs.join('\n') || 'Waiting for logs…' }}</pre>
      </div>
    </template>
  </div>
</template>

<style scoped>
.back-link { display: inline-flex; align-items: center; gap: 4px; color: var(--text-muted); font-size: 13px; text-decoration: none; margin-bottom: 4px; }
.back-link:hover { color: var(--text); }
.kv { display: flex; gap: 12px; padding: 8px 0; border-top: 1px solid var(--border-primary); }
.kv:first-child { border-top: none; }
.kv-label { width: 90px; font-size: 12px; font-weight: 600; color: var(--text-muted); text-transform: uppercase; }
.mono { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
.logbox { background: #0b0e14; color: #d6deeb; padding: 14px; border-radius: 8px; font-family: 'JetBrains Mono', monospace; font-size: 12px; line-height: 1.5; max-height: 480px; overflow: auto; white-space: pre-wrap; word-break: break-all; margin: 0; }
</style>
