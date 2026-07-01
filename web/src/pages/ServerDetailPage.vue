<script setup lang="ts">
// Single-server detail. Header (name/location/type/IP/system), a realtime metric
// panel fed by a per-server WebSocket, and a range-selectable history section
// rendered with MetricsChart (category x-axis, no date adapter).

import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import axios from 'axios'
import { getHistory, getServer } from '@/services/api'
import type {
  HistoryPoint,
  HistoryRange,
  IpInfo,
  MetricsRow,
  Server,
  SysInfo,
  WsFrame,
} from '@/types'
import { WSManager } from '@/services/ws'
import { useI18n } from 'vue-i18n'
import MetricsChart, { type ChartSeries } from '@/components/MetricsChart.vue'
import {
  bytes,
  bytesRate,
  dash,
  dateTime,
  loadAvg,
  loss,
  mib,
  ms,
  pct,
  pctValue,
  sinceSeconds,
  uptime,
} from '@/utils/format'

const props = defineProps<{ id: string }>()
const { t } = useI18n()

const RANGES: HistoryRange[] = ['1h', '6h', '24h', '7d', '30d', '180d']

// ---- Server header + live snapshot ---------------------------------------
const server = ref<Server | null>(null)
const loading = ref(true)
const notFound = ref(false)
const errored = ref(false)

// Live metric snapshot merged from the WebSocket; seeded from latest_metrics.
const metrics = reactive<Partial<MetricsRow>>({})
const lastUpdated = ref(0)
const online = ref(false)

const sys = computed<SysInfo | undefined>(() => server.value?.sys_info)
const ip = computed<IpInfo | undefined>(() => server.value?.ip_info)

// ---- WebSocket (scope = this server id) ----------------------------------
const ws = new WSManager()

function applyMetricData(data: Partial<MetricsRow>, ts?: number): void {
  Object.assign(metrics, data)
  if (ts) {
    metrics.timestamp = ts
    lastUpdated.value = ts
  }
  online.value = true
}

function onFrame(frame: WsFrame): void {
  if (frame.type === 'update' && frame.serverId === props.id && frame.data) {
    applyMetricData(frame.data, frame.ts)
  } else if (frame.type === 'batchUpdate' && frame.updates) {
    const mine = frame.updates.find((u) => u.serverId === props.id)
    if (mine?.samples?.length) {
      const newest = mine.samples.reduce((a, b) => (b.ts > a.ts ? b : a))
      applyMetricData(newest.data, newest.ts)
    }
  }
}

// ---- Load the server + seed the live panel -------------------------------
async function loadServer(id: string): Promise<void> {
  loading.value = true
  notFound.value = false
  errored.value = false
  try {
    const s = await getServer(id)
    server.value = s
    online.value = s.online
    lastUpdated.value = s.last_updated
    for (const k of Object.keys(metrics)) delete (metrics as Record<string, unknown>)[k]
    if (s.latest_metrics) Object.assign(metrics, s.latest_metrics)
    ws.connect(id, onFrame)
  } catch (e) {
    server.value = null
    if (axios.isAxiosError(e) && e.response?.status === 404) notFound.value = true
    else errored.value = true
  } finally {
    loading.value = false
  }
}

// ---- History -------------------------------------------------------------
const range = ref<HistoryRange>('1h')
const samples = ref<HistoryPoint[]>([])
const historyLoading = ref(false)

async function loadHistory(id: string, r: HistoryRange): Promise<void> {
  historyLoading.value = true
  try {
    const res = await getHistory(id, r)
    samples.value = res.samples
  } catch {
    samples.value = []
  } finally {
    historyLoading.value = false
  }
}

function setRange(r: HistoryRange): void {
  if (r === range.value) return
  range.value = r
  void loadHistory(props.id, r)
}

