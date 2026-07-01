// vue-i18n setup (Composition API). Locale persists in localStorage('lang')
// and defaults from the <html lang> attribute, falling back to zh.

import { createI18n } from 'vue-i18n'
import en from './en'
import zh from './zh'

export type Locale = 'zh' | 'en'
const LANG_KEY = 'lang'

function normalize(v: string | null | undefined): Locale | null {
  return v === 'zh' || v === 'en' ? v : null
}

function initialLocale(): Locale {
  const stored = normalize(localStorage.getItem(LANG_KEY))
  if (stored) return stored
  const htmlLang = normalize(document.documentElement.lang)
  return htmlLang ?? 'zh'
}

const i18n = createI18n({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: 'en',
  messages: { zh, en },
})

/** Switch and persist the active locale. */
export function setLocale(locale: Locale): void {
  i18n.global.locale.value = locale
  localStorage.setItem(LANG_KEY, locale)
  document.documentElement.lang = locale
}

export default i18n
