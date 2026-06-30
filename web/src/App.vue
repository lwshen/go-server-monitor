<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

// TODO(P5): wire real theme switching (light/dark/terminal) persisted to
// localStorage, and a proper language selector. These handlers are placeholders
// that intentionally do nothing yet.
function toggleTheme(): void {
  // TODO(P5): implement theme toggle (useTheme composable).
  console.log('[App] toggleTheme: not implemented (P5)')
}

function toggleLang(): void {
  // TODO(P5): implement persisted language switch.
  locale.value = locale.value === 'zh' ? 'en' : 'zh'
  console.log('[App] toggleLang: placeholder switch only (P5)')
}
</script>

<template>
  <div id="app-shell">
    <header class="nav-placeholder">
      <span class="brand">{{ t('app.title') }}</span>
      <nav>
        <RouterLink to="/">{{ t('nav.dashboard') }}</RouterLink>
        <RouterLink to="/admin">{{ t('nav.admin') }}</RouterLink>
      </nav>
      <div class="controls">
        <!-- Placeholder toggles — wired in P5. -->
        <button type="button" @click="toggleTheme">theme</button>
        <button type="button" @click="toggleLang">{{ locale }}</button>
      </div>
    </header>

    <main>
      <RouterView />
    </main>

    <!-- TODO(P5): Footer component (version / credits / legal links). -->
  </div>
</template>

<style scoped>
.nav-placeholder {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 1rem;
}
.nav-placeholder .brand {
  font-weight: 600;
}
.nav-placeholder nav {
  display: flex;
  gap: 0.75rem;
}
.nav-placeholder .controls {
  margin-left: auto;
  display: flex;
  gap: 0.5rem;
}
</style>
