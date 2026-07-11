<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import {
  platformBackupApi,
  type PlatformBackup,
  type PlatformBackupSettings,
  type PlatformBackupSettingsPayload,
  type PlatformVolume,
} from '@/api/platformBackup'
import { useNotificationStore } from '@/stores/notification'
import { useLicenseStore } from '@/stores/license'
import { useEntitlement } from '@/composables/useEntitlement'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const notify = useNotificationStore()
const licenseStore = useLicenseStore()
const ent = useEntitlement('platform_backup')

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const running = ref(false)
const restoringId = ref<number | null>(null)

const backups = ref<PlatformBackup[]>([])
const volumes = ref<PlatformVolume[]>([])
const settings = ref<PlatformBackupSettings | null>(null)

// Editable settings form.
const form = reactive<PlatformBackupSettingsPayload>({
  s3_enabled: false,
  s3_endpoint: '',
  s3_bucket: '',
  s3_region: '',
  s3_access_key: '',
  s3_secret_key: '',
  s3_use_ssl: true,
  s3_force_path_style: false,
  database_backup_path: '',
  volume_backup_path: '',
  schedule_enabled: false,
  schedule_cron: '0 3 * * *',
  max_backups: 7,
  retention_days: 30,
  volumes: [],
})
const secretSet = ref(false)

// "Back up now" selection.
const backupDB = ref(true)
const backupVolumes = ref<string[]>([])

function applySettings(s: PlatformBackupSettings) {
  settings.value = s
  form.s3_enabled = s.s3_enabled
  form.s3_endpoint = s.s3_endpoint ?? ''
  form.s3_bucket = s.s3_bucket ?? ''
  form.s3_region = s.s3_region ?? ''
  form.s3_access_key = s.s3_access_key ?? ''
  form.s3_secret_key = ''
  form.s3_use_ssl = s.s3_use_ssl
  form.s3_force_path_style = s.s3_force_path_style
  form.database_backup_path = s.database_backup_path ?? ''
  form.volume_backup_path = s.volume_backup_path ?? ''
  form.schedule_enabled = s.schedule_enabled
  form.schedule_cron = s.schedule_cron || '0 3 * * *'
  form.max_backups = s.max_backups
  form.retention_days = s.retention_days
  form.volumes = [...(s.volumes ?? [])]
  secretSet.value = s.s3_secret_set
}

