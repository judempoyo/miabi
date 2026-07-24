<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { nodesApi, type PlaceableNode } from '@/api/nodes'

// v-model is the chosen server id (0 = local manager node).
const props = defineProps<{ modelValue: number; label?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const nodes = ref<PlaceableNode[]>([])

onMounted(async () => {
  try {
    nodes.value = (await nodesApi.placeable()).data.data ?? []
  } catch {
    nodes.value = []
  }
})

// Only show the picker when there's an actual choice (a remote node exists).
const hasChoice = computed(() => nodes.value.length > 1)

const value = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', Number(v)),
})

// A node accepts new placements when it's local, or online and not cordoned.
function placeable(n: PlaceableNode): boolean {
  return n.is_local || (n.online && !n.cordoned)
}
function optionLabel(n: PlaceableNode): string {
  if (n.is_local) return `${n.name} (manager)`
  if (!n.online) return `${n.name} — offline`
  if (n.cordoned) return `${n.name} — cordoned`
  const mode = n.connectivity === 'edge-gateway' ? 'edge gateway' : 'port forward'
  return `${n.name} (${mode})`
}
</script>

<template>
  <div v-if="hasChoice" class="form-group">
    <label class="form-label">{{ label || 'Node' }}</label>
    <select v-model="value" class="form-select" :aria-label="label || 'Node'">
      <option v-for="n in nodes" :key="n.id" :value="n.is_local ? 0 : n.id" :disabled="!placeable(n)">
        {{ optionLabel(n) }}
      </option>
    </select>
    <p class="form-hint">A database or volume an app uses must be on the same node.</p>
  </div>
</template>

<style scoped>
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
</style>
