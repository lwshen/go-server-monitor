// Pinia store for the server list, aggregate stats, runtime config, and the
// realtime feed. Live updates arrive over WebSocket; if the socket drops the
// store falls back to polling and keeps retrying the socket.

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { AppConfig, MetricsRow, Server, Stats, WsFrame } from '@/types'
import { getConfig, getServers } from '@/services/api'
import { WSManager } from '@/services/ws'

const POLL_INTERVAL = 10000

export const useServersStore = defineStore('servers', () => {
  const servers = ref<Server[]>([])
  const stats = ref<Stats>({ total: 0, online: 0, offline: 0 })
  const config = ref<AppConfig | null>(null)
  const loading = ref(false)
  const live = ref(false)

  const ws = new WSManager()
  let pollTimer: ReturnType<typeof setInterval> | null = null

  /** id -> index into servers.value, rebuilt whenever the list changes. */
  const indexById = new Map<string, number>()
  function reindex(): void {
    indexById.clear()
    servers.value.forEach((s, i) => indexById.set(s.id, i))
  }

  function findIndex(id: string): number {
    const i = indexById.get(id)
    if (i !== undefined && servers.value[i]?.id === id) return i
    return servers.value.findIndex((s) => s.id === id)
  }

  /** Fetch public runtime config (site title, defaults). */
  async function loadConfig(): Promise<AppConfig | null> {
    try {
      config.value = await getConfig()
    } catch {
      config.value = null
    }
    return config.value
  }

  /** Fetch the server list + aggregate stats. */
  async function fetchServers(): Promise<void> {
    loading.value = true
    try {
      const res = await getServers()
      servers.value = res.servers
      stats.value = res.stats
      reindex()
    } finally {
      loading.value = false
    }
  }

  function getById(id: string): Server | undefined {
    const i = findIndex(id)
    return i >= 0 ? servers.value[i] : undefined
  }

  /** Merge a dynamic-metric map into a server's latest snapshot. */
  function mergeMetrics(serverId: string, ts: number, data: Partial<MetricsRow>): void {
    const i = findIndex(serverId)
    if (i < 0) return
    const server = servers.value[i]
    const base: MetricsRow = (server.latest_metrics ?? { server_id: serverId }) as MetricsRow
    const merged: MetricsRow = { ...base, ...data, timestamp: ts || base.timestamp }
    servers.value[i] = { ...server, latest_metrics: merged, last_updated: ts, online: true }
  }

  function applyFrame(frame: WsFrame): void {
    if (frame.type === 'update' && frame.serverId && frame.data) {
      mergeMetrics(frame.serverId, frame.ts ?? Math.floor(Date.now() / 1000), frame.data)
    } else if (frame.type === 'batchUpdate' && frame.updates) {
      for (const u of frame.updates) {
        if (!u.samples?.length) continue
        // Apply the newest sample per server.
        const newest = u.samples.reduce((a, b) => (b.ts > a.ts ? b : a))
        mergeMetrics(u.serverId, newest.ts, newest.data)
      }
    }
  }

  function startPolling(): void {
    if (pollTimer) return
    pollTimer = setInterval(() => {
      void fetchServers()
      // Keep trying to restore the realtime socket.
      if (!ws.isOpen) ws.connect('all', applyFrame)
    }, POLL_INTERVAL)
  }

  function stopPolling(): void {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  /** Open the realtime feed (subscribe=all) with polling as a safety net. */
  function startLive(): void {
    if (live.value) return
    live.value = true
    ws.connect('all', applyFrame)
    startPolling()
  }

  /** Tear down the realtime feed and polling. */
  function stopLive(): void {
    live.value = false
    ws.disconnect()
    stopPolling()
  }

  return {
    servers,
    stats,
    config,
    loading,
    live,
    loadConfig,
    fetchServers,
    getById,
    startLive,
    stopLive,
  }
})
