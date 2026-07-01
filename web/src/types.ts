// Shared TypeScript interfaces mirroring the Go backend JSON shapes.
// Field names match the API exactly. Unmeasured numeric fields arrive as
// `null` or `-1`; the formatters in utils/format.ts render those as "—".

/** Per-sample metric row. Cumulative network fields are bytes; rx/tx are B/s. */
export interface MetricsRow {
  server_id: string
  timestamp: number
  cpu: number
  /** Space-separated "l1 l5 l15". */
  load_avg: string
  processes: number
  tcp_conn: number
  udp_conn: number
  thread: number
  cpu_cores: number
  cpu_info: string
  cpu_model: string
  memory_total: number
  memory_used: number
  swap_total: number
  swap_used: number
  hdd_total: number
  hdd_used: number
  /** Network speed, B/s. */
  network_rx: number
  network_tx: number
  /** Cumulative counters, bytes. */
  network_in: number
  network_out: number
  last_network_in: number
  last_network_out: number
  ping_ct: number | null
  ping_cu: number | null
  ping_cm: number | null
  ping_bd: number | null
  loss_ct: number | null
  loss_cu: number | null
  loss_cm: number | null
  loss_bd: number | null
  online4: 0 | 1
  online6: 0 | 1
  os: string
  os_release: string
  kernel_version: string
  arch: string
  os_family: string
  uptime: number
  host_name: string
  gpu: number | null
  gpu_info: string
  region: string
  gid: string
  location: string
  vnstat: string
  custom: string
  /** Raw JSON string describing per-disk usage. */
  disks_json: string
}

/** One point in a history series. */
export interface HistoryPoint {
  ts: number
  cpu: number
  memory_used: number
  memory_total: number
  swap_used: number
  swap_total: number
  hdd_used: number
  hdd_total: number
  network_rx: number
  network_tx: number
  tcp_conn: number
  processes: number
  ping_ct: number | null
  ping_cu: number | null
  ping_cm: number | null
  ping_bd: number | null
  loss_ct: number | null
  loss_cu: number | null
  loss_cm: number | null
  loss_bd: number | null
}

/** System info reported by the agent. */
export interface SysInfo {
  name: string
  version: string
  os_name: string
  os_arch: string
  os_family: string
  os_release: string
  kernel_version: string
  cpu_num: number
  cpu_brand: string
  cpu_vender_id: string
  host_name: string
}

/** GeoIP / network location info. */
export interface IpInfo {
  query: string
  source: string
  continent: string
  country: string
  region_name: string
  city: string
  isp: string
  org: string
  as: string
  asname: string
  lat: number
  lon: number
  timezone: string
}

/** A monitored server plus its latest snapshot and metadata. */
export interface Server {
  id: string
  name: string
  server_group: string
  price: number
  expire_date: string
  bandwidth: number
  traffic_limit: number
  traffic_calc_type: string
  reset_day: number
  collect_interval: number
  report_interval: number
  ping_mode: string
  is_hidden: boolean
  sort_order: number
  created_at: number
  updated_at: number
  gid: string
  alias: string
  type: string
  location: string
  notify: boolean
  last_updated: number
  online: boolean
  latest_metrics: MetricsRow | null
  sys_info?: SysInfo
  ip_info?: IpInfo
}

/** Aggregate dashboard counters. */
export interface Stats {
  total: number
  online: number
  offline: number
}

/** Runtime site config (GET /api/config). All values are strings. */
export interface AppConfig {
  site_title: string
  theme_default: string
  lang_default: string
  is_public: string
  [key: string]: string
}

/** GET /api/servers */
export interface ServersResponse {
  servers: Server[]
  stats: Stats
}

/** Supported history ranges. */
export type HistoryRange = '1h' | '6h' | '24h' | '7d' | '30d' | '180d'

/** GET /api/history */
export interface HistoryResponse {
  id: string
  range: HistoryRange
  samples: HistoryPoint[]
}

/** POST /api/admin/login */
export interface LoginResponse {
  token: string
  expires_in: number
}

/** One server's batch of samples inside a batchUpdate frame. */
export interface WsBatchUpdate {
  serverId: string
  samples: Array<{ ts: number; data: Partial<MetricsRow> }>
}

/** A frame received over the /ws socket. */
export interface WsFrame {
  type: 'hello' | 'update' | 'batchUpdate' | 'ping' | 'pong'
  serverId?: string
  ts?: number
  /** Present on "update": dynamic-only metric map. */
  data?: Partial<MetricsRow>
  /** Present on "batchUpdate". */
  updates?: WsBatchUpdate[]
  /** Present on "hello". */
  subscribed?: string
}