async function load() {
  if (!ent.has.value) return
  loading.value = true
  try {
    const [s, b, v] = await Promise.all([
      platformBackupApi.getSettings(),
      platformBackupApi.list(0, 50),
      platformBackupApi.volumes().catch(() => null),
    ])
    applySettings(s.data.data)
    backups.value = b.data.data ?? []
    volumes.value = v?.data.data ?? []
  } catch (e) {
    notify.apiError(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  licenseStore.load()
  load()
})

const payload = computed<PlatformBackupSettingsPayload>(() => ({
  ...form,
  // Omit the secret when left blank so the stored one is preserved.
  s3_secret_key: form.s3_secret_key ? form.s3_secret_key : undefined,
}))

async function save() {
  if (form.s3_enabled && !form.s3_bucket.trim()) {
    notify.error('An S3 bucket is required when S3 is enabled')
    return
  }
  saving.value = true
  try {
    const res = await platformBackupApi.updateSettings(payload.value)
    applySettings(res.data.data)
    notify.success('Platform backup settings saved')
  } catch (e) {
    notify.apiError(e)
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  try {
    await platformBackupApi.testSettings(payload.value)
    notify.success('Platform backup settings look valid')
  } catch (e) {
    notify.apiError(e)
  } finally {
    testing.value = false
  }
}

async function runBackup() {
  if (!backupDB.value && backupVolumes.value.length === 0) {
    notify.error('Select the database and/or at least one volume')
    return
  }
  running.value = true
  try {
    await platformBackupApi.create({ database: backupDB.value, volumes: backupVolumes.value })
    notify.success('Backup started')
    backupVolumes.value = []
    await load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    running.value = false
  }
}

const pendingRestore = ref<PlatformBackup | null>(null)
const restoreMessage = computed(() => {
  const b = pendingRestore.value
  if (!b) return ''
  const what =
    b.subject === 'database'
      ? 'the control-plane database (overwrites the running database in place)'
      : `volume "${b.volume_name}" (overwrites its contents)`
  return (
    `Restore ${what}? This is destructive. Put the platform in maintenance mode first. ` +
    `You also need the original MIABI_ENCRYPTION_KEY to decrypt restored secrets.`
  )
})
async function restore() {
  const b = pendingRestore.value
  if (!b) return
  pendingRestore.value = null
  restoringId.value = b.id
  try {
    await platformBackupApi.restore(b.id)
    notify.success('Restore completed')
    await load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    restoringId.value = null
  }
}

const pendingDelete = ref<PlatformBackup | null>(null)
const deleting = ref(false)
async function remove() {
  const b = pendingDelete.value
  if (!b) return
  deleting.value = true
  try {
    await platformBackupApi.remove(b.id)
    notify.success('Backup deleted')
    pendingDelete.value = null
    await load()
  } catch (e) {
    notify.apiError(e)
  } finally {
    deleting.value = false
  }
}

async function download(b: PlatformBackup) {
  try {
    const res = await platformBackupApi.download(b.id)
    const url = URL.createObjectURL(res.data as Blob)
    const a = document.createElement('a')
    a.href = url
    a.download = b.filename || `platform-backup-${b.id}.sql.gz`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    notify.apiError(e)
  }
}

function canDownload(b: PlatformBackup): boolean {
  return b.subject === 'database' && b.destination === 'local' && b.status === 'completed' && !!b.filename
}
function canRestore(b: PlatformBackup): boolean {
  return b.status === 'completed' && !!b.filename
}

function statusBadge(s: string): string {
  switch (s) {
    case 'completed':
      return 'badge-success'
    case 'failed':
      return 'badge-danger'
    case 'running':
      return 'badge-info'
    default:
      return 'badge-warning'
  }
}
function fmtDate(s?: string | null): string {
  return s ? new Date(s).toLocaleString() : '—'
}
function fmtSize(n: number): string {
  if (!n) return '—'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(1)} ${u[i]}`
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Platform Backup</h1>
        <p class="text-muted">
          Disaster recovery for Miabi itself — back up and restore the control-plane database and
          platform volumes. Separate from per-workspace backups.
        </p>
      </div>
    </div>

    <!-- Locked (Community / not entitled) -->
    <div v-if="!ent.has.value" class="card">
      <div class="card-body locked">
        <span class="mdi mdi-lock-outline"></span>
        <div>
          <p>
            Platform backup is an Enterprise feature. Back up Miabi's own database and platform
            volumes to S3, on demand or on a schedule, and restore them for disaster recovery.
          </p>
          <router-link to="/admin/license" class="btn btn-secondary btn-sm">Manage license</router-link>
        </div>
      </div>
    </div>

    <template v-else>
      <!-- DR key reminder -->
      <div class="card dr-note">
        <div class="card-body">
          <span class="mdi mdi-key-alert-outline"></span>
          <div>
            <strong>Keep your encryption key safe.</strong>
            A database backup contains ciphertext (workspace keys, provider credentials) encrypted
            under <code>MIABI_ENCRYPTION_KEY</code>. You must preserve that key out-of-band — a
            restore onto a fresh server is useless without it. Store S3 off-box for true DR.
          </div>
        </div>
      </div>

      <!-- Settings -->
      <div class="card">
        <div class="card-header"><h3>Destination &amp; schedule</h3></div>
        <div class="card-body">
          <div class="form-group">
            <label class="check-row"><input v-model="form.s3_enabled" type="checkbox" /> Store backups in S3 (recommended)</label>
            <span class="form-hint">Volume backups require S3 — they have no local destination.</span>
          </div>

          <template v-if="form.s3_enabled">
            <div class="grid2">
              <div class="form-group">
                <label class="form-label">Endpoint</label>
                <input v-model="form.s3_endpoint" class="form-input" placeholder="https://s3.amazonaws.com (blank = AWS)" />
              </div>
              <div class="form-group">
                <label class="form-label">Bucket</label>
                <input v-model="form.s3_bucket" class="form-input" placeholder="miabi-platform-backups" />
              </div>
              <div class="form-group">
                <label class="form-label">Region</label>
                <input v-model="form.s3_region" class="form-input" placeholder="us-east-1" />
              </div>
              <div class="form-group">
                <label class="form-label">Access key</label>
                <input v-model="form.s3_access_key" class="form-input" autocomplete="off" />
              </div>
              <div class="form-group">
                <label class="form-label">Secret key</label>
                <input
                  v-model="form.s3_secret_key"
                  class="form-input"
                  type="password"
                  autocomplete="off"
                  :placeholder="secretSet ? '••••• (set — leave blank to keep)' : ''"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Database path prefix</label>
                <input v-model="form.database_backup_path" class="form-input" placeholder="database" />
              </div>
              <div class="form-group">
                <label class="form-label">Volume path prefix</label>
                <input v-model="form.volume_backup_path" class="form-input" placeholder="volumes" />
              </div>
            </div>
            <div class="form-group">
              <label class="check-row"><input v-model="form.s3_use_ssl" type="checkbox" /> Use SSL</label>
              <label class="check-row"><input v-model="form.s3_force_path_style" type="checkbox" /> Force path-style addressing (MinIO)</label>
            </div>
          </template>

          <hr class="sep" />

          <div class="form-group">
            <label class="check-row"><input v-model="form.schedule_enabled" type="checkbox" /> Run on a schedule</label>
          </div>
          <div v-if="form.schedule_enabled" class="grid2">
            <div class="form-group">
              <label class="form-label">Cron</label>
              <input v-model="form.schedule_cron" class="form-input" placeholder="0 3 * * *" />
              <span class="form-hint">Standard 5-field cron. Backs up the DB plus every selected volume.</span>
            </div>
            <div class="form-group">
              <label class="form-label">Keep at most (max backups)</label>
              <input v-model.number="form.max_backups" class="form-input" type="number" min="0" />
            </div>
            <div class="form-group">
              <label class="form-label">Retention (days)</label>
              <input v-model.number="form.retention_days" class="form-input" type="number" min="0" />
            </div>
          </div>

          <div v-if="volumes.length" class="form-group">
            <label class="form-label">Platform volumes in scheduled backups</label>
            <div class="vol-list">
              <label v-for="v in volumes" :key="v.name" class="check-row">
                <input v-model="form.volumes" type="checkbox" :value="v.name" />
                <span class="mono">{{ v.name }}</span>
                <span v-if="v.role" class="badge badge-info">{{ v.role }}</span>
              </label>
            </div>
          </div>
        </div>
        <div class="card-footer actions-right">
          <button class="btn btn-secondary" :disabled="testing || !form.s3_enabled" @click="test">
            {{ testing ? 'Testing…' : 'Test S3' }}
          </button>
          <button class="btn btn-primary" :disabled="saving" @click="save">
            {{ saving ? 'Saving…' : 'Save settings' }}
          </button>
        </div>
      </div>

      <!-- Back up now -->
      <div class="card">
        <div class="card-header"><h3>Back up now</h3></div>
        <div class="card-body">
          <label class="check-row"><input v-model="backupDB" type="checkbox" /> Control-plane database</label>
          <div v-if="volumes.length" class="vol-list mt-2">
            <label v-for="v in volumes" :key="v.name" class="check-row">
              <input v-model="backupVolumes" type="checkbox" :value="v.name" :disabled="!form.s3_enabled" />
              <span class="mono">{{ v.name }}</span>
              <span v-if="v.role" class="badge badge-info">{{ v.role }}</span>
            </label>
            <p v-if="!form.s3_enabled" class="form-hint">Enable S3 above to back up volumes.</p>
          </div>
        </div>
        <div class="card-footer actions-right">
          <button class="btn btn-primary" :disabled="running" @click="runBackup">
            <span class="mdi mdi-cloud-upload-outline"></span> {{ running ? 'Starting…' : 'Back up now' }}
          </button>
        </div>
      </div>

      <!-- History -->
      <div class="card">
        <div class="card-header"><h3>History</h3></div>
        <div v-if="loading && backups.length === 0" class="card-body"><span class="spinner"></span></div>
        <div v-else-if="backups.length === 0" class="empty-state">
          <span class="mdi mdi-database-clock-outline" style="font-size: 44px; color: var(--text-muted)"></span>
          <h3>No platform backups yet</h3>
          <p class="text-muted">Run a backup above to get started.</p>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr><th>Subject</th><th>Status</th><th>Trigger</th><th>Destination</th><th>Size</th><th>Created</th><th></th></tr>
            </thead>
            <tbody>
              <tr v-for="b in backups" :key="b.id">
                <td>
                  <span class="cell-title">{{ b.subject === 'database' ? 'Database' : 'Volume' }}</span>
                  <span v-if="b.volume_name" class="cell-sub mono">{{ b.volume_name }}</span>
                </td>
                <td>
                  <span class="badge badge-dot" :class="statusBadge(b.status)" :title="b.error || ''">{{ b.status }}</span>
                </td>
                <td class="text-muted">{{ b.trigger }}</td>
                <td class="text-muted">{{ b.destination }}</td>
                <td class="text-muted">{{ fmtSize(b.size_bytes) }}</td>
                <td class="text-muted">{{ fmtDate(b.created_at) }}</td>
                <td class="text-right actions" @click.stop>
                  <button
                    v-if="canRestore(b)"
                    class="btn-icon btn-icon-muted"
                    title="Restore"
                    aria-label="Restore"
                    :disabled="restoringId === b.id"
                    @click="pendingRestore = b"
                  >
                    <span class="mdi mdi-backup-restore"></span>
                  </button>
                  <button v-if="canDownload(b)" class="btn-icon btn-icon-muted" title="Download" aria-label="Download" @click="download(b)">
                    <span class="mdi mdi-download"></span>
                  </button>
                  <button class="btn-icon btn-icon-danger" title="Delete" aria-label="Delete" @click="pendingDelete = b"><span class="mdi mdi-delete"></span></button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <ConfirmDialog
      :open="!!pendingRestore"
      title="Restore platform backup?"
      :message="restoreMessage"
      confirm-label="Restore"
      variant="danger"
      :busy="restoringId !== null"
      @confirm="restore"
      @cancel="pendingRestore = null"
    />

    <ConfirmDialog
      :open="!!pendingDelete"
      title="Delete platform backup?"
      message="Delete this platform backup?"
      confirm-label="Delete"
      variant="danger"
      :busy="deleting"
      @confirm="remove"
      @cancel="pendingDelete = null"
    />
  </div>
</template>

<style scoped>
.locked {
  display: flex;
  align-items: center;
  gap: 14px;
}
.locked .mdi {
  font-size: 28px;
  color: var(--text-muted);
}
.locked p {
  margin: 0 0 8px;
  max-width: 64ch;
  color: var(--text-secondary, var(--text-muted));
}
.dr-note .card-body {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.dr-note .mdi {
  font-size: 24px;
  color: var(--warning, #d97706);
}
.dr-note code {
  font-family: var(--font-mono, monospace);
  font-size: 12px;
}
.grid2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.check-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.vol-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mono {
  font-family: var(--font-mono, monospace);
  font-size: 12px;
}
.sep {
  border: none;
  border-top: 1px solid var(--border, #2a2a2a);
  margin: 16px 0;
}
.actions-right {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
}
.cell-sub {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
}
</style>
