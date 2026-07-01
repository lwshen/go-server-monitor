<script setup lang="ts">
// Compact, client-sortable table view of the dashboard server list.
// Row click -> detail page. Metrics render live from the reactive store rows.
// The edit/delete emits are kept so the admin view (P6) can reuse this shell.

import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type { Server } from '@/types'
import { bytesRate, ms, pct, uptime } from '@/utils/format'

const props = defineProps<{ servers: Server[] }>()

defineEmits<{
  edit: [server: Server]
  delete: [id: string]
}>()

const router = useRouter()
const { t } = useI18n()

type SortKey = 'name' | 'group' | 'cpu' | 'mem' | 'net' | 'ping' | 'uptime' | 'status'
const sortKey = ref<SortKey>('name')
const sortDir = ref<'asc' | 'desc'>('asc')

function setSort(key: SortKey): void {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = 'asc'
  }
}

function ariaSort(key: SortKey): 'ascending' | 'descending' | 'none' {
  if (sortKey.value !== key) return 'none'
  return sortDir.value === 'asc' ? 'ascending' : 'descending'
}

/** Extract a comparable value for a server under the active sort key. */
function sortValue(s: Server, key: SortKey): number | string {
  const m = s.latest_metrics
  switch (key) {
    case 'name':
      return (s.alias?.trim() || s.name).toLowerCase()
    case 'group':
      return (s.server_group || '').toLowerCase()
    case 'cpu':
      return m?.cpu ?? -1
    case 'mem': {
      const used = m?.memory_used
      const total = m?.memory_total
      if (typeof used !== 'number' || typeof total !== 'number' || total <= 0) return -1
      return used / total
    }
    case 'net':
      return (m?.network_rx ?? 0) + (m?.network_tx ?? 0)
    case 'ping':
      // Sort unmeasured (null) to the bottom.
      return m?.ping_ct ?? Number.POSITIVE_INFINITY
    case 'uptime':
      return m?.uptime ?? -1
    case 'status':
      return s.online ? 1 : 0
  }
}

const sorted = computed(() => {
  const dir = sortDir.value === 'asc' ? 1 : -1
  const key = sortKey.value
  return [...props.servers].sort((a, b) => {
    const va = sortValue(a, key)
    const vb = sortValue(b, key)
    if (typeof va === 'string' && typeof vb === 'string') {
      return va.localeCompare(vb) * dir
    }
    return ((va as number) - (vb as number)) * dir
  })
})

const columns: { key: SortKey; label: string; num?: boolean }[] = [
  { key: 'name', label: 'th.name' },
  { key: 'group', label: 'th.group' },
  { key: 'cpu', label: 'th.cpu', num: true },
  { key: 'mem', label: 'th.mem', num: true },
  { key: 'net', label: 'th.net', num: true },
  { key: 'ping', label: 'th.ping', num: true },
  { key: 'uptime', label: 'th.uptime', num: true },
  { key: 'status', label: 'th.status' },
]

function cpuLabel(s: Server): string {
  const c = s.latest_metrics?.cpu
  return typeof c === 'number' && c >= 0 ? `${Math.round(c)}%` : '—'
}

function open(id: string): void {
  void router.push({ name: 'server-detail', params: { id } })
}
</script>

<template>
  <div class="table-scroll">
    <table class="data">
      <thead>
        <tr>
          <th
            v-for="col in columns"
            :key="col.key"
            :class="{ num: col.num, sortable: true, active: sortKey === col.key }"
            :aria-sort="ariaSort(col.key)"
            @click="setSort(col.key)"
          >
            <span class="th-inner">
              {{ t(col.label) }}
              <span v-if="sortKey === col.key" class="sort-caret">{{
                sortDir === 'asc' ? '▲' : '▼'
              }}</span>
            </span>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in sorted" :key="s.id" class="clickable" @click="open(s.id)">
          <td>
            <span class="row gap-2">
              <span
                class="status-dot"
                :class="s.online ? 'is-online' : 'is-offline'"
              />
              <span class="td-name">{{ s.alias?.trim() || s.name }}</span>
            </span>
          </td>
          <td class="dim">{{ s.server_group || '—' }}</td>
          <td class="num">{{ cpuLabel(s) }}</td>
          <td class="num">{{ pct(s.latest_metrics?.memory_used, s.latest_metrics?.memory_total) }}</td>
          <td class="num net-cell">
            <span class="net-up">↑ {{ bytesRate(s.latest_metrics?.network_tx) }}</span>
            <span class="net-down">↓ {{ bytesRate(s.latest_metrics?.network_rx) }}</span>
          </td>
          <td class="num">{{ ms(s.latest_metrics?.ping_ct ?? null) }}</td>
          <td class="num">{{ uptime(s.latest_metrics?.uptime) }}</td>
          <td>
            <span class="badge" :class="s.online ? 'is-online' : 'is-offline'">
              {{ s.online ? t('common.online') : t('common.offline') }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
th.sortable {
  cursor: pointer;
  user-select: none;
}
th.sortable:hover {
  color: var(--text);
}
th.active {
  color: var(--accent);
}
.th-inner {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-1);
}
.sort-caret {
  font-size: 0.5rem;
}
tr.clickable {
  cursor: pointer;
}
.td-name {
  font-weight: 550;
}
.net-cell {
  text-align: right;
}
.net-cell .net-up,
.net-cell .net-down {
  display: block;
  white-space: nowrap;
}
.net-up {
  color: var(--text-dim);
}
</style>
