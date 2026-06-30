import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Server } from '@/services/api'

// Pinia store for the server list and realtime state.
// TODO(P5): real fetch (getServers), WebSocket merge of batchUpdate samples,
// per-server lastUpdateTime tracking, online derivation.
export const useServersStore = defineStore('servers', () => {
  const servers = ref<Server[]>([])
  const selectedServer = ref<Server | null>(null)

  /** Initial load of the server list. */
  async function fetchServers(): Promise<void> {
    // TODO(P5): servers.value = await getServers()
    console.log('[serversStore] fetchServers: not implemented (P5)')
  }

  /** Merge a realtime update for one server. */
  function updateServer(data: Partial<Server> & { id: string }): void {
    // TODO(P5): find by id and Object.assign latest sample; track lastUpdateTime.
    void data
    console.log('[serversStore] updateServer: not implemented (P5)')
  }

  return { servers, selectedServer, fetchServers, updateServer }
})
