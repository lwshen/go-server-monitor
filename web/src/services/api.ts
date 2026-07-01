// Typed axios client for the backend. baseURL is relative by default so the
// Vite dev proxy (and same-origin prod) forwards /api to the Go server.

import axios, { type AxiosInstance } from 'axios'
import type {
  AppConfig,
  HistoryRange,
  HistoryResponse,
  LoginResponse,
  Server,
  ServersResponse,
} from '@/types'

// Re-export the core domain types so views can import them from either
// '@/types' (canonical) or '@/services/api' (convenience).
export type {
  AppConfig,
  HistoryPoint,
  HistoryRange,
  HistoryResponse,
  IpInfo,
  LoginResponse,
  MetricsRow,
  Server,
  ServersResponse,
  Stats,
  SysInfo,
  WsFrame,
} from '@/types'

export const TOKEN_KEY = 'token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  headers: { 'Content-Type': 'application/json' },
})

// Attach the bearer token when present.
http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401, drop the token and let stores/guards react (no hard redirect).
http.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error?.response?.status === 401) {
      clearToken()
    }
    return Promise.reject(error)
  },
)

export { http }

// ---- Public endpoints -----------------------------------------------------

export async function getConfig(): Promise<AppConfig> {
  const { data } = await http.get<AppConfig>('/api/config')
  return data
}

export async function getServers(): Promise<ServersResponse> {
  const { data } = await http.get<ServersResponse>('/api/servers')
  return data
}

export async function getServer(id: string): Promise<Server> {
  const { data } = await http.get<Server>('/api/server', { params: { id } })
  return data
}

export async function getHistory(id: string, range: HistoryRange): Promise<HistoryResponse> {
  const { data } = await http.get<HistoryResponse>('/api/history', { params: { id, range } })
  return data
}

// ---- Admin endpoints (JWT) ------------------------------------------------

export async function login(username: string, password: string): Promise<LoginResponse> {
  const { data } = await http.post<LoginResponse>('/api/admin/login', { username, password })
  return data
}

export async function adminListServers(): Promise<Server[]> {
  const { data } = await http.post<{ servers: Server[] }>('/api/admin/servers')
  return data.servers
}

export interface AddServerBody {
  name: string
  server_group?: string
  expire_date?: string
  [key: string]: unknown
}

export async function addServer(body: AddServerBody): Promise<Server> {
  const { data } = await http.post<Server>('/api/admin/servers/add', body)
  return data
}

export interface EditServerBody {
  id: string
  [key: string]: unknown
}

export async function editServer(body: EditServerBody): Promise<void> {
  await http.post('/api/admin/servers/edit', body)
}

export async function deleteServer(id: string): Promise<void> {
  await http.post('/api/admin/servers/delete', { id })
}

export async function reorderServers(ids: string[]): Promise<void> {
  await http.post('/api/admin/servers/reorder', { ids })
}

export async function getSettings(): Promise<Record<string, string | boolean>> {
  const { data } = await http.get<Record<string, string | boolean>>('/api/admin/settings')
  return data
}

export async function saveSettings(obj: Record<string, unknown>): Promise<void> {
  await http.post('/api/admin/settings', obj)
}

export async function dbRebuild(): Promise<void> {
  await http.post('/api/admin/db/rebuild', { confirm: true })
}
