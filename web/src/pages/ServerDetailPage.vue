<script setup lang="ts">
import { useRoute } from 'vue-router'

// Single-server detail. Realtime panel + history chart + system/disk info.
// TODO(P5): fetch /api/server?id=, /api/history?id=&range=, subscribe ws scope=<id>.
const route = useRoute()
const serverId = route.params.id
</script>

<template>
  <section class="server-detail-page">
    <h1>Server Detail</h1>
    <p>Server id: {{ serverId }}</p>
    <p>TODO P5: realtime metrics, MetricsChart history, disk/system info.</p>
  </section>
</template>
