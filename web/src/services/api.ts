import axios from 'axios'
import type { AxiosInstance } from 'axios'

// Axios instance. baseURL from VITE_API_BASE_URL (default http://localhost:8080).
// TODO(P5): request/response interceptors (attach JWT, handle 401 -> login),
// error normalization to the {error,code} contract.
const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export const apiClient: AxiosInstance = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
})

// ---- Light placeholder types (refined in P5 against report-types.ts) ----

/** Public runtime config from GET /api/config. */
export interface AppConfig {
  [key: string]: unknown
}

/** A server entry with its latest metrics. Fields TBD in P5. */
export interface Server {
  id: string
  name: string
  [key: string]: unknown
}

/** One downsampled history sample. Fields TBD in P5. */
export interface HistoryPoint {
  timestamp: number
  [key: string]: unknown
}

/** History query range buckets (matches backend /api/history range param). */
export type HistoryRange = '1h' | '6h' | '24h' | '7d' | '30d' | '180d'

// ---- Stub API functions (no real logic until P5) ----

/** GET /api/config — public runtime config. */
export function getConfig(): Promise<AppConfig> {
  // TODO(P5): return apiClient.get('/api/config').then(r => r.data)
  return Promise.reject(new Error('getConfig not implemented (P5)'))
}

/** GET /api/servers — list + stats. */
export function getServers(): Promise<Server[]> {
  // TODO(P5): return apiClient.get('/api/servers').then(r => r.data)
  return Promise.reject(new Error('getServers not implemented (P5)'))
}

/** GET /api/server?id=<id> — one server detail. */
export function getServer(id: string): Promise<Server> {
  // TODO(P5): return apiClient.get('/api/server', { params: { id } }).then(r => r.data)
  void id
  return Promise.reject(new Error('getServer not implemented (P5)'))
}

/** GET /api/history?id=<id>&range=<r> — downsampled history. */
export function getServerHistory(
  id: string,
  range: HistoryRange,
): Promise<HistoryPoint[]> {
  // TODO(P5): return apiClient.get('/api/history', { params: { id, range } }).then(r => r.data)
  void id
  void range
  return Promise.reject(new Error('getServerHistory not implemented (P5)'))
}