// ---- Chart series --------------------------------------------------------
const usageSeries = computed<ChartSeries[]>(() => [
  {
    key: 'cpu',
    label: t('detail.chartCpu'),
    colorVar: '--accent',
    unit: '%',
    pick: (p) => p.cpu,
  },
  {
    key: 'mem',
    label: t('detail.chartMem'),
    colorVar: '--warn',
    unit: '%',
    pick: (p) => pctValue(p.memory_used, p.memory_total),
  },
])

const netSeries = computed<ChartSeries[]>(() => [
  {
    key: 'rx',
    label: t('detail.chartRx'),
    colorVar: '--accent',
    unit: ' B/s',
    pick: (p) => p.network_rx,
  },
  {
    key: 'tx',
    label: t('detail.chartTx'),
    colorVar: '--online',
    unit: ' B/s',
    pick: (p) => p.network_tx,
  },
])

const pingSeries = computed<ChartSeries[]>(() => [
  { key: 'ct', label: 'CT', colorVar: '--accent', unit: ' ms', pick: (p) => p.ping_ct },
  { key: 'cu', label: 'CU', colorVar: '--warn', unit: ' ms', pick: (p) => p.ping_cu },
  { key: 'cm', label: 'CM', colorVar: '--online', unit: ' ms', pick: (p) => p.ping_cm },
  { key: 'bd', label: 'BD', colorVar: '--offline', unit: ' ms', pick: (p) => p.ping_bd },
])

// A history series has any usable value for a metric key set.
const hasNet = computed(() =>
  samples.value.some((p) => p.network_rx > 0 || p.network_tx > 0),
)
const hasPing = computed(() =>
  samples.value.some(
    (p) =>
      (p.ping_ct ?? -1) >= 0 ||
      (p.ping_cu ?? -1) >= 0 ||
      (p.ping_cm ?? -1) >= 0 ||
      (p.ping_bd ?? -1) >= 0,
  ),
)

// ---- Live-panel derived values -------------------------------------------
const memPct = computed(() => pctValue(metrics.memory_used, metrics.memory_total))
const swapPct = computed(() => pctValue(metrics.swap_used, metrics.swap_total))
const diskPct = computed(() => pctValue(metrics.hdd_used, metrics.hdd_total))
const cpuPct = computed(() =>
  metrics.cpu === undefined || metrics.cpu === null || metrics.cpu < 0 ? null : metrics.cpu,
)

/** Meter fill class from a 0..100 value: alert >=90, warn >=75. */
function meterClass(v: number | null): string {
  if (v === null) return ''
  if (v >= 90) return 'is-alert'
  if (v >= 75) return 'is-warn'
  return ''
}

const pings = computed(() => [
  { key: 'CT', ping: metrics.ping_ct ?? null, loss: metrics.loss_ct ?? null },
  { key: 'CU', ping: metrics.ping_cu ?? null, loss: metrics.loss_cu ?? null },
  { key: 'CM', ping: metrics.ping_cm ?? null, loss: metrics.loss_cm ?? null },
  { key: 'BD', ping: metrics.ping_bd ?? null, loss: metrics.loss_bd ?? null },
])

const headerLocation = computed(() => {
  const parts = [ip.value?.city, ip.value?.country || server.value?.location].filter(Boolean)
  return parts.length ? parts.join(', ') : dash(server.value?.location)
})

// ---- Lifecycle -----------------------------------------------------------
watch(
  () => props.id,
  (id) => {
    ws.disconnect()
    range.value = '1h'
    samples.value = []
    void loadServer(id)
    void loadHistory(id, range.value)
  },
  { immediate: true },
)

onBeforeUnmount(() => ws.disconnect())
</script>

