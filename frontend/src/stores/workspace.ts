import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { SelectWorkspace, OpenWorkspace, GetWorkspaceInfo, GetWorkspaceHistory, RemoveWorkspaceFromHistory } from '../../wailsjs/go/main/App'
import { main, config } from '../../wailsjs/go/models'

export const useWorkspaceStore = defineStore('workspace', () => {
  const info = ref<main.WorkspaceInfo | null>(null)
  const history = ref<config.WorkspaceEntry[]>([])
  const hasWorkspace = computed(() => info.value !== null && info.value.path !== '')

  async function loadHistory() {
    const result = await GetWorkspaceHistory()
    history.value = result || []
  }

  async function selectWorkspace() {
    const result = await SelectWorkspace()
    if (result) {
      info.value = result
      await loadHistory()
    }
    return result
  }

  async function openWorkspace(path: string) {
    const result = await OpenWorkspace(path)
    if (result) {
      info.value = result
      await loadHistory()
    }
    return result
  }

  async function removeFromHistory(path: string) {
    await RemoveWorkspaceFromHistory(path)
    await loadHistory()
  }

  async function refresh() {
    info.value = await GetWorkspaceInfo()
  }

  return { info, history, hasWorkspace, loadHistory, selectWorkspace, openWorkspace, removeFromHistory, refresh }
})
