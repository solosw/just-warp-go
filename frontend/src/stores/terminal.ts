import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { CreateTerminal, WriteToTerminal, ResizeTerminal, CloseTerminal } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export interface TabItem {
  id: string
  type: 'terminal' | 'file'
  title: string
  filePath?: string
}

export const useTerminalStore = defineStore('terminal', () => {
  const tabs = ref<TabItem[]>([])
  const activeTabId = ref<string | null>(null)
  const activeTab = computed(() => tabs.value.find(t => t.id === activeTabId.value) || null)
  const error = ref<string | null>(null)

  let terminalCounter = 0
  let fileCounter = 0

  async function createTerminal(): Promise<TabItem | null> {
    error.value = null
    try {
      const id = await CreateTerminal()
      if (!id) {
        error.value = '创建终端失败'
        return null
      }
      terminalCounter++
      const tab: TabItem = { id, type: 'terminal', title: `终端 ${terminalCounter}` }
      tabs.value.push(tab)
      activeTabId.value = id
      return tab
    } catch (e: any) {
      error.value = '创建终端失败: ' + (e?.message || e)
      return null
    }
  }

  function openFile(filePath: string) {
    // Check if already open
    const existing = tabs.value.find(t => t.type === 'file' && t.filePath === filePath)
    if (existing) {
      activeTabId.value = existing.id
      return existing
    }
    fileCounter++
    const name = filePath.replace(/\\/g, '/').split('/').pop() || filePath
    const id = 'file-' + fileCounter + '-' + Date.now()
    const tab: TabItem = { id, type: 'file', title: name, filePath }
    tabs.value.push(tab)
    activeTabId.value = id
    return tab
  }

  async function closeTab(id: string) {
    const tab = tabs.value.find(t => t.id === id)
    if (!tab) return
    if (tab.type === 'terminal') {
      try { await CloseTerminal(id) } catch {}
      EventsOff('terminal-output:' + id)
    }
    const idx = tabs.value.findIndex(t => t.id === id)
    if (idx !== -1) tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value.length > 0 ? tabs.value[tabs.value.length - 1].id : null
    }
  }

  function setActive(id: string) {
    activeTabId.value = id
  }

  // Terminal-specific methods
  function writeToTerminal(tabId: string, data: string) {
    WriteToTerminal(tabId, data)
  }

  function resizeTerminal(tabId: string, cols: number, rows: number) {
    ResizeTerminal(tabId, cols, rows)
  }

  function subscribeTerminal(id: string, handler: (data: string) => void): () => void {
    const eventName = 'terminal-output:' + id
    EventsOn(eventName, handler)
    return () => EventsOff(eventName)
  }

  const layoutMode = ref<'tabs' | 'grid'>('tabs')
  function toggleLayout() {
    layoutMode.value = layoutMode.value === 'tabs' ? 'grid' : 'tabs'
  }

  return {
    tabs, activeTabId, activeTab, error,
    layoutMode, toggleLayout,
    createTerminal, openFile, closeTab, setActive,
    writeToTerminal, resizeTerminal, subscribeTerminal
  }
})
