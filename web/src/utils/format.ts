// Pure display formatters. Every function treats null / undefined / -1 as
// "unmeasured" and renders the em-dash placeholder so views never leak raw
// sentinel values.

export const DASH = '—'

/** True when a value is an unmeasured sentinel (null, undefined, or -1). */
function isEmpty(v: number | null | undefined): boolean {
  return v === null || v === undefined || v === -1 || Number.isNaN(v as number)
}

/** Render a value or the em-dash when null/undefined/-1. */
export function dash(v: unknown): string {
  if (v === null || v === undefined || v === -1 || v === '') return DASH
  return String(v)
}

/**
 * Format a MiB count as GiB (>= 1024 MiB) or MiB.
 * mib(2048) -> "2.0 GiB", mib(512) -> "512 MiB".
 */
export function mib(n: number | null | undefined): string {
  if (isEmpty(n)) return DASH
  const v = n as number
  if (v >= 1024) return `${(v / 1024).toFixed(1)} GiB`
  return `${Math.round(v)} MiB`
}

/** Percentage of used over total. pct(45, 100) -> "45%". */
export function pct(used: number | null | undefined, total: number | null | undefined): string {
  if (isEmpty(used) || isEmpty(total) || (total as number) <= 0) return DASH
  return `${Math.round(((used as number) / (total as number)) * 100)}%`
}

/** Numeric percentage 0..100 (or null) for progress bars. */
export function pctValue(
  used: number | null | undefined,
  total: number | null | undefined,
): number | null {
  if (isEmpty(used) || isEmpty(total) || (total as number) <= 0) return null
  return Math.min(100, Math.max(0, ((used as number) / (total as number)) * 100))
}

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

/** Human-readable byte size. bytes(1536) -> "1.5 KB". */
export function bytes(n: number | null | undefined): string {
  if (isEmpty(n)) return DASH
  let v = n as number
  if (v < 1) return '0 B'
  let i = 0
  while (v >= 1024 && i < BYTE_UNITS.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? Math.round(v) : v.toFixed(1)} ${BYTE_UNITS[i]}`
}

/** Byte-per-second rate. bytesRate(1258291) -> "1.2 MB/s". */
export function bytesRate(bps: number | null | undefined): string {
  if (isEmpty(bps)) return DASH
  return `${bytes(bps)}/s`
}

/** Milliseconds. ms(32) -> "32 ms", ms(null) -> "—". */
export function ms(n: number | null | undefined): string {
  if (isEmpty(n)) return DASH
  const v = n as number
  return `${v < 10 ? v.toFixed(1) : Math.round(v)} ms`
}

/** Loss percentage. loss(0) -> "0%", loss(null) -> "—". */
export function loss(n: number | null | undefined): string {
  if (isEmpty(n)) return DASH
  return `${Math.round(n as number)}%`
}

/** Relative time from a Unix timestamp in seconds. sinceSeconds -> "3s ago". */
export function sinceSeconds(unixSec: number | null | undefined): string {
  if (isEmpty(unixSec)) return DASH
  const diff = Math.max(0, Math.floor(Date.now() / 1000 - (unixSec as number)))
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

/** Uptime in seconds -> "3d 4h" / "5h 12m" / "42m". */
export function uptime(sec: number | null | undefined): string {
  if (isEmpty(sec)) return DASH
  const s = sec as number
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

/** Parse a "l1 l5 l15" load-average string into a normalized triple string. */
export function loadAvg(str: string | null | undefined): string {
  if (!str) return DASH
  const parts = str.trim().split(/\s+/).slice(0, 3)
  if (parts.length === 0) return DASH
  const nums = parts.map((p) => {
    const f = parseFloat(p)
    return Number.isFinite(f) ? f.toFixed(2) : DASH
  })
  return nums.join(' ')
}

/** Absolute local date-time from Unix seconds. */
export function dateTime(unixSec: number | null | undefined): string {
  if (isEmpty(unixSec)) return DASH
  return new Date((unixSec as number) * 1000).toLocaleString()
}

/** HH:MM:SS clock label from Unix seconds — for category-axis chart labels. */
export function clockLabel(unixSec: number | null | undefined): string {
  if (isEmpty(unixSec)) return DASH
  const d = new Date((unixSec as number) * 1000)
  const p = (x: number) => String(x).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}`
}
