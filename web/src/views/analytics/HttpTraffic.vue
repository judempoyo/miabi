<script setup lang="ts">
import AnalyticsShell from './AnalyticsShell.vue'
import Breakdown from './Breakdown.vue'
import StatusPie from './StatusPie.vue'
import WorldMap from '@/components/WorldMap.vue'
import { fmtNum } from './format'

const GEOIP_DOCS = 'https://docs.miabi.io/docs/operations/analytics#geoip-database'
</script>

<template>
  <AnalyticsShell v-slot="{ report }">
    <div class="card">
      <div class="a-card-header">
        <h3>Requests by country</h3>
        <span class="a-muted">{{ fmtNum(report.totals.requests) }} requests · {{ report.web.top_countries.length }} countries</span>
      </div>
      <div class="card-body">
        <WorldMap v-if="report.web.top_countries.length" :countries="report.web.top_countries" />
        <!-- Miabi ships no GeoIP database (licensing — see docs), so for most installs this
             empty state IS the setup instructions. Keep it actionable: the path and the link
             are the whole point. -->
        <div v-else class="empty-state">
          <h3>No country data yet</h3>
          <p>
            Resolving countries needs a GeoIP database on the gateway. Put one at
            <code>/etc/miabi/country.mmdb</code> and restart the gateway.
          </p>
          <p>
            <a :href="GEOIP_DOCS" target="_blank" rel="noopener">Where to get one →</a>
          </p>
        </div>
      </div>
    </div>

    <div class="break-grid">
      <Breakdown title="Top countries" :items="report.web.top_countries" kind="country"
        empty-hint="Needs a GeoIP database at /etc/miabi/country.mmdb — see the map above." />
      <Breakdown title="HTTP methods" :items="report.web.top_methods" />

      <div class="card">
        <div class="a-card-header"><h3>Status codes</h3></div>
        <div class="card-body">
          <StatusPie :status="report.status" />
        </div>
      </div>

      <Breakdown title="Top paths" :items="report.web.top_paths" />
      <Breakdown title="Referrers" :items="report.web.top_referrers" />
    </div>
  </AnalyticsShell>
</template>
