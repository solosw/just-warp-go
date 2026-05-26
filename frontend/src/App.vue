<script setup lang="ts">
import { onMounted } from 'vue'
import { useWorkspaceStore } from './stores/workspace'
import { useFileChangesStore } from './stores/fileChanges'
import { GetStartupWorkspace } from '../wailsjs/go/main/App'
import WorkspaceBar from './components/WorkspaceBar.vue'
import FileTreePanel from './components/FileTreePanel.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import FileChangesPanel from './components/FileChangesPanel.vue'

const ws = useWorkspaceStore()
const fc = useFileChangesStore()

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
      <FileTreePanel v-if="ws.hasWorkspace" />
      <TerminalPanel />
      <FileChangesPanel v-if="ws.hasWorkspace" />
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
</style>
