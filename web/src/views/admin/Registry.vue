<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { registryApi, type RegistrySettingsPayload } from '@/api/registry'
import { useNotificationStore } from '@/stores/notification'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const notify = useNotificationStore()

const loading = ref(true)
const saving = ref(false)
const runningGc = ref(false)
const effectiveHost = ref('')
const secretSet = ref(false)
const s3Entitled = ref(false)

const volumeName = ref('mb-registry-data')

// Enablement and the hostname are fixed at boot from MIABI_REGISTRY_ENABLED /
// MIABI_REGISTRY_HOST. They are shown, never edited: the host is what every
// stored image reference is anchored to and what decides which workspace owns an
// image, so moving it under a running platform would strand those references.
const enabled = ref(false)
const hostSource = ref<'env' | 'stored' | 'base_domain' | 'unset'>('unset')

const hostOrigin = computed(() => {
  switch (hostSource.value) {
    case 'env':
      return 'Set by MIABI_REGISTRY_HOST.'
    case 'base_domain':
      return 'Derived from the external base domain (registry.<domain>).'
    case 'stored':
      return 'A value saved before the host became environment-only. Set MIABI_REGISTRY_HOST to move it.'
    default:
      return 'No usable hostname — set MIABI_REGISTRY_HOST or an external base domain.'
  }
})

const form = ref<RegistrySettingsPayload>({
  storage_type: 'filesystem',
  s3_endpoint: '',
  s3_bucket: '',
  s3_region: '',
  s3_access_key: '',
  s3_secret_key: '',
  s3_force_path_style: false,
  delete_enabled: false,
  per_workspace_quota_mb: 0,
})

