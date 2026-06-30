import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import i18n from './i18n'

// App bootstrap: wire router, pinia, and i18n, then mount.
// TODO(P5): runtime config init (fetch /api/config), theme restore from
// localStorage, and any global error handling.
const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(i18n)

app.mount('#app')
