<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import { useLicenseStore } from '@/stores/license'
import { apiError as decodeApiError } from '@/api/client'
import { nodesApi, type CreateNodePayload } from '@/api/nodes'
import { clusterApi } from '@/api/cluster'
import { adminApi } from '@/api/admin'
import { ACCESS_MODES, CONNECTIVITY_TYPES, nodeOptionDescription } from '@/constants/node'
import FieldInfo from '@/components/FieldInfo.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { copyText } from '@/utils/clipboard'
import type { Server, ServerConnectivity, ClusterStatus, ClusterJoinInstructions } from '@/api/types'

const notify = useNotificationStore()
const router = useRouter()
const license = useLicenseStore()

const nodes = ref<Server[]>([])
const loading = ref(false)
const agentImage = ref('ghcr.io/miabi-io/agent:latest')

// Edition node cap (-1 = unlimited). Count comes from the live list so it stays
// accurate after add/remove. Enforced server-side; this just surfaces it.
const nodeLimit = computed(() => license.view?.node_usage?.limit ?? -1)
const nodeCount = computed(() => nodes.value.length)
const limited = computed(() => nodeLimit.value >= 0)
const atNodeLimit = computed(() => limited.value && nodeCount.value >= nodeLimit.value)

async function load() {
  loading.value = true
  try {
    nodes.value = (await nodesApi.list()).data.data ?? []
  } catch (e) {
    notify.apiError(e)
  } finally {
    loading.value = false
  }
}

// --- Cluster (Docker Swarm) ---
const cluster = ref<ClusterStatus | null>(null)
const clusterEnabled = computed(() => cluster.value?.enabled === true)
// Command to attach a self-managed reverse proxy to the shared ingress overlay,
// so clustered apps are publicly reachable through it. Miabi attaches its own
// managed gateway automatically; this is only needed for a user-run proxy.
const ingressAttachCmd = computed(() =>
  cluster.value?.ingress_network
    ? `docker network connect ${cluster.value.ingress_network} <your-proxy-container>`
    : '',
)
const clusterBusy = ref(false)
const showEnable = ref(false)
const advertiseAddr = ref('')

// Workspace networks still on node-local bridges. While cluster mode is on, these
// workspaces have NO cross-node connectivity: their apps and databases sit on
// per-node islands, and an app on one node cannot resolve a database on another.
// Normal for an install that was already clustered before this version — the
// conversion only runs on the enable transition — so it must be applied explicitly.
const networksPending = computed(() => cluster.value?.networks_pending ?? 0)
const showApplyNetworking = ref(false)

async function applyNetworking() {
  showApplyNetworking.value = false
  clusterBusy.value = true
  try {
    cluster.value = (await clusterApi.applyNetworking()).data.data
    notify.success('Workspace networks converted to cluster overlays')
    load()
  } catch (e) {
    notify.apiError(e, 'Failed to apply cluster networking')
  } finally {
    clusterBusy.value = false
  }
}

async function loadCluster() {
  try {
    cluster.value = (await clusterApi.status()).data.data
  } catch { /* status is best-effort; the page still works without it */ }
}

function openEnable() {
  // Prefill with the manager's known address as a sensible default; the admin
  // should use the private/WG address peers can reach.
  const mgr = nodes.value.find((n) => n.is_local)
  advertiseAddr.value = mgr?.address || mgr?.public_ip || ''
  showEnable.value = true
}

async function enableCluster() {
  clusterBusy.value = true
  try {
    cluster.value = (await clusterApi.enable(advertiseAddr.value.trim())).data.data
    showEnable.value = false
    notify.success('Cluster mode enabled')
    load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    clusterBusy.value = false
  }
}

const showDisableCluster = ref(false)
async function disableCluster() {
  showDisableCluster.value = false
  clusterBusy.value = true
  try {
    await clusterApi.disable()
    notify.success('Cluster mode disabled')
    await loadCluster()
    load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    clusterBusy.value = false
  }
}

async function joinNode(n: Server) {
  clusterBusy.value = true
  try {
    await clusterApi.joinNode(n.id)
    notify.success(`${n.name} joined the cluster`)
    await loadCluster()
    load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    clusterBusy.value = false
  }
}

