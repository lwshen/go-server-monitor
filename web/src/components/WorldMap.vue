<script setup lang="ts">
import { ref } from 'vue'
import type { Server } from '@/services/api'

// Leaflet world map with per-server markers (online/offline color coding).
// TODO(P5): initialize Leaflet map, plot markers from ip_info lat/lon,
// popups with realtime data, click -> detail page.
defineProps<{
  servers: Server[]
}>()

const mapElement = ref<HTMLDivElement | null>(null)
</script>

<template>
  <div class="world-map">
    <p>TODO P5: Leaflet world map with server markers.</p>
    <div ref="mapElement" class="map-canvas"></div>
  </div>
</template>
