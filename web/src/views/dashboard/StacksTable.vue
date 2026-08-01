<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { Stack } from '@/api/types'
import { fmtDate } from './format'

defineProps<{ stacks: Stack[] }>()

const router = useRouter()

function stackBadge(s: Stack): string {
  const total = s.status?.total ?? 0
  const running = s.status?.running ?? 0
  if (total === 0) return 'badge-neutral'
  if (running === total) return 'badge-success'
  if (running === 0) return 'badge-danger'
  return 'badge-warning'
}
</script>

<template>
  <div class="card">
    <div class="card-header">
      <h2>Stacks</h2>
      <button class="btn btn-ghost btn-sm" @click="router.push('/stacks')">View all</button>
    </div>
    <div class="table-wrapper">
      <table>
        <thead><tr><th>Stack</th><th>Apps</th><th>Created</th><th class="text-right">Status</th></tr></thead>
        <tbody>
          <tr v-for="s in stacks" :key="s.id" class="row-clickable" @click="router.push(`/stacks/${s.id}`)">
            <td>
              <div class="cell-id">
                <span class="avatar avatar-sm"><span class="mdi mdi-layers-outline" style="font-size: 14px"></span></span>
                <span class="cell-text"><span class="cell-title">{{ s.name }}</span></span>
              </div>
            </td>
            <td class="cell-sub">{{ s.app_count ?? 0 }}</td>
            <td class="cell-sub">{{ fmtDate(s.created_at) }}</td>
            <td class="text-right">
              <span class="badge badge-dot" :class="stackBadge(s)">
                {{ s.status?.running ?? 0 }}/{{ s.status?.total ?? 0 }} running
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
