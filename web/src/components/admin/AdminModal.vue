<script setup lang="ts">
// Lightweight modal shell for admin dialogs. Renders a dimmed backdrop and a
// centered panel; closes on backdrop click, the × button, or Escape.
import { onMounted, onBeforeUnmount } from 'vue'

defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()

function onKey(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')">
    <div class="modal-panel card" role="dialog" aria-modal="true">
      <header class="modal-head row between">
        <h3 class="modal-title">{{ title }}</h3>
        <button type="button" class="icon-btn" :aria-label="'close'" @click="emit('close')">✕</button>
      </header>
      <div class="modal-body">
        <slot />
      </div>
      <footer v-if="$slots.footer" class="modal-foot row gap-2">
        <slot name="footer" />
      </footer>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-4);
  background: color-mix(in srgb, #000 55%, transparent);
  backdrop-filter: blur(3px);
}
.modal-panel {
  width: 100%;
  max-width: 520px;
  max-height: calc(100vh - var(--sp-8));
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
  box-shadow: var(--shadow);
  border-color: var(--border-strong);
}
.modal-head {
  border-bottom: 1px solid var(--border);
  padding-bottom: var(--sp-3);
}
.modal-title {
  font-size: var(--fs-lg);
}
.modal-body {
  overflow-y: auto;
}
.modal-foot {
  justify-content: flex-end;
  border-top: 1px solid var(--border);
  padding-top: var(--sp-3);
}
</style>
