<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServersStore } from '@/stores/servers'
import { useTheme } from '@/composables/useTheme'
import { setLocale, type Locale } from '@/i18n'

const { t, locale } = useI18n()
const store = useServersStore()
const { theme, init: initTheme, toggle: toggleTheme } = useTheme()

const siteTitle = computed(() => store.config?.site_title || t('app.title'))
const year = new Date().getFullYear()

function toggleLang(): void {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}

onMounted(async () => {
  await store.loadConfig()
  // Only honor the config theme default when the user hasn't chosen one.
  initTheme(store.config?.theme_default)
  if (!localStorage.getItem('lang') && store.config?.lang_default) {
    const def = store.config.lang_default
    if (def === 'zh' || def === 'en') setLocale(def as Locale)
  }
})
</script>

<template>
  <div id="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="brand" :aria-label="siteTitle">
        <span class="brand-dot" aria-hidden="true"></span>
        <span class="title-text">{{ siteTitle }}</span>
      </RouterLink>

      <nav>
        <RouterLink to="/">{{ t('nav.dashboard') }}</RouterLink>
        <RouterLink to="/admin">{{ t('nav.admin') }}</RouterLink>
      </nav>

      <div class="spacer"></div>

      <div class="controls">
        <button
          type="button"
          class="icon-btn"
          :title="t('common.lang')"
          @click="toggleLang"
        >
          {{ locale === 'zh' ? '中' : 'EN' }}
        </button>
        <button
          type="button"
          class="icon-btn"
          :title="t('common.theme')"
          @click="toggleTheme"
        >
          {{ theme === 'dark' ? '☾' : '☀' }}
        </button>
      </div>
    </header>

    <main>
      <div class="container">
        <RouterView />
      </div>
    </main>

    <footer class="app-footer">{{ siteTitle }} · {{ year }}</footer>
  </div>
</template>