const pendingLeave = ref<Server | null>(null)
async function leaveNode() {
  const n = pendingLeave.value
  if (!n) return
  pendingLeave.value = null
  clusterBusy.value = true
  try {
    await clusterApi.leaveNode(n.id)
    notify.success(`${n.name} removed from the cluster`)
    await loadCluster()
    load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    clusterBusy.value = false
  }
}

// A node can be joined when cluster mode is on, the node is a remote, online,
// non-member node.
function canJoin(n: Server): boolean {
  return clusterEnabled.value && !n.is_local && !!n.agent_connected && !n.in_swarm
}

// --- Join nodes to the cluster (header action) ---
const showJoin = ref(false)
const joinBusy = ref(false)
const joinSelected = ref<Record<number, boolean>>({})
// Manual `docker swarm join` command for hosts not connected to the manager.
const manualJoin = ref<ClusterJoinInstructions | null>(null)

// Candidate nodes for the join dialog: remote nodes not yet in the swarm
// (offline ones are listed but cannot be joined until their agent reconnects).
const joinCandidates = computed(() => nodes.value.filter((n) => !n.is_local && !n.in_swarm))
const selectedJoinIds = computed(() =>
  joinCandidates.value.filter((n) => joinSelected.value[n.id] && n.agent_connected).map((n) => n.id),
)

async function openJoin() {
  // Preselect every eligible (online) candidate.
  const sel: Record<number, boolean> = {}
  for (const n of joinCandidates.value) if (n.agent_connected) sel[n.id] = true
  joinSelected.value = sel
  manualJoin.value = null
  showJoin.value = true
  // Fetch the manual join command for hosts not connected to the manager.
  try {
    manualJoin.value = (await clusterApi.joinToken()).data.data
  } catch { /* best-effort; the host-side command just won't show */ }
}

async function joinSelectedNodes() {
  const ids = selectedJoinIds.value
  if (ids.length === 0) return
  joinBusy.value = true
  let joined = 0
  for (const id of ids) {
    try {
      await clusterApi.joinNode(id)
      joined++
    } catch (e) {
      notify.apiError(e)
    }
  }
  joinBusy.value = false
  showJoin.value = false
  if (joined > 0) notify.success(`${joined} node${joined > 1 ? 's' : ''} joined the cluster`)
  await loadCluster()
  load()
}
async function loadAgentImage() {
  try {
    const cfg = (await adminApi.getDeploymentConfig()).data.data
    const agent = cfg.images?.find((i) => i.key === 'agent')
    if (agent?.effective) agentImage.value = agent.effective
  } catch { /* fall back to the default ref */ }
}
onMounted(() => { load(); loadAgentImage(); loadCluster(); license.load() })

// --- Add node ---
const showCreate = ref(false)
const creating = ref(false)
const blankForm = (): CreateNodePayload => ({
  name: '', address: '', connectivity: 'port-forward', access_mode: 'agent',
  docker_endpoint: '', tls_ca_cert: '', tls_cert: '', tls_key: '',
})
const form = ref<CreateNodePayload>(blankForm())
const createdToken = ref<string | null>(null)

function openCreate() {
  form.value = blankForm()
  createdToken.value = null
  showCreate.value = true
}

// Endpoint placeholder per access mode.
const endpointPlaceholder = computed(() => form.value.access_mode === 'api' ? 'tcp://10.0.0.10:2376' : '')

// Hint lines describing the currently selected option.
const accessModeDesc = computed(() => nodeOptionDescription(ACCESS_MODES, form.value.access_mode))
const connectivityDesc = computed(() => nodeOptionDescription(CONNECTIVITY_TYPES, form.value.connectivity))