<template>
  <div class="container detail">
    <!-- Not found -->
    <div v-if="notFound" class="card notfound">
      <h1 class="page-title">{{ t('detail.notFound') }}</h1>
      <p class="dim">{{ props.id }}</p>
      <RouterLink class="btn btn-primary mt-4" :to="{ name: 'dashboard' }">
        {{ t('detail.back') }}
      </RouterLink>
    </div>

    <!-- Error -->
    <div v-else-if="errored" class="card notfound">
      <h1 class="page-title">{{ t('common.error') }}</h1>
      <button class="btn mt-4" @click="loadServer(props.id)">{{ t('common.retry') }}</button>
    </div>

    <!-- Loading -->
    <div v-else-if="loading && !server" class="row" style="justify-content: center; padding: var(--sp-8)">
      <span class="spinner"></span>
    </div>

    <template v-else-if="server">
      <!-- Header -->
      <header class="detail-header">
        <RouterLink class="back-link" :to="{ name: 'dashboard' }">← {{ t('detail.back') }}</RouterLink>
        <div class="row between wrap" style="align-items: flex-start">
          <div class="stack gap-2">
            <div class="row gap-3" style="align-items: center">
              <span class="status-dot" :class="online ? 'is-online' : 'is-offline'"></span>
              <h1 class="page-title">{{ server.name }}</h1>
              <span class="badge" :class="online ? 'is-online' : 'is-offline'">
                {{ online ? t('common.online') : t('common.offline') }}
              </span>
            </div>
            <div class="row wrap gap-3 meta">
              <span v-if="server.type"><span class="eyebrow">{{ t('detail.system') }}</span> {{ server.type }}</span>
              <span><span class="eyebrow">{{ t('detail.location') }}</span> {{ headerLocation }}</span>
              <span v-if="ip?.query"><span class="eyebrow">{{ t('detail.ip') }}</span> <span class="mono">{{ ip.query }}</span></span>
            </div>
          </div>
          <div class="stack gap-1" style="text-align: right">
            <span class="eyebrow">{{ t('detail.lastUpdate') }}</span>
            <span class="mono dim">{{ lastUpdated ? sinceSeconds(lastUpdated) : dash(null) }}</span>
          </div>
        </div>
      </header>

      <!-- Realtime metrics -->
      <section class="stack gap-3 mb-4">
        <div class="row between">
          <h2 class="section-title">{{ t('detail.realtime') }}</h2>
          <span class="badge" :class="online ? 'is-online' : ''">{{ t('detail.live') }}</span>
        </div>
        <p v-if="!online" class="faint" style="font-size: var(--fs-sm)">{{ t('detail.offlineNote') }}</p>

        <div class="grid metric-grid">
          <!-- CPU -->
          <div class="card metric">
            <div class="row between">
              <span class="card-title">{{ t('detail.cpu') }}</span>
              <span class="metric-value">{{ cpuPct === null ? dash(null) : `${Math.round(cpuPct)}%` }}</span>
            </div>
            <div class="meter mt-4"><div class="meter-fill" :class="meterClass(cpuPct)" :style="{ width: `${cpuPct ?? 0}%` }"></div></div>
            <div class="sub mono dim">{{ loadAvg(metrics.load_avg) }}</div>
          </div>

          <!-- Memory -->
          <div class="card metric">
            <div class="row between">
              <span class="card-title">{{ t('detail.memory') }}</span>
              <span class="metric-value">{{ pct(metrics.memory_used, metrics.memory_total) }}</span>
            </div>
            <div class="meter mt-4"><div class="meter-fill" :class="meterClass(memPct)" :style="{ width: `${memPct ?? 0}%` }"></div></div>
            <div class="sub mono dim">{{ mib(metrics.memory_used) }} / {{ mib(metrics.memory_total) }}</div>
          </div>

          <!-- Swap -->
          <div class="card metric">
            <div class="row between">
              <span class="card-title">{{ t('detail.swap') }}</span>
              <span class="metric-value">{{ pct(metrics.swap_used, metrics.swap_total) }}</span>
            </div>
            <div class="meter mt-4"><div class="meter-fill" :class="meterClass(swapPct)" :style="{ width: `${swapPct ?? 0}%` }"></div></div>
            <div class="sub mono dim">{{ mib(metrics.swap_used) }} / {{ mib(metrics.swap_total) }}</div>
          </div>

          <!-- Disk -->
          <div class="card metric">
            <div class="row between">
              <span class="card-title">{{ t('detail.disk') }}</span>
              <span class="metric-value">{{ pct(metrics.hdd_used, metrics.hdd_total) }}</span>
            </div>
            <div class="meter mt-4"><div class="meter-fill" :class="meterClass(diskPct)" :style="{ width: `${diskPct ?? 0}%` }"></div></div>
            <div class="sub mono dim">{{ mib(metrics.hdd_used) }} / {{ mib(metrics.hdd_total) }}</div>
          </div>

          <!-- Network -->
          <div class="card metric">
            <span class="card-title">{{ t('detail.network') }}</span>
            <div class="kv mt-4">
              <span class="dim">↓ {{ t('detail.download') }}</span><span class="metric-value">{{ bytesRate(metrics.network_rx) }}</span>
              <span class="dim">↑ {{ t('detail.upload') }}</span><span class="metric-value">{{ bytesRate(metrics.network_tx) }}</span>
              <span class="dim">{{ t('detail.trafficIn') }}</span><span class="metric-value">{{ bytes(metrics.network_in) }}</span>
              <span class="dim">{{ t('detail.trafficOut') }}</span><span class="metric-value">{{ bytes(metrics.network_out) }}</span>
            </div>
          </div>

          <!-- Connections / processes -->
          <div class="card metric">
            <span class="card-title">{{ t('detail.connections') }}</span>
            <div class="kv mt-4">
              <span class="dim">{{ t('detail.tcp') }}</span><span class="metric-value">{{ dash(metrics.tcp_conn) }}</span>
              <span class="dim">{{ t('detail.udp') }}</span><span class="metric-value">{{ dash(metrics.udp_conn) }}</span>
              <span class="dim">{{ t('detail.processes') }}</span><span class="metric-value">{{ dash(metrics.processes) }}</span>
              <span class="dim">{{ t('detail.uptime') }}</span><span class="metric-value">{{ uptime(metrics.uptime) }}</span>
            </div>
          </div>

          <!-- Ping / loss (three networks + BD) -->
          <div class="card metric ping-card">
            <span class="card-title">{{ t('detail.pingLoss') }}</span>
            <div class="ping-grid mt-4">
              <template v-for="p in pings" :key="p.key">
                <span class="dim mono">{{ p.key }}</span>
                <span class="metric-value">{{ ms(p.ping) }}</span>
                <span class="metric-value loss" :class="{ 'text-warn': (p.loss ?? 0) > 0 && (p.loss ?? 0) < 100, 'text-offline': (p.loss ?? -1) >= 100 }">{{ loss(p.loss) }}</span>
              </template>
            </div>
          </div>
        </div>
      </section>

      <!-- System + location info -->
      <section class="grid info-grid mb-4">
        <div class="card">
          <span class="card-title">{{ t('detail.system') }}</span>
          <dl class="kv mt-4">
            <dt class="dim">{{ t('detail.os') }}</dt><dd class="mono">{{ dash(sys?.os_name || metrics.os) }}</dd>
            <dt class="dim">{{ t('detail.kernel') }}</dt><dd class="mono">{{ dash(sys?.kernel_version || metrics.kernel_version) }}</dd>
            <dt class="dim">{{ t('detail.arch') }}</dt><dd class="mono">{{ dash(sys?.os_arch || metrics.arch) }}</dd>
            <dt class="dim">{{ t('detail.cpuModel') }}</dt><dd class="mono">{{ dash(sys?.cpu_brand || metrics.cpu_model) }}</dd>
            <dt class="dim">{{ t('detail.cores') }}</dt><dd class="mono">{{ dash(sys?.cpu_num || metrics.cpu_cores) }}</dd>
            <dt class="dim">{{ t('detail.host') }}</dt><dd class="mono">{{ dash(sys?.host_name || metrics.host_name) }}</dd>
          </dl>
        </div>
        <div class="card">
          <span class="card-title">{{ t('detail.location') }}</span>
          <dl class="kv mt-4">
            <dt class="dim">{{ t('detail.ip') }}</dt><dd class="mono">{{ dash(ip?.query) }}</dd>
            <dt class="dim">{{ t('detail.isp') }}</dt><dd class="mono">{{ dash(ip?.isp) }}</dd>
            <dt class="dim">{{ t('detail.org') }}</dt><dd class="mono">{{ dash(ip?.org) }}</dd>
            <dt class="dim">{{ t('detail.asn') }}</dt><dd class="mono">{{ dash(ip?.asname || ip?.as) }}</dd>
            <dt class="dim">{{ t('detail.region') }}</dt><dd class="mono">{{ dash(ip?.region_name) }}</dd>
            <dt class="dim">{{ t('detail.country') }}</dt><dd class="mono">{{ dash(ip?.country) }}</dd>
          </dl>
        </div>
      </section>

      <!-- History -->
      <section class="stack gap-4">
        <div class="row between wrap gap-3">
          <h2 class="section-title">{{ t('detail.history') }}</h2>
          <div class="segmented" role="tablist">
            <button
              v-for="r in RANGES"
              :key="r"
              :class="{ 'is-active': r === range }"
              @click="setRange(r)"
            >
              {{ t(`range.${r}`) }}
            </button>
          </div>
        </div>

        <div v-if="historyLoading" class="row" style="justify-content: center; padding: var(--sp-6)">
          <span class="spinner"></span>
        </div>
        <div v-else-if="!samples.length" class="card faint" style="text-align: center">
          {{ t('dash.empty') }}
        </div>

        <template v-else>
          <div class="card chart-card">
            <span class="card-title">{{ t('detail.usage') }}</span>
            <MetricsChart :samples="samples" :series="usageSeries" :range="range" :y-percent="true" :y-label="t('detail.axisPercent')" />
          </div>

          <div v-if="hasNet" class="card chart-card">
            <span class="card-title">{{ t('detail.network') }}</span>
            <MetricsChart :samples="samples" :series="netSeries" :range="range" :y-label="t('detail.axisRate')" />
          </div>

          <div v-if="hasPing" class="card chart-card">
            <span class="card-title">{{ t('detail.pingLoss') }}</span>
            <MetricsChart :samples="samples" :series="pingSeries" :range="range" :y-label="t('detail.axisMs')" />
          </div>
        </template>
      </section>
    </template>
  </div>
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
}

