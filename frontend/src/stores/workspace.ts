import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { SelectWorkspace, OpenWorkspace, GetWorkspaceInfo, GetWorkspaceHistory, RemoveWorkspaceFromHistory, OpenInNewWindow, RefreshRemoteWorkspace, GetRemoteWorkspaces, RemoveRemoteWorkspace, OpenRemoteWorkspace, ListRemoteDir } from '../../wailsjs/go/main/App'
import { main, config } from '../../wailsjs/go/models'
import { useFileChangesStore } from './fileChanges'

export const useWorkspaceStore = defineStore('workspace', () => {
  const info = ref<main.WorkspaceInfo | null>(null)
  const history = ref<config.WorkspaceEntry[]>([])
  const remoteList = ref<config.RemoteWorkspaceEntry[]>([])
  const hasWorkspace = computed(() => info.value !== null && info.value.path !== '')

  const previewFiles = ref<string[]>([])
  const activePreviewFile = ref<string | null>(null)

  function openPreviewFile(path: string) {
    if (!previewFiles.value.includes(path)) previewFiles.value.push(path)
    activePreviewFile.value = path
  }
  function closePreviewFile(path: string) {
    const idx = previewFiles.value.indexOf(path)
    if (idx !== -1) previewFiles.value.splice(idx, 1)
    if (activePreviewFile.value === path) {
      activePreviewFile.value = previewFiles.value.length > 0 ? previewFiles.value[previewFiles.value.length - 1] : null
    }
  }

  async function syncChanges() { const fc = useFileChangesStore(); await fc.refresh() }

  async function loadHistory() {
    history.value = (await GetWorkspaceHistory()) || []
    remoteList.value = (await GetRemoteWorkspaces()) || []
  }

  async function selectWorkspace() {
    const r = await SelectWorkspace()
    if (r) { info.value = r; syncChanges(); await loadHistory() }
    return r
  }

  async function openWorkspace(path: string) {
    const r = await OpenWorkspace(path)
    if (r) { info.value = r; syncChanges(); await loadHistory() }
    return r
  }

  async function openRemoteWorkspace(entry: config.RemoteWorkspaceEntry) {
    const r = await OpenRemoteWorkspace(entry as any, entry.remotePath)
    if (r) { info.value = r; syncChanges(); await loadHistory() }
    return r
  }

  async function removeRemote(name: string) {
    await RemoveRemoteWorkspace(name)
    await loadHistory()
  }

  async function openInNewWindow(path: string) { await OpenInNewWindow(path) }
  async function removeFromHistory(path: string) { await RemoveWorkspaceFromHistory(path); await loadHistory() }
  async function refresh() { info.value = await GetWorkspaceInfo() }
  async function refreshRemote() { const r = await RefreshRemoteWorkspace(); if (r) { info.value = r; syncChanges() } }
  async function loadRemoteDir(dir: string) { return await ListRemoteDir(dir) }

  return {
    info, history, remoteList, hasWorkspace,
    previewFiles, activePreviewFile, openPreviewFile, closePreviewFile,
    loadHistory, selectWorkspace, openWorkspace, openRemoteWorkspace, removeRemote,
    openInNewWindow, removeFromHistory, refresh, refreshRemote, loadRemoteDir
  }
})
