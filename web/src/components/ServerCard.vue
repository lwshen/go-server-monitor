<script setup lang="ts">
import type { Server } from '@/services/api'

// Server card for the dashboard grid (name, CPU/RAM/DISK, network quality, traffic).
// TODO(P5): online computation from lastUpdateTime, progress bars, network-quality
// badges (ct/cu/cm/bd), traffic formatting, detail/install actions.
defineProps<{
  server: Server
  lastUpdateTime?: number
}>()
</script>

<template>
  <div class="server-card">
    <h3>{{ server.name }}</h3>
    <p>TODO P5: metrics, network quality, traffic.</p>
  </div>
</template>
