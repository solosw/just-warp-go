import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetChangedFiles, AcceptAll, RevertAll, AcceptFile, RevertFile, GetFileDiff } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { snapshot } from '../../wailsjs/go/models'
import type { FileDiff } from '../types'

export const useFileChangesStore = defineStore('fileChanges', () => {
  const changes = ref<snapshot.FileChange[]>([])
  const hasChanges = computed(() => changes.value.length > 0)
  const selectedFile = ref<string | null>(null)

  function initListener() {
    EventsOn('file-changes', (data: snapshot.FileChange[]) => {
      changes.value = data || []
    })
  }

  async function refresh() {
    const result = await GetChangedFiles()
    changes.value = result || []
  }

  async function acceptAll() {
    await AcceptAll()
    await refresh()
  }

  async function revertAll() {
    await RevertAll()
    await refresh()
  }

  async function acceptFile(path: string) {
    await AcceptFile(path)
    await refresh()
  }

  async function revertFile(path: string) {
    await RevertFile(path)
    await refresh()
  }

  async function getDiff(path: string): Promise<FileDiff> {
    const result = await GetFileDiff(path)
    return {
      old: result?.old || '',
      new: result?.new || ''
    }
  }

  return { changes, hasChanges, selectedFile, initListener, refresh, acceptAll, revertAll, acceptFile, revertFile, getDiff }
})