.notfound {
  text-align: center;
  padding: var(--sp-8);
}

.detail-header {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}
.back-link {
  font-size: var(--fs-sm);
  color: var(--text-dim);
  align-self: flex-start;
  transition: color var(--dur) var(--ease);
}
.back-link:hover {
  color: var(--accent);
}
.detail-header .meta {
  font-size: var(--fs-sm);
  color: var(--text);
}
.detail-header .meta .eyebrow {
  margin-right: var(--sp-1);
}

.section-title {
  font-size: var(--fs-lg);
  letter-spacing: -0.01em;
}

.metric-grid {
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
}
.info-grid {
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
}

.metric .metric-value {
  font-size: var(--fs-lg);
}
.metric .sub {
  margin-top: var(--sp-2);
  font-size: var(--fs-xs);
}

/* Two-column key/value list used across metric + info cards. */
.kv {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--sp-1) var(--sp-4);
  align-items: baseline;
  font-size: var(--fs-sm);
}
.kv dt,
.kv > span:nth-child(odd) {
  font-size: var(--fs-xs);
}
.kv dd,
.kv .metric-value {
  text-align: right;
  overflow-wrap: anywhere;
}

/* Ping card: network | ping | loss triple columns. */
.ping-grid {
  display: grid;
  grid-template-columns: auto 1fr 1fr;
  gap: var(--sp-2) var(--sp-3);
  align-items: baseline;
  font-size: var(--fs-sm);
}
.ping-grid .metric-value {
  text-align: right;
  font-size: var(--fs-sm);
}
.ping-grid .loss {
  color: var(--text-dim);
}

.chart-card {
  padding-top: var(--sp-3);
}
.chart-card .card-title {
  display: block;
  margin-bottom: var(--sp-2);
}
</style>
