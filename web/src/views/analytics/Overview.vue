<script setup lang="ts">
import { useRouter } from 'vue-router'
import Sparkline from '@/components/Sparkline.vue'
import AnalyticsShell from './AnalyticsShell.vue'
import StatTile from './StatTile.vue'
import RequestsChart from './RequestsChart.vue'
import StatusPie from './StatusPie.vue'
import type { AnalyticsReport } from '@/api/analytics'
import { fmtNum, fmtBytes, fmtMs, fmtPct, delta, countryFlag, countryName } from './format'

const router = useRouter()

// Rate of n over the range's total requests (0..1).
function rate(n: number, total: number): number {
  return total > 0 ? n / total : 0
}
// Quiet buckets carry no latency, so they'd drag the sparkline to zero.
function latency(r: AnalyticsReport): number[] {
  return r.series.filter((p) => p.requests > 0).map((p) => p.p95_latency_ms)
}
const topCountries = (r: AnalyticsReport) => r.web.top_countries.slice(0, 5)
</script>

<template>
  <AnalyticsShell v-slot="{ report }">
    <div class="a-grid">
      <StatTile label="Requests" icon="mdi-swap-horizontal" :value="fmtNum(report.totals.requests)"
        :delta="delta(report.totals.requests, report.compare?.requests)" />
      <StatTile label="Unique visitors" icon="mdi-account-multiple-outline" :value="fmtNum(report.totals.unique_visitors)"
        :delta="delta(report.totals.unique_visitors, report.compare?.unique_visitors)" />
      <StatTile label="Data served" icon="mdi-swap-vertical" :value="fmtBytes(report.totals.bytes_in + report.totals.bytes_out)"
        :delta="delta(report.totals.bytes_out, report.compare?.bytes_out)">
        <template #sub>
          <span class="mdi mdi-arrow-down-thin io-in" title="Inbound (requests received)"></span>{{ fmtBytes(report.totals.bytes_in) }} in
          <span class="io-dot">·</span>
          <span class="mdi mdi-arrow-up-thin io-out" title="Outbound (responses sent)"></span>{{ fmtBytes(report.totals.bytes_out) }} out
        </template>
      </StatTile>
      <StatTile label="Server errors (5xx)" icon="mdi-alert-octagon-outline"
        :value="fmtPct(rate(report.status.s5xx, report.totals.requests))"
        :danger="rate(report.status.s5xx, report.totals.requests) >= 0.01" invert
        :sub="`Client errors (4xx): ${fmtPct(rate(report.status.s4xx, report.totals.requests))}`" />
    </div>

    <div class="card">
      <div class="a-card-header">
        <h3>Requests over time</h3>
        <span class="a-muted">{{ fmtNum(report.totals.requests) }} requests · per {{ report.granularity }}</span>
      </div>
      <div class="card-body">
        <RequestsChart v-if="report.series.length" :series="report.series" :granularity="report.granularity" />
        <p v-else class="a-muted">Not enough data points to plot.</p>
      </div>
    </div>

    <div class="two-col">
      <div class="card">
        <div class="a-card-header"><h3>Status codes</h3></div>
        <div class="card-body">
          <StatusPie :status="report.status" />
          <div class="perf-inline">
            <span>Avg {{ fmtMs(report.totals.avg_latency_ms) }}</span>
            <span>p95 {{ fmtMs(report.totals.p95_latency_ms) }}</span>
            <Sparkline v-if="latency(report).length > 1" :values="latency(report)" :width="180" :height="34" stroke="var(--warning-600)" />
          </div>
        </div>
      </div>

      <div class="card">
        <div class="a-card-header">
          <h3>Top countries</h3>
          <a class="a-muted" href="#" @click.prevent="router.push('/analytics/http')">View map →</a>
        </div>
        <div class="card-body">
          <div v-for="c in topCountries(report)" :key="c.label" class="brow">
            <span class="brow-label">{{ countryFlag(c.label) }}&nbsp; {{ countryName(c.label) }}</span>
            <span class="brow-track"><span class="brow-fill" :style="{ width: (c.count / (topCountries(report)[0]?.count || 1)) * 100 + '%' }"></span></span>
            <span class="brow-count">{{ fmtNum(c.count) }}</span>
          </div>
          <p v-if="!report.web.top_countries.length" class="a-muted">Country data needs the GeoIP database on the gateway.</p>
        </div>
      </div>
    </div>
  </AnalyticsShell>
</template>

<style scoped>
.two-col { display: grid; grid-template-columns: repeat(auto-fit, minmax(340px, 1fr)); gap: 14px; }
.perf-inline { display: flex; align-items: center; gap: 16px; margin-top: 14px; font-size: 13px; color: var(--text-secondary); }
.io-in { color: var(--success-600); font-size: 15px; vertical-align: -2px; }
.io-out { color: #2563eb; font-size: 15px; vertical-align: -2px; }
.io-dot { margin: 0 3px; color: var(--text-muted); }
</style>
