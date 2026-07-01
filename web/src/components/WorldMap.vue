<script setup lang="ts">
// Leaflet world map. One circleMarker per server that reports ip_info lat/lon.
// Green = online, red = offline. Popup shows name, location, CPU/mem, status.
// Clicking a marker routes to the server detail page. We use circleMarker (SVG,
// no image assets) to sidestep Leaflet's default-icon path problem, so no
// marker-icon PNGs are ever requested.

import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import type { Server } from '@/types'
import { pct } from '@/utils/format'

const props = defineProps<{
  servers: Server[]
}>()

const router = useRouter()
const { t } = useI18n()

const mapElement = ref<HTMLDivElement | null>(null)

let map: L.Map | null = null
let markerLayer: L.LayerGroup | null = null

// Resolve semantic status colors from the CSS custom properties so the map
// tracks the active theme instead of hard-coding literals.
function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

/** A server is mappable only when it carries finite geo coordinates. */
function hasCoords(s: Server): boolean {
  const info = s.ip_info
  return (
    !!info &&
    Number.isFinite(info.lat) &&
    Number.isFinite(info.lon) &&
    !(info.lat === 0 && info.lon === 0)
  )
}

/** Human-readable location line assembled from ip_info / server fields. */
function locationLabel(s: Server): string {
  const info = s.ip_info
  const parts = [info?.city, info?.region_name, info?.country].filter(
    (p): p is string => !!p && p.trim().length > 0,
  )
  if (parts.length > 0) return parts.join(', ')
  return s.location || t('common.dash')
}

/** Escape user-controlled strings before injecting into popup HTML. */
function esc(v: string): string {
  return v
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function popupHtml(s: Server): string {
  const m = s.latest_metrics
  const online = s.online
  const statusColor = online ? cssVar('--online', '#34d399') : cssVar('--offline', '#f87171')
  const statusText = online ? t('common.online') : t('common.offline')
  const cpuText = m && Number.isFinite(m.cpu) && m.cpu >= 0 ? `${Math.round(m.cpu)}%` : t('common.dash')
  const memText = m ? pct(m.memory_used, m.memory_total) : t('common.dash')

  return `
    <div class="wm-popup">
      <div class="wm-popup-head">
        <span class="wm-dot" style="background:${statusColor}"></span>
        <span class="wm-name">${esc(s.name)}</span>
      </div>
      <div class="wm-loc">${esc(locationLabel(s))}</div>
      <div class="wm-metrics">
        <span class="wm-metric"><span class="wm-k">${esc(t('detail.cpu'))}</span><span class="wm-v">${cpuText}</span></span>
        <span class="wm-metric"><span class="wm-k">${esc(t('detail.memory'))}</span><span class="wm-v">${memText}</span></span>
      </div>
      <div class="wm-status" style="color:${statusColor}">${esc(statusText)}</div>
    </div>
  `
}

function renderMarkers(): void {
  if (!map || !markerLayer) return
  markerLayer.clearLayers()

  const online = cssVar('--online', '#34d399')
  const offline = cssVar('--offline', '#f87171')
  const points: L.LatLngExpression[] = []

  for (const s of props.servers) {
    if (!hasCoords(s)) continue
    const info = s.ip_info!
    const color = s.online ? online : offline
    const latlng: L.LatLngExpression = [info.lat, info.lon]
    points.push(latlng)

    const marker = L.circleMarker(latlng, {
      radius: 7,
      color,
      weight: 2,
      fillColor: color,
      fillOpacity: 0.7,
    })

    marker.bindPopup(popupHtml(s), { closeButton: true })
    marker.on('click', () => {
      void router.push({ name: 'server-detail', params: { id: s.id } })
    })
    marker.addTo(markerLayer)
  }

  // Fit to markers when we have any; otherwise keep the default world view.
  if (points.length === 1) {
    map.setView(points[0], 4)
  } else if (points.length > 1) {
    map.fitBounds(L.latLngBounds(points), { padding: [40, 40], maxZoom: 6 })
  }
}

onMounted(() => {
  if (!mapElement.value) return

  map = L.map(mapElement.value, {
    center: [20, 0],
    zoom: 2,
    minZoom: 1,
    maxZoom: 12,
    worldCopyJump: true,
    attributionControl: true,
    zoomControl: true,
  })

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; OpenStreetMap contributors',
    maxZoom: 19,
  }).addTo(map)

  markerLayer = L.layerGroup().addTo(map)
  renderMarkers()
})

// Re-plot when the server list (positions, online state, metrics) changes.
watch(
  () => props.servers,
  () => renderMarkers(),
  { deep: true },
)

onBeforeUnmount(() => {
  if (map) {
    map.remove()
    map = null
  }
  markerLayer = null
})
</script>

<template>
  <div ref="mapElement" class="world-map" role="region" :aria-label="t('nav.map')"></div>
</template>

<style scoped>
.world-map {
  height: 360px;
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--surface-2);
}

/* Popup chrome themed to match the dashboard surfaces. Leaflet renders the
   popup outside this component's scope in most builds, so these are :deep. */
.world-map :deep(.leaflet-popup-content-wrapper) {
  background: var(--surface);
  color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}
.world-map :deep(.leaflet-popup-tip) {
  background: var(--surface);
  border: 1px solid var(--border-strong);
}
.world-map :deep(.leaflet-popup-content) {
  margin: var(--sp-3);
}
.world-map :deep(.leaflet-container a.leaflet-popup-close-button) {
  color: var(--text-dim);
}

.world-map :deep(.wm-popup) {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
  min-width: 160px;
  font-family: var(--font-ui);
}
.world-map :deep(.wm-popup-head) {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.world-map :deep(.wm-dot) {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
.world-map :deep(.wm-name) {
  font-weight: 650;
  font-size: var(--fs-base);
  color: var(--text);
}
.world-map :deep(.wm-loc) {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.world-map :deep(.wm-metrics) {
  display: flex;
  gap: var(--sp-4);
}
.world-map :deep(.wm-metric) {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.world-map :deep(.wm-k) {
  font-size: var(--fs-xs);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--text-dim);
  font-weight: 600;
}
.world-map :deep(.wm-v) {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--fs-sm);
  color: var(--text);
}
.world-map :deep(.wm-status) {
  font-size: var(--fs-xs);
  font-weight: 600;
}
</style>