async function submit() {
  if (!form.value.name.trim()) return
  const mode = form.value.access_mode || 'agent'
  if (mode === 'api' && !form.value.docker_endpoint?.trim()) {
    notify.error('A Docker endpoint is required for this access mode')
    return
  }
  creating.value = true
  const payload: CreateNodePayload = {
    name: form.value.name.trim(),
    address: form.value.address?.trim() || undefined,
    connectivity: form.value.connectivity,
    access_mode: mode,
    docker_endpoint: form.value.docker_endpoint?.trim() || undefined,
    tls_ca_cert: form.value.tls_ca_cert || undefined,
    tls_cert: form.value.tls_cert || undefined,
    tls_key: form.value.tls_key || undefined,
  }
  try {
    const res = await nodesApi.create(payload)
    load()
    license.load(true) // refresh node usage so the cap chip stays accurate
    if (mode === 'agent') {
      createdToken.value = res.data.data.token
      notify.success('Node added — copy the join token now')
    } else {
      showCreate.value = false
      notify.success('Node added — connecting…')
    }
  } catch (e) {
    // The node cap is an edition limit: surface an upgrade-oriented message and
    // reveal the in-modal upgrade banner rather than a bare error toast.
    if (decodeApiError(e).code === 'NODE_LIMIT_REACHED') {
      license.load(true)
      notify.error(
        `Community edition is limited to ${nodeLimit.value} nodes. Upgrade to Enterprise to add more.`,
        { title: 'Node limit reached' },
      )
    } else {
      notify.apiError(e)
    }
  } finally {
    creating.value = false
  }
}

const ACCESS_LABELS: Record<string, string> = { socket: 'Local socket', agent: 'Agent', api: 'Docker API' }
function accessLabel(m?: string): string { return ACCESS_LABELS[m || 'agent'] || m || '—' }

const controlUrl = computed(() => window.location.origin)
const agentCommand = computed(
  () =>
    `docker run -d --name miabi-agent --restart unless-stopped \\\n` +
    `  -e MIABI_CONTROL_URL=${controlUrl.value} \\\n` +
    `  -e MIABI_NODE_TOKEN=${createdToken.value ?? '<token>'} \\\n` +
    `  -v /var/run/docker.sock:/var/run/docker.sock \\\n` +
    `  ${agentImage.value}`,
)

async function copy(text: string) {
  if (await copyText(text)) notify.success('Copied')
  else notify.error('Copy failed — select and copy it manually')
}

function connectivityLabel(c?: ServerConnectivity): string {
  return c === 'edge-gateway' ? 'Edge gateway' : 'Port forwarding'
}
function statusClass(n: Server): string {
  if (n.is_local || n.agent_connected) return 'badge-success badge-dot'
  return 'badge-danger'
}
function statusLabel(n: Server): string {
  if (n.is_local) return 'manager'
  return n.agent_connected ? 'online' : 'offline'
}
function roleLabel(n: Server): string {
  return n.role || (n.is_local ? 'manager' : 'node')
}
function fmtDate(s?: string): string { return s ? new Date(s).toLocaleDateString() : '—' }
// The Agent column only applies to agent-mode nodes; socket/Docker-API have none.
function agentLabel(n: Server): string {
  if (n.access_mode !== 'agent') return 'N/A'
  return n.agent_version || (n.agent_connected ? 'connected' : '—')
}

