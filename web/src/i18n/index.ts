import { createI18n } from 'vue-i18n'

import zh from './zh'
import en from './en'

// vue-i18n setup. Composition API mode (legacy: false).
// TODO(P5): persist locale to localStorage and detect browser/default language.
const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  fallbackLocale: 'en',
  messages: {
    zh,
    en,
  },
})

export default i18n
