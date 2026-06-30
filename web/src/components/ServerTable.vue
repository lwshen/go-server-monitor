<script setup lang="ts">
import type { Server } from '@/services/api'

// Compact table view (dashboard table mode + admin server management).
// TODO(P5/P6): columns, online status, selection, edit/delete row actions.
defineProps<{
  servers: Server[]
}>()

defineEmits<{
  edit: [server: Server]
  delete: [id: string]
}>()
</script>

<template>
  <table class="server-table">
    <thead>
      <tr>
        <th>Name</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="s in servers" :key="s.id">
        <td>{{ s.name }}</td>
      </tr>
    </tbody>
  </table>
  <!-- TODO P5/P6: full columns, sorting, selection, actions. -->
</template>
