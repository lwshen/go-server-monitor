/// <reference types="vite/client" />

// Shim for single-file components so TypeScript can resolve `.vue` imports.
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// Typed env vars exposed via import.meta.env.
// TODO(P5): extend with any additional VITE_* config the SPA needs.
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
