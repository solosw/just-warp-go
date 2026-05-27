import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { CreateTerminal, WriteToTerminal, ResizeTerminal, CloseTerminal } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export interface TabItem {
  id: string
  title: string
}

export const useTerminalStore = defineStore('terminal', () => {
  const tabs = ref<TabItem[]>([])
  const activeTabId = ref<string | null>(null)
  const activeTab = computed(() => tabs.value.find(t => t.id === activeTabId.value) || null)
  const error = ref<string | null>(null)
  let counter = 0

  async function createTerminal(): Promise<TabItem | null> {
    error.value = null
    try {
      const id = await CreateTerminal()
      if (!id) {
        error.value = '创建终端失败'
        return null
      }
      counter++
      const tab: TabItem = { id, title: `终端 ${counter}` }
      tabs.value.push(tab)
      activeTabId.value = id
      return tab
    } catch (e: any) {
      error.value = '创建终端失败: ' + (e?.message || e)
      return null
    }
  }

  async function closeTab(id: string) {
    try { await CloseTerminal(id) } catch {}
    EventsOff('terminal-output:' + id)
    const idx = tabs.value.findIndex(t => t.id === id)
    if (idx !== -1) tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value.length > 0 ? tabs.value[tabs.value.length - 1].id : null
    }
  }

  function setActive(id: string) { activeTabId.value = id }

  function addSSHTab(id: string, title: string) {
    tabs.value.push({ id, title })
    activeTabId.value = id
  }

  function writeToTerminal(tabId: string, data: string) { WriteToTerminal(tabId, data) }
  function resizeTerminal(tabId: string, cols: number, rows: number) { ResizeTerminal(tabId, cols, rows) }

  function subscribeTerminal(id: string, handler: (data: string) => void): () => void {
    const eventName = 'terminal-output:' + id
    EventsOn(eventName, handler)
    return () => EventsOff(eventName)
  }

  const layoutMode = ref<'tabs' | 'grid'>('tabs')
  function toggleLayout() { layoutMode.value = layoutMode.value === 'tabs' ? 'grid' : 'tabs' }

  return {
    tabs, activeTabId, activeTab, error,
    layoutMode, toggleLayout,
    createTerminal, addSSHTab, closeTab, setActive,
    writeToTerminal, resizeTerminal, subscribeTerminal
  }
})