// Swarm column: the node's role in the cluster (or "standalone"), with a hint of
// its availability when it differs from the normal "active".
function swarmLabel(n: Server): string {
  if (!clusterEnabled.value) return '—'
  const role = n.swarm_role || 'standalone'
  if (n.in_swarm && n.swarm_availability && n.swarm_availability !== 'active') {
    return `${role} · ${n.swarm_availability}`
  }
  return role
}
function swarmClass(n: Server): string {
  if (!clusterEnabled.value || !n.in_swarm) return 'badge-muted'
  if (n.swarm_role === 'leader') return 'badge-info'
  if (n.swarm_state && n.swarm_state !== 'ready') return 'badge-warning'
  return 'badge-success'
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Nodes</h1>
      <div class="header-actions">
        <span v-if="limited" class="node-usage" :class="{ 'node-usage--full': atNodeLimit }" :title="atNodeLimit ? 'Node limit reached — upgrade to add more' : 'Nodes used of your edition limit'">
          <span class="mdi mdi-server"></span> {{ nodeCount }} / {{ nodeLimit }} nodes
        </span>
        <button v-if="clusterEnabled" class="btn btn-secondary" @click="openJoin"><span class="mdi mdi-lan-connect"></span> Join the cluster</button>
        <button class="btn btn-primary" @click="openCreate"><span class="mdi mdi-plus"></span> Add node</button>
      </div>
    </div>

    <!-- Cluster (Docker Swarm) status. Cluster mode is opt-in and auto-detected;
         single-node on plain Docker stays first-class. -->
    <div v-if="cluster" class="card cluster-bar">
      <div class="cluster-bar-main">
        <span class="mdi" :class="clusterEnabled ? 'mdi-lan-connect' : 'mdi-lan-disconnect'" style="font-size: 22px"></span>
        <div>
          <div class="cluster-bar-title">
            Cluster networking
            <span class="badge" :class="clusterEnabled ? 'badge-success' : 'badge-muted'">{{ clusterEnabled ? 'enabled' : 'disabled' }}</span>
          </div>
          <div class="cell-sub">
            <template v-if="clusterEnabled">Docker Swarm · {{ cluster.managers }} manager(s), {{ cluster.nodes }} node(s)<span v-if="cluster.manager_addr"> · advertises {{ cluster.manager_addr }}</span></template>
            <template v-else>Run apps across nodes on a private overlay network. Single-node on plain Docker is unaffected.</template>
            <span v-if="cluster.error" class="badge badge-danger" style="margin-left: 8px">{{ cluster.error }}</span>
          </div>
          <!-- Self-managed reverse proxy: Miabi attaches its own gateway to the
               ingress overlay automatically. If you run your own proxy, connect it
               to that overlay so clustered apps are reachable through it. -->
          <div v-if="clusterEnabled && cluster.ingress_network" class="ingress-hint">
            <span class="mdi mdi-information-outline"></span>
            Running your own reverse proxy? Attach it to the ingress overlay
            <code>{{ cluster.ingress_network }}</code>:
            <code class="ingress-cmd">{{ ingressAttachCmd }}</code>
            <button type="button" class="btn btn-ghost btn-sm" title="Copy command" @click="copy(ingressAttachCmd)">
              <span class="mdi mdi-content-copy"></span>
            </button>
          </div>
          <!-- Cluster mode is on, but some workspaces are still on node-local
               bridges, so they have no cross-node connectivity at all. Say exactly
               that, rather than leaving the admin to discover it as an app that
               can't resolve its database. -->
          <div v-if="clusterEnabled && networksPending > 0" class="pending-hint">
            <span class="mdi mdi-alert-outline"></span>
            <span>
              <strong>{{ networksPending }} workspace network(s) are still node-local bridges.</strong>
              Apps and databases in them can't reach each other across nodes — an app on one node
              won't resolve a database on another. Convert them to cluster overlays to fix it.
            </span>
            <button type="button" class="btn btn-sm btn-primary" :disabled="clusterBusy" @click="showApplyNetworking = true">
              Apply cluster networking
            </button>
          </div>
        </div>
      </div>
      <div>
        <button v-if="!clusterEnabled" class="btn btn-secondary" :disabled="clusterBusy" @click="openEnable">Enable cluster</button>
        <button v-else class="btn btn-secondary" :disabled="clusterBusy" @click="showDisableCluster = true">Disable cluster</button>
      </div>
    </div>

    <div class="card">
      <div v-if="loading && nodes.length === 0" class="card-body"><span class="spinner"></span></div>
      <div v-else-if="nodes.length === 0" class="empty-state">
        <span class="mdi mdi-server-network" style="font-size: 44px; color: var(--text-muted)"></span>
        <h3>No nodes</h3>
        <p>Add a node to run apps on additional Docker hosts.</p>
        <button class="btn btn-primary mt-4" @click="openCreate">Add a node</button>
      </div>
      <div v-else class="table-wrapper">
        <table>
          <thead><tr><th>Name</th><th>Role</th><th>Access</th><th>Connectivity</th><th>Status</th><th v-if="clusterEnabled">Swarm</th><th>Agent</th><th>Created</th></tr></thead>
          <tbody>
            <tr v-for="n in nodes" :key="n.id" class="row-clickable" @click="router.push(`/admin/nodes/${n.id}`)">
              <td>
                <div class="cell-id">
                  <span class="avatar avatar-sm"><span class="mdi mdi-server" style="font-size: 14px"></span></span>
                  <span class="cell-text">
                    <span class="cell-title">{{ n.name }}<span v-if="n.cordoned" class="badge badge-warning" style="margin-left: 8px">cordoned</span></span>
                    <span class="cell-sub">{{ n.address || (n.is_local ? 'local socket' : '—') }}</span>
                  </span>
                </div>
              </td>
              <td><span class="badge" :class="roleLabel(n) === 'manager' ? 'badge-info' : 'badge-muted'">{{ roleLabel(n) }}</span></td>
              <td>
                <span class="badge badge-muted">{{ accessLabel(n.access_mode) }}</span>
                <span v-if="n.access_mode === 'api' && n.tls_enabled" class="mdi mdi-lock-outline" title="TLS" style="margin-left: 4px"></span>
              </td>
              <td><span class="badge badge-muted">{{ connectivityLabel(n.connectivity) }}</span></td>
              <td><span class="badge" :class="statusClass(n)">{{ statusLabel(n) }}</span></td>
              <td v-if="clusterEnabled">
                <span class="badge" :class="swarmClass(n)">{{ swarmLabel(n) }}</span>
                <button v-if="canJoin(n)" class="btn btn-xs btn-secondary" style="margin-left: 6px" :disabled="clusterBusy" @click.stop="joinNode(n)">Join</button>
                <button v-else-if="n.in_swarm && !n.is_local" class="btn btn-xs btn-secondary" style="margin-left: 6px" :disabled="clusterBusy" @click.stop="pendingLeave = n">Leave</button>
              </td>
              <td class="cell-sub">{{ agentLabel(n) }}</td>
              <td class="cell-sub">{{ fmtDate(n.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add node / token reveal -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
        <div class="modal">
          <div class="modal-header">
            <h3>Add node</h3>
            <button class="btn-icon btn-icon-muted" aria-label="Close" @click="showCreate = false"><span class="mdi mdi-close"></span></button>
          </div>

          <template v-if="!createdToken">
            <form @submit.prevent="submit">
              <div class="modal-body">
                <!-- Edition node cap reached: nodes are a paid scale dimension.
                     Block the add and point to the upgrade page. -->
                <div v-if="atNodeLimit" class="app-banner app-banner--warning">
                  <span class="mdi mdi-lock-outline app-banner-icon"></span>
                  <div class="app-banner-content">
                    <p class="app-banner-title">Node limit reached</p>
                    <p class="app-banner-text">
                      Community edition is limited to {{ nodeLimit }} nodes (standalone or Swarm).
                      Upgrade to Enterprise to add more.
                    </p>
                    <router-link to="/admin/license" class="btn btn-secondary btn-sm" style="margin-top: 10px">Upgrade</router-link>
                  </div>
                </div>
                <div class="form-group">
                  <label class="form-label">Name</label>
                  <input v-model="form.name" class="form-input" placeholder="e.g. edge-eu-1" required autofocus />
                </div>
                <div class="form-group">
                  <span class="form-label label-row">
                    Access mode
                    <FieldInfo :items="ACCESS_MODES" title="Access modes explained" />
                  </span>
                  <select v-model="form.access_mode" class="form-input">
                    <option v-for="o in ACCESS_MODES" :key="o.value" :value="o.value">{{ o.label }}</option>
                  </select>
                  <p class="form-hint">{{ accessModeDesc }}</p>
                </div>

                <!-- api: endpoint + TLS -->
                <template v-if="form.access_mode === 'api'">
                  <div class="form-group">
                    <label class="form-label">Docker endpoint</label>
                    <input v-model="form.docker_endpoint" class="form-input" :placeholder="endpointPlaceholder" required style="font-family: monospace" />
                    <p class="cell-sub" style="margin-top: 4px">The node must be reachable from the manager (inbound).</p>
                  </div>
                  <div class="form-group">
                    <label class="form-label">TLS <span class="cell-sub">(optional — leave blank for plaintext on a trusted network)</span></label>
                    <textarea v-model="form.tls_ca_cert" class="form-input" rows="2" placeholder="CA certificate (PEM)" style="font-family: monospace; font-size: 12px"></textarea>
                    <textarea v-model="form.tls_cert" class="form-input" rows="2" placeholder="Client certificate (PEM) — for mTLS" style="font-family: monospace; font-size: 12px; margin-top: 6px"></textarea>
                    <textarea v-model="form.tls_key" class="form-input" rows="2" placeholder="Client key (PEM) — stored encrypted" style="font-family: monospace; font-size: 12px; margin-top: 6px"></textarea>
                  </div>
                </template>

                <div v-if="form.access_mode !== 'api'" class="form-group">
                  <label class="form-label">Address <span class="cell-sub">(host/IP the proxy reaches published ports at)</span></label>
                  <input v-model="form.address" class="form-input" placeholder="e.g. 10.0.0.7" />
                </div>
                <div class="form-group" style="margin-bottom: 0">
                  <span class="form-label label-row">
                    Connectivity
                    <FieldInfo :items="CONNECTIVITY_TYPES" title="Connectivity types explained" placement="top" />
                  </span>
                  <select v-model="form.connectivity" class="form-input">
                    <option v-for="o in CONNECTIVITY_TYPES" :key="o.value" :value="o.value">{{ o.label }}</option>
                  </select>
                  <p class="form-hint">{{ connectivityDesc }}</p>
                </div>
              </div>
              <div class="modal-footer">
                <button type="button" class="btn btn-secondary" @click="showCreate = false">Cancel</button>
                <button type="submit" class="btn btn-primary" :disabled="creating || atNodeLimit">{{ creating ? 'Saving…' : 'Add node' }}</button>
              </div>
            </form>
          </template>

          <template v-else>
            <div class="modal-body">
              <div class="app-banner app-banner--warning">
                <span class="mdi mdi-alert-outline app-banner-icon"></span>
                <div class="app-banner-content">
                  <p class="app-banner-title">Copy the join token now</p>
                  <p class="app-banner-text">This is the only time it is shown. Run the agent on the node:</p>
                </div>
              </div>
              <div class="code-block" style="margin-top: 14px; white-space: pre">{{ agentCommand }}</div>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" @click="copy(createdToken!)">Copy token</button>
              <button type="button" class="btn btn-secondary" @click="copy(agentCommand)">Copy command</button>
              <button type="button" class="btn btn-primary" @click="showCreate = false">Done</button>
            </div>
          </template>
        </div>
      </div>
    </Teleport>

    <!-- Join nodes to the cluster -->
    <Teleport to="body">
      <div v-if="showJoin" class="modal-overlay" @click.self="showJoin = false">
        <div class="modal">
          <div class="modal-header">
            <h3>Join nodes to the cluster</h3>
            <button class="btn-icon btn-icon-muted" aria-label="Close" @click="showJoin = false"><span class="mdi mdi-close"></span></button>
          </div>
          <div class="modal-body">
            <!-- Managed nodes (connected to the manager): one-click join. -->
            <p v-if="joinCandidates.length === 0" class="cell-sub">All managed nodes are already in the cluster.</p>
            <template v-else>
              <p class="cell-sub" style="margin-bottom: 12px">
                Select the nodes to join the swarm overlay network. Offline nodes can't be joined from here until their agent reconnects — use the manual command below.
              </p>
              <label
                v-for="n in joinCandidates"
                :key="n.id"
                class="join-row"
                :class="{ 'join-row--disabled': !n.agent_connected }"
              >
                <input type="checkbox" :disabled="!n.agent_connected" v-model="joinSelected[n.id]" />
                <span class="join-row-name">{{ n.name }}</span>
                <span class="badge" :class="n.agent_connected ? 'badge-success badge-dot' : 'badge-danger'">{{ n.agent_connected ? 'online · standalone' : 'offline' }}</span>
              </label>
            </template>

            <!-- Manual join: for a host not connected to the manager (offline or
                 unmanaged). The operator runs the command on the host itself. -->
            <div class="manual-join">
              <div class="manual-join-title">Join a node manually</div>
              <p class="cell-sub" style="margin-bottom: 8px">
                For a host that isn't connected to Miabi, run this on the host. It must reach the manager on ports 2377/tcp, 7946/tcp+udp and 4789/udp.
              </p>
              <div v-if="manualJoin" class="code-block" style="white-space: pre-wrap; word-break: break-all">{{ manualJoin.command }}</div>
              <p v-else class="cell-sub">Join command unavailable.</p>
              <button v-if="manualJoin" type="button" class="btn btn-secondary btn-sm" style="margin-top: 10px" @click="copy(manualJoin.command)">Copy command</button>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showJoin = false">Cancel</button>
            <button type="button" class="btn btn-primary" :disabled="joinBusy || selectedJoinIds.length === 0" @click="joinSelectedNodes">
              {{ joinBusy ? 'Joining…' : selectedJoinIds.length ? `Join ${selectedJoinIds.length}` : 'Join' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Enable cluster mode -->
    <Teleport to="body">
      <div v-if="showEnable" class="modal-overlay" @click.self="showEnable = false">
        <div class="modal">
          <div class="modal-header">
            <h3>Enable cluster mode</h3>
            <button class="btn-icon btn-icon-muted" aria-label="Close" @click="showEnable = false"><span class="mdi mdi-close"></span></button>
          </div>
          <form @submit.prevent="enableCluster">
            <div class="modal-body">
              <p class="cell-sub" style="margin-bottom: 12px">
                The manager initializes a Docker Swarm. Member nodes can then be joined to a private overlay network.
                If Docker is already in swarm mode, Miabi adopts it instead.
              </p>
              <div class="form-group" style="margin-bottom: 0">
                <label class="form-label">Advertise address</label>
                <input v-model="advertiseAddr" class="form-input" placeholder="e.g. 10.0.0.1" style="font-family: monospace" autofocus />
                <p class="form-hint">The address swarm peers reach this manager on — use a private/WG address reachable from your nodes.</p>
              </div>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" @click="showEnable = false">Cancel</button>
              <button type="submit" class="btn btn-primary" :disabled="clusterBusy || !advertiseAddr.trim()">{{ clusterBusy ? 'Enabling…' : 'Enable cluster' }}</button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>

    <ConfirmDialog
      :open="showApplyNetworking"
      title="Apply cluster networking?"
      message="Each workspace network is converted from a node-local bridge to a cluster overlay, so apps and databases reach each other across nodes. Containers are NOT restarted, but connections open inside a workspace drop briefly while it switches over."
      confirm-label="Apply"
      :busy="clusterBusy"
      @confirm="applyNetworking"
      @cancel="showApplyNetworking = false"
    />

    <ConfirmDialog
      :open="showDisableCluster"
      title="Disable cluster mode?"
      message="The manager and all member nodes will leave the swarm. Workspace networks are moved back to node-local bridges first, so apps and databases stop being reachable across nodes — anything relying on that will break. Containers are not restarted."
      confirm-label="Disable cluster"
      variant="danger"
      :busy="clusterBusy"
      @confirm="disableCluster"
      @cancel="showDisableCluster = false"
    />

    <ConfirmDialog
      :open="!!pendingLeave"
      title="Remove node from cluster?"
      :message="`Remove ${pendingLeave?.name} from the cluster?`"
      confirm-label="Remove"
      variant="danger"
      :busy="clusterBusy"
      @confirm="leaveNode"
      @cancel="pendingLeave = null"
    />
  </div>
</template>

<style scoped>
.label-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.cluster-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 18px;
  margin-bottom: 16px;
}
.cluster-bar-main {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}
.cluster-bar-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}
.ingress-hint {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.ingress-cmd {
  font-family: var(--font-mono, monospace);
  user-select: all;
}
.pending-hint {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 10px;
  padding: 8px 10px;
  border: 1px solid var(--warning-border, #f5c26b);
  border-radius: 6px;
  background: var(--warning-bg, rgba(245, 194, 107, 0.12));
  font-size: 12px;
  color: var(--text);
}
.pending-hint > span:first-child {
  color: var(--warning, #b45309);
  font-size: 16px;
}
.header-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.node-usage {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  background: var(--surface-2, rgba(127, 127, 127, 0.1));
}
.node-usage--full {
  color: var(--warning, #b7791f);
  background: var(--warning-bg, rgba(183, 121, 31, 0.12));
}
.join-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  cursor: pointer;
}
.join-row--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.join-row-name {
  flex: 1;
  font-weight: 500;
}
.manual-join {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}
.manual-join-title {
  font-weight: 600;
  margin-bottom: 6px;
}
</style>
