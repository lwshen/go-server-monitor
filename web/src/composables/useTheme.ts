// Theme composable. Applies the theme via `document.documentElement.dataset.theme`
// so style.css can key off [data-theme=...]. Choice persists in localStorage and
// falls back to a config default or "dark".

import { ref } from 'vue'

export type Theme = 'dark' | 'light'

const THEME_KEY = 'theme'
const theme = ref<Theme>('dark')

function apply(t: Theme): void {
  document.documentElement.dataset.theme = t
}

function normalize(v: string | null | undefined): Theme | null {
  return v === 'dark' || v === 'light' ? v : null
}

export function useTheme() {
  /**
   * Initialize from localStorage, then a config default, then "dark".
   * Call once at app start (pass config.theme_default when available).
   */
  function init(configDefault?: string): void {
    const stored = normalize(localStorage.getItem(THEME_KEY))
    theme.value = stored ?? normalize(configDefault) ?? 'dark'
    apply(theme.value)
  }

  function setTheme(t: Theme): void {
    theme.value = t
    apply(t)
    localStorage.setItem(THEME_KEY, t)
  }

  function toggle(): void {
    setTheme(theme.value === 'dark' ? 'light' : 'dark')
  }

  return { theme, init, setTheme, toggle }
}
