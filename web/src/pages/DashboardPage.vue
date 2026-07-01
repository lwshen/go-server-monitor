<script setup lang="ts">
// Dashboard home. Fetches the server list once, opens the realtime feed, and
// renders global stat cards + a cards/table view (toggle persisted) + the world
// map. All per-server metrics update reactively as the store merges WS frames.

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useServersStore } from '@/stores/servers'
import type { Server } from '@/types'
import ServerCard from '@/components/ServerCard.vue'
import ServerTable from '@/components/ServerTable.vue'
import WorldMap from '@/components/WorldMap.vue'

const { t } = useI18n()
const store = useServersStore()
const { servers, stats, loading } = storeToRefs(store)

// ---- View toggle (persisted) ----------------------------------------------
type ViewMode = 'cards' | 'table'
const VIEW_KEY = 'dash_view'
function initialView(): ViewMode {
  const v = localStorage.getItem(VIEW_KEY)
  return v === 'table' ? 'table' : 'cards'
}
const view = ref<ViewMode>(initialView())
function setView(v: ViewMode): void {
  view.value = v
  localStorage.setItem(VIEW_KEY, v)
}

// ---- Group-by toggle (persisted) -------------------------------------------
const GROUP_KEY = 'dash_grouped'
const grouped = ref<boolean>(localStorage.getItem(GROUP_KEY) === '1')
function toggleGrouped(): void {
  grouped.value = !grouped.value
  localStorage.setItem(GROUP_KEY, grouped.value ? '1' : '0')
}

// ---- Aggregate metrics across online servers -------------------------------
const onlineServers = computed(() => servers.value.filter((s) => s.online))

const avgCpu = computed<number | null>(() => {
  const vals = onlineServers.value
    .map((s) => s.latest_metrics?.cpu)
    .filter((c): c is number => typeof c === 'number' && c >= 0)
  if (vals.length === 0) return null
  return vals.reduce((a, b) => a + b, 0) / vals.length
})

const avgMem = computed<number | null>(() => {
  const ratios = onlineServers.value
    .map((s) => {
      const used = s.latest_metrics?.memory_used
      const total = s.latest_metrics?.memory_total
      if (typeof used !== 'number' || typeof total !== 'number' || total <= 0) return null
      return used / total
    })
    .filter((r): r is number => r !== null)
  if (ratios.length === 0) return null
  return (ratios.reduce((a, b) => a + b, 0) / ratios.length) * 100
})

function pctLabel(v: number | null): string {
  return v === null ? '—' : `${Math.round(v)}%`
}

const statCards = computed(() => [
  { key: 'total', label: t('dash.total'), value: String(stats.value.total), tone: '' },
  { key: 'online', label: t('dash.online'), value: String(stats.value.online), tone: 'online' },
  { key: 'offline', label: t('dash.offline'), value: String(stats.value.offline), tone: 'offline' },
  { key: 'cpu', label: t('dash.cpuAvg'), value: pctLabel(avgCpu.value), tone: '' },
  { key: 'mem', label: t('dash.memAvg'), value: pctLabel(avgMem.value), tone: '' },
])

// ---- Grouping --------------------------------------------------------------
const groups = computed<{ name: string; servers: Server[] }[]>(() => {
  const map = new Map<string, Server[]>()
  for (const s of servers.value) {
    const g = s.server_group?.trim() || '—'
    const arr = map.get(g)
    if (arr) arr.push(s)
    else map.set(g, [s])
  }
  return [...map.entries()].map(([name, list]) => ({ name, servers: list }))
})

// ---- Lifecycle -------------------------------------------------------------
onMounted(() => {
  void store.fetchServers()
  store.startLive()
})
onUnmounted(() => {
  store.stopLive()
})

const showEmpty = computed(() => !loading.value && servers.value.length === 0)
const showLoading = computed(() => loading.value && servers.value.length === 0)
</script>

<template>
  <div class="container dashboard-page">
    <!-- Global stats -->
    <section class="grid grid-stats stats-row" aria-label="overview">
      <div v-for="c in statCards" :key="c.key" class="card stat-card">
        <span class="eyebrow">{{ c.label }}</span>
        <span
          class="stat-value"
          :class="{ 'text-online': c.tone === 'online', 'text-offline': c.tone === 'offline' }"
          >{{ c.value }}</span
        >
      </div>
    </section>

    <!-- Toolbar: title + group toggle + view toggle -->
    <div class="row between wrap toolbar">
      <h1 class="page-title">{{ t('nav.dashboard') }}</h1>
      <div class="row gap-2 wrap">
        <button
          type="button"
          class="btn btn-sm"
          :class="{ 'is-on': grouped }"
          :aria-pressed="grouped"
          @click="toggleGrouped"
        >
          {{ t('th.group') }}
        </button>
        <div class="segmented" role="tablist" :aria-label="t('common.theme')">
          <button
            type="button"
            role="tab"
            :aria-selected="view === 'cards'"
            :class="{ 'is-active': view === 'cards' }"
            @click="setView('cards')"
          >
            {{ t('dash.cards') }}
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="view === 'table'"
            :class="{ 'is-active': view === 'table' }"
            @click="setView('table')"
          >
            {{ t('dash.table') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="showLoading" class="state-block">
      <span class="spinner" aria-hidden="true" />
      <span class="dim">{{ t('common.loading') }}</span>
    </div>

    <!-- Empty -->
    <div v-else-if="showEmpty" class="state-block">
      <span class="dim">{{ t('dash.empty') }}</span>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Table view -->
      <ServerTable v-if="view === 'table'" :servers="servers" />

      <!-- Cards view -->
      <template v-else>
        <!-- Grouped -->
        <template v-if="grouped">
          <section v-for="g in groups" :key="g.name" class="group-section">
            <div class="row gap-2 group-head">
              <span class="eyebrow">{{ g.name }}</span>
              <span class="faint group-count">{{ g.servers.length }}</span>
            </div>
            <div class="grid grid-cards">
              <ServerCard v-for="s in g.servers" :key="s.id" :server="s" />
            </div>
          </section>
        </template>
        <!-- Flat -->
        <div v-else class="grid grid-cards">
          <ServerCard v-for="s in servers" :key="s.id" :server="s" />
        </div>
      </template>

      <!-- World map (owned by the map agent) -->
      <section class="map-section">
        <WorldMap :servers="servers" />
      </section>
    </template>
  </div>
</template>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
}

.stats-row {
  margin-top: var(--sp-1);
}
.stat-card {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.toolbar {
  gap: var(--sp-3);
}

.btn.is-on {
  border-color: var(--accent);
  color: var(--accent);
}

.state-block {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-3);
  min-height: 160px;
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  background: var(--surface);
}

.group-section {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}
.group-section + .group-section {
  margin-top: var(--sp-4);
}
.group-head {
  align-items: baseline;
}
.group-count {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
}

.map-section {
  margin-top: var(--sp-2);
}
</style>
