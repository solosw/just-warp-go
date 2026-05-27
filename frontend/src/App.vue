<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useWorkspaceStore } from './stores/workspace'
import { useFileChangesStore } from './stores/fileChanges'
import { GetStartupWorkspace } from '../wailsjs/go/main/App'
import WorkspaceBar from './components/WorkspaceBar.vue'
import FileTreePanel from './components/FileTreePanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import FilePreviewPanel from './components/FilePreviewPanel.vue'
import FileChangesPanel from './components/FileChangesPanel.vue'

const ws = useWorkspaceStore()
const fc = useFileChangesStore()

// Resizable panel widths
const treeWidth = ref(220)
const previewWidth = ref(320)
const changesWidth = ref(280)

function startResize(target: 'tree' | 'preview' | 'changes') {
  const onMove = (e: MouseEvent) => {
    if (target === 'tree') {
      treeWidth.value = Math.max(140, Math.min(400, e.clientX - 4))
    } else if (target === 'changes') {
      changesWidth.value = Math.max(180, Math.min(500, window.innerWidth - e.clientX - 4))
    } else if (target === 'preview') {
      previewWidth.value = Math.max(200, Math.min(600, e.clientX - treeWidth.value - 12))
    }
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

onMounted(async () => {
  ws.loadHistory()
  fc.initListener()
  const startupWs = await GetStartupWorkspace()
  if (startupWs) {
    await ws.openWorkspace(startupWs)
  }
})
</script>

<template>
  <div class="app-layout">
    <WorkspaceBar />
    <div class="main-area">
      <FileTreePanel v-if="ws.hasWorkspace" :style="{ width: treeWidth + 'px' }" />
      <div
        v-if="ws.hasWorkspace"
        class="resize-handle"
        @mousedown="startResize('tree')"
      ></div>
      <TerminalPanel />
      <div
        v-if="ws.hasWorkspace"
        class="resize-handle"
        @mousedown="startResize('preview')"
      ></div>
      <FilePreviewPanel v-if="ws.hasWorkspace" :style="{ width: previewWidth + 'px' }" />
      <div
        v-if="ws.hasWorkspace"
        class="resize-handle"
        @mousedown="startResize('changes')"
      ></div>
      <FileChangesPanel v-if="ws.hasWorkspace" :style="{ width: changesWidth + 'px' }" />
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}
.main-area {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.resize-handle {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  transition: background 0.15s;
  flex-shrink: 0;
  z-index: 10;
}
.resize-handle:hover {
  background: #58a6ff;
}
</style>
