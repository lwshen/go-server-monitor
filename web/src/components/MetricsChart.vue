<script setup lang="ts">
import { ref } from 'vue'
import type { HistoryPoint } from '@/services/api'

// Chart.js multi-metric history line chart.
// TODO(P5): instantiate Chart.js on the canvas, map history rows to datasets,
// metric toggles (cpu/ram/disk/netIn/netOut/tcp/process), tooltips, reactive update.
defineProps<{
  serverId: string
  data: HistoryPoint[]
  range?: string
}>()

const chartElement = ref<HTMLCanvasElement | null>(null)
</script>

<template>
  <div class="metrics-chart-container">
    <p>TODO P5: Chart.js multi-metric history chart.</p>
    <canvas ref="chartElement"></canvas>
  </div>
</template>
