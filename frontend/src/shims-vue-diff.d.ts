declare module 'vue-diff/dist/index.es.js' {
  import type { DefineComponent } from 'vue'
  export const Diff: DefineComponent<{
    mode?: 'split' | 'unified'
    theme?: 'dark' | 'light'
    language?: string
    prev?: string
    current?: string
    folding?: boolean
    inputDelay?: number
    virtualScroll?: boolean | Record<string, unknown>
  }>
}