async function load() {
  loading.value = true
  try {
    const s = (await registryApi.getSettings()).data.data
    effectiveHost.value = s.effective_host
    secretSet.value = s.s3_secret_set
    s3Entitled.value = s.s3_entitled
    volumeName.value = s.volume_name || 'mb-registry-data'
    enabled.value = s.enabled
    hostSource.value = s.host_source
    form.value = {
      storage_type: s.storage_type,
      s3_endpoint: s.s3_endpoint ?? '',
      s3_bucket: s.s3_bucket ?? '',
      s3_region: s.s3_region ?? '',
      s3_access_key: s.s3_access_key ?? '',
      s3_secret_key: '', // never returned; blank keeps the stored secret
      s3_force_path_style: s.s3_force_path_style,
      delete_enabled: s.delete_enabled,
      per_workspace_quota_mb: s.per_workspace_quota_mb,
    }
  } catch (e) {
    notify.apiError(e)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (form.value.storage_type === 's3' && !s3Entitled.value) {
    notify.error('S3/MinIO storage requires an Enterprise license. Use local storage or upgrade.')
    return
  }
  if (form.value.storage_type === 's3' && !form.value.s3_bucket.trim()) {
    notify.error('An S3 bucket is required for the S3 storage driver')
    return
  }
  saving.value = true
  try {
    const s = (await registryApi.updateSettings(form.value)).data.data
    effectiveHost.value = s.effective_host
    hostSource.value = s.host_source
    secretSet.value = s.s3_secret_set
    form.value.s3_secret_key = ''
    notify.success('Registry settings saved')
  } catch (e) {
    notify.apiError(e)
  } finally {
    saving.value = false
  }
}

const showGcConfirm = ref(false)
async function runGc() {
  showGcConfirm.value = false
  runningGc.value = true
  try {
    const res = (await registryApi.runGc()).data.data
    notify.success(res.message || 'Garbage collection complete')
  } catch (e) {
    notify.apiError(e)
  } finally {
    runningGc.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Container Registry</h1>
    </div>

    <div v-if="loading" class="card"><div class="card-body"><span class="spinner"></span></div></div>

    <div v-else class="card" style="max-width: 720px">
      <div class="card-header">
        <div>
          <h2>Built-in registry</h2>
          <p class="text-muted text-sm" style="margin: 4px 0 0">
            A first-party, multi-tenant Docker registry. Members push & pull with
            <code>docker login {{ effectiveHost || 'registry.&lt;domain&gt;' }}</code>.
          </p>
        </div>
      </div>
      <div class="card-body">
        <label class="toggle-row is-locked">
          <input :checked="enabled" type="checkbox" disabled />
          <span>Enable the registry (runs the container and seeds its gateway route)</span>
        </label>
        <small class="form-hint">
          Set by <code>MIABI_REGISTRY_ENABLED={{ enabled ? 'true' : 'false' }}</code>. Fixed by the platform —
          change it and restart Miabi.
        </small>

        <div class="form-group" style="margin-top: 16px">
          <label class="form-label">Host</label>
          <input :value="effectiveHost" class="form-input mono" placeholder="—" style="max-width: 420px" disabled />
          <small class="form-hint">
            Public hostname for docker login. {{ hostOrigin }} Fixed by the platform: every image reference
            Miabi stores is anchored to this hostname, and matching against it is how a workspace's own
            images are told apart from another's — so it cannot move while apps are running.
          </small>
        </div>

        <div class="form-group">
          <label class="form-label">Storage driver</label>
          <select v-model="form.storage_type" class="form-select" style="max-width: 280px">
            <option value="filesystem">Local volume (filesystem)</option>
            <option value="s3" :disabled="!s3Entitled">S3 / MinIO{{ s3Entitled ? '' : ' — Enterprise' }}</option>
          </select>
          <small class="form-hint">
            Switching drivers recreates the registry; <strong>data does not migrate</strong> between drivers.
            <template v-if="!s3Entitled"> S3/MinIO storage requires an Enterprise license; local storage is free.</template>
          </small>
        </div>

        <div v-if="form.storage_type === 'filesystem'" class="form-group">
          <label class="form-label">Data volume</label>
          <input :value="volumeName" class="form-input mono" style="max-width: 320px" disabled />
          <small class="form-hint">The managed volume images are stored in. Fixed by the platform.</small>
        </div>

        <fieldset v-else class="s3-fields">
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label">Bucket</label>
              <input v-model="form.s3_bucket" class="form-input" placeholder="my-registry" />
            </div>
            <div class="form-group">
              <label class="form-label">Region</label>
              <input v-model="form.s3_region" class="form-input" placeholder="us-east-1" />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Endpoint <span class="text-muted">(optional, S3-compatible / MinIO)</span></label>
            <input v-model="form.s3_endpoint" class="form-input" placeholder="https://s3.amazonaws.com" />
          </div>
          <div class="form-grid">
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
                autocomplete="new-password"
                :placeholder="secretSet ? '••••• (set — leave blank to keep)' : ''"
              />
            </div>
          </div>
          <label class="toggle-row">
            <input v-model="form.s3_force_path_style" type="checkbox" />
            <span>Force path-style URLs (MinIO and some S3-compatible stores)</span>
          </label>
        </fieldset>

        <div class="form-group" style="margin-top: 8px">
          <label class="form-label">Per-workspace quota (MB)</label>
          <input v-model.number="form.per_workspace_quota_mb" type="number" min="0" class="form-input" style="max-width: 200px" />
          <small class="form-hint">0 = unlimited.</small>
        </div>

        <label class="toggle-row">
          <input v-model="form.delete_enabled" type="checkbox" />
          <span>Enable tag deletion &amp; garbage collection</span>
        </label>

        <div class="actions">
          <button class="btn btn-primary" :disabled="saving" @click="save">
            {{ saving ? 'Saving…' : 'Save settings' }}
          </button>
          <button
            v-if="enabled && form.delete_enabled"
            class="btn btn-secondary"
            :disabled="runningGc"
            @click="showGcConfirm = true"
          >
            {{ runningGc ? 'Collecting…' : 'Run garbage collection' }}
          </button>
        </div>
      </div>
    </div>

    <ConfirmDialog
      :open="showGcConfirm"
      title="Run garbage collection"
      :message="`Run garbage collection? The registry switches to read-only (pulls keep working, pushes pause) while it reclaims space.`"
      confirm-label="Run garbage collection"
      variant="primary"
      :busy="runningGc"
      @confirm="runGc"
      @cancel="showGcConfirm = false"
    />
  </div>
</template>

<style scoped>
.actions { display: flex; gap: 10px; margin-top: 20px; flex-wrap: wrap; }
.toggle-row { display: flex; align-items: center; gap: 8px; cursor: pointer; color: var(--text-primary); }
.toggle-row input { width: auto; margin: 0; }
.s3-fields { border: 0; padding: 0; margin: 0 0 8px; min-width: 0; }
.form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px 16px; }
.form-hint { display: block; font-size: 12px; color: var(--text-muted); margin-top: 4px; }
/* A locked toggle is not clickable, so it must not present as one. */
.toggle-row.is-locked { cursor: default; color: var(--text-muted); }
.text-muted { color: var(--text-muted); }
.text-sm { font-size: 13px; }
.mono { font-family: monospace; }
</style>
