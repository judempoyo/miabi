<script setup lang="ts">
import { computed } from 'vue'
import { fmtDelta } from './format'

// A headline metric tile with an optional period-over-period delta. `invert`
// flips the color meaning (up = bad, e.g. error rate / latency).
const props = defineProps<{
  label: string
  value: string
  sub?: string
  delta?: number | null
  invert?: boolean
  danger?: boolean
}>()

const dir = computed(() => {
  if (props.delta === null || props.delta === undefined) return 'flat'
  if (Math.abs(props.delta) < 0.001) return 'flat'
  return props.delta > 0 ? 'up' : 'down'
})
</script>

<template>
  <div class="a-tile">
    <div class="t-label">{{ label }}</div>
    <div class="t-value" :class="{ danger }">
      {{ value }}
      <span
        v-if="delta !== null && delta !== undefined && dir !== 'flat'"
        class="a-delta"
        :class="[dir, { invert }]"
      >{{ fmtDelta(delta) }}</span>
    </div>
    <div v-if="sub" class="t-sub">{{ sub }}</div>
  </div>
</template>
