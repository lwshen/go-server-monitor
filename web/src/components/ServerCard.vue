<script setup lang="ts">
// Dashboard grid card for a single server. All metrics come straight from the
// store's reactive `server` object, so WebSocket merges re-render this live.
// Numeric values render through the shared formatters and use the mono font.

import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Server } from '@/types'
import { bytesRate, mib, ms, pct, pctValue, sinceSeconds } from '@/utils/format'

const props = defineProps<{ server: Server }>()

const router = useRouter()
const { t } = useI18n()

/** Prefer alias for display, fall back to the canonical name. */
const displayName = computed(() => props.server.alias?.trim() || props.server.name)

/** location comes off the server record; fall back to ip_info city/country. */
const displayLocation = computed(() => {
  const s = props.server
  if (s.location?.trim()) return s.location
  const ip = s.ip_info
  if (!ip) return ''
  return [ip.city, ip.country].filter(Boolean).join(', ')
})

const m = computed(() => props.server.latest_metrics)

const cpuValue = computed(() => {
  const c = m.value?.cpu
  return typeof c === 'number' && c >= 0 ? c : null
})

const memPct = computed(() => pctValue(m.value?.memory_used, m.value?.memory_total))

/** Map a 0..100 utilisation to the shared meter warn/alert classes. */
function meterClass(v: number | null): string {
  if (v === null) return ''
  if (v >= 90) return 'is-alert'
  if (v >= 75) return 'is-warn'
  return ''
}

/** Three-backbone + intl ping columns (CT / CU / CM / BD). */
const pings = computed(() => {
  const row = m.value
  return [
    { key: 'ct', value: row?.ping_ct ?? null },
    { key: 'cu', value: row?.ping_cu ?? null },
    { key: 'cm', value: row?.ping_cm ?? null },
    { key: 'bd', value: row?.ping_bd ?? null },
  ]
})

function open(): void {
  void router.push({ name: 'server-detail', params: { id: props.server.id } })
}
</script>

<template>
  <article
    class="card server-card"
    role="button"
    tabindex="0"
    :aria-label="displayName"
    @click="open"
    @keydown.enter="open"
    @keydown.space.prevent="open"
  >
    <!-- Header: status dot + name + group / location -->
    <header class="sc-head">
      <span
        class="status-dot"
        :class="server.online ? 'is-online' : 'is-offline'"
        :title="server.online ? t('common.online') : t('common.offline')"
      />
      <div class="sc-title">
        <span class="sc-name">{{ displayName }}</span>
        <span class="sc-sub dim">
          <span v-if="server.server_group">{{ server.server_group }}</span>
          <span v-if="server.server_group && displayLocation" class="sc-dot">·</span>
          <span v-if="displayLocation">{{ displayLocation }}</span>
        </span>
      </div>
      <span class="badge sc-status" :class="server.online ? 'is-online' : 'is-offline'">
        {{ server.online ? t('common.online') : t('common.offline') }}
      </span>
    </header>

    <!-- CPU -->
    <div class="sc-metric">
      <div class="row between sc-metric-head">
        <span class="eyebrow">{{ t('th.cpu') }}</span>
        <span class="metric-value">{{
          cpuValue === null ? '—' : Math.round(cpuValue) + '%'
        }}</span>
      </div>
      <div class="meter">
        <div
          class="meter-fill"
          :class="meterClass(cpuValue)"
          :style="{ width: (cpuValue ?? 0) + '%' }"
        />
      </div>
    </div>

    <!-- Memory -->
    <div class="sc-metric">
      <div class="row between sc-metric-head">
        <span class="eyebrow">{{ t('th.mem') }}</span>
        <span class="metric-value">
          {{ mib(m?.memory_used) }} / {{ mib(m?.memory_total) }}
          <span class="dim">({{ pct(m?.memory_used, m?.memory_total) }})</span>
        </span>
      </div>
      <div class="meter">
        <div
          class="meter-fill"
          :class="meterClass(memPct)"
          :style="{ width: (memPct ?? 0) + '%' }"
        />
      </div>
    </div>

    <!-- Network up / down -->
    <div class="sc-grid2">
      <div class="sc-cell">
        <span class="eyebrow">↑ {{ t('detail.upload') }}</span>
        <span class="metric-value">{{ bytesRate(m?.network_tx) }}</span>
      </div>
      <div class="sc-cell">
        <span class="eyebrow">↓ {{ t('detail.download') }}</span>
        <span class="metric-value">{{ bytesRate(m?.network_rx) }}</span>
      </div>
    </div>

    <!-- Multi-line ping -->
    <div class="sc-ping">
      <span class="eyebrow">{{ t('th.ping') }}</span>
      <div class="sc-ping-row">
        <span v-for="p in pings" :key="p.key" class="sc-ping-cell">
          <span class="sc-ping-label faint">{{ p.key.toUpperCase() }}</span>
          <span class="metric-value">{{ ms(p.value) }}</span>
        </span>
      </div>
    </div>

    <!-- Footer: last seen -->
    <footer class="sc-foot faint">
      {{ t('detail.lastUpdate') }}: {{ sinceSeconds(server.last_updated) }}
    </footer>
  </article>
</template>

<style scoped>
.server-card {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  cursor: pointer;
  padding: var(--sp-4);
}
.server-card:hover {
  border-color: var(--accent);
}

.sc-head {
  display: flex;
  align-items: flex-start;
  gap: var(--sp-2);
}
.sc-head .status-dot {
  margin-top: 5px;
}
.sc-title {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}
.sc-name {
  font-weight: 650;
  font-size: var(--fs-base);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sc-sub {
  font-size: var(--fs-xs);
  display: inline-flex;
  gap: var(--sp-1);
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sc-dot {
  opacity: 0.5;
}
.sc-status {
  flex: none;
}

.sc-metric {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
}
.sc-metric-head {
  align-items: baseline;
}
.sc-metric-head .metric-value {
  font-size: var(--fs-sm);
}

.sc-grid2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-2);
}
.sc-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: var(--sp-2);
}
.sc-cell .metric-value {
  font-size: var(--fs-sm);
}

.sc-ping {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
}
.sc-ping-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-1);
}
.sc-ping-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  padding: var(--sp-1) 2px;
  background: var(--surface-2);
  border-radius: var(--radius-sm);
}
.sc-ping-label {
  font-size: 0.625rem;
  letter-spacing: 0.04em;
}
.sc-ping-cell .metric-value {
  font-size: var(--fs-xs);
}

.sc-foot {
  font-size: var(--fs-xs);
  border-top: 1px solid var(--border);
  padding-top: var(--sp-2);
  margin-top: auto;
}
</style>
