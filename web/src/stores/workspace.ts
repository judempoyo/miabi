import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { workspaceApi, type CreateWorkspaceInput } from '@/api/workspaces'
import type { Workspace, WorkspaceRole } from '@/api/types'

const STORAGE_KEY = 'mb_workspace_id'

export const useWorkspaceStore = defineStore('workspace', () => {
  const workspaces = ref<Workspace[]>([])
  const currentWorkspaceId = ref<number | null>(
    localStorage.getItem(STORAGE_KEY) ? Number(localStorage.getItem(STORAGE_KEY)) : null,
  )
  const loaded = ref(false)

  const currentWorkspace = computed(
    () => workspaces.value.find((w) => w.id === currentWorkspaceId.value) ?? null,
  )

  const currentRole = computed<WorkspaceRole | null>(() => currentWorkspace.value?.role ?? null)
  const contextLabel = computed(
    () => currentWorkspace.value?.display_name || currentWorkspace.value?.name || 'Select workspace',
  )
  const isWorkspaceContext = computed(() => currentWorkspace.value !== null)
  const isWorkspaceAdmin = computed(
    () => currentRole.value === 'owner' || currentRole.value === 'admin',
  )
  const isWorkspaceOwner = computed(() => currentRole.value === 'owner')
  // Roles allowed to mutate resources (deploy, create, edit).
  const canEdit = computed(() => {
    const role = currentRole.value
    return role === 'owner' || role === 'admin' || role === 'developer'
  })

  function setWorkspace(id: number | null) {
    currentWorkspaceId.value = id
    if (id) localStorage.setItem(STORAGE_KEY, String(id))
    else localStorage.removeItem(STORAGE_KEY)
  }

  async function fetchWorkspaces() {
    const res = await workspaceApi.list()
    workspaces.value = res.data.data ?? []
    // Keep the active selection valid; otherwise land on the first workspace.
    const stillValid =
      currentWorkspaceId.value && workspaces.value.some((w) => w.id === currentWorkspaceId.value)
    if (!stillValid) {
      setWorkspace(workspaces.value[0]?.id ?? null)
    }
    loaded.value = true
  }

  async function create(input: CreateWorkspaceInput | string) {
    const res = await workspaceApi.create(input)
    await fetchWorkspaces()
    setWorkspace(res.data.data.id)
    return res.data.data
  }

  function clear() {
    workspaces.value = []
    setWorkspace(null)
    loaded.value = false
  }

  return {
    workspaces,
    currentWorkspaceId,
    currentWorkspace,
    currentRole,
    contextLabel,
    isWorkspaceContext,
    isWorkspaceAdmin,
    isWorkspaceOwner,
    canEdit,
    loaded,
    setWorkspace,
    fetchWorkspaces,
    create,
    clear,
  }
})
