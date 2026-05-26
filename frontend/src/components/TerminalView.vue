<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { useTerminalStore } from '../stores/terminal'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{ tabId: string }>()

const store = useTerminalStore()
const termEl = ref<HTMLDivElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let unsubscribe: (() => void) | null = null

onMounted(async () => {
  await nextTick()
  const el = termEl.value
  if (!el) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Consolas, "Courier New", monospace',
    theme: {
      background: '#161618',
      foreground: '#cccccc',
      cursor: '#ffffff',
      selectionBackground: '#444'
    },
    allowProposedApi: true
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(el)

  requestAnimationFrame(() => {
    if (fitAddon && el.offsetParent !== null) {
      try { fitAddon.fit() } catch {}
    }
  })

  unsubscribe = store.subscribeTerminal(props.tabId, (data: string) => {
    term?.write(data)
  })

  term.onData((data: string) => {
    store.writeToTerminal(props.tabId, data)
  })

  term.onResize(({ cols, rows }) => {
    store.resizeTerminal(props.tabId, cols, rows)
  })

  const observer = new ResizeObserver(() => {
    if (fitAddon && el.offsetParent !== null) {
      try { fitAddon.fit() } catch {}
    }
  })
  observer.observe(el)
})

onUnmounted(() => {
  unsubscribe?.()
  term?.dispose()
})
</script>

<template>
  <div ref="termEl" class="terminal-container"></div>
</template>

<style scoped>
.terminal-container {
  width: 100%;
  height: 100%;
}
</style>
