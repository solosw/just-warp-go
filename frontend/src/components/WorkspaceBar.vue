<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'

const ws = useWorkspaceStore()
const showDropdown = ref(false)
const dropdownRef = ref<HTMLElement>()

function toggleDropdown() {
  showDropdown.value = !showDropdown.value
}

function closeDropdown() {
  showDropdown.value = false
}

function onDocClick(e: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

async function selectFromHistory(path: string) {
  closeDropdown()
  // If selecting the same workspace, stay; otherwise open in new window
  if (ws.info?.path === path) return
  if (ws.hasWorkspace) {
    await ws.openInNewWindow(path)
  } else {
    await ws.openWorkspace(path)
  }
}

async function removeHistory(path: string, e: Event) {
  e.stopPropagation()
  await ws.removeFromHistory(path)
}
</script>

<template>
  <div class="workspace-bar">
    <div ref="dropdownRef" class="dropdown-wrapper">
      <button class="btn-select" @click="toggleDropdown">
        <span class="icon">&#x1F4C1;</span>
        <span>{{ ws.hasWorkspace ? ws.info?.name : '选择工作区...' }}</span>
        <span class="arrow">&#x25BE;</span>
      </button>

      <div v-if="showDropdown" class="dropdown-menu">
        <div class="dropdown-header">工作区历史</div>
        <div v-if="ws.history.length === 0" class="dropdown-empty">
          暂无历史工作区
        </div>
        <div
          v-for="entry in ws.history"
          :key="entry.path"
          class="dropdown-item"
          :class="{ active: ws.info?.path === entry.path }"
          @click="selectFromHistory(entry.path)"
        >
          <span class="item-name">{{ entry.name }}</span>
          <span class="item-path">{{ entry.path }}</span>
          <button class="item-remove" @click="(e: Event) => removeHistory(entry.path, e)" title="移除">&times;</button>
        </div>
        <div class="dropdown-footer" @click="ws.selectWorkspace(); closeDropdown()">
          + 浏览文件夹...
        </div>
      </div>
    </div>

    <div v-if="ws.hasWorkspace" class="workspace-info">
      <span class="file-count">{{ ws.info?.fileCount }} 个文件</span>
    </div>
  </div>
</template>

<style scoped>
.workspace-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px;
  background: #1a1a1c;
  border-bottom: 1px solid #333;
  height: 36px;
  position: relative;
  z-index: 50;
}
.dropdown-wrapper {
  position: relative;
}
.btn-select {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #2a2a2e;
  border: 1px solid #444;
  color: #ccc;
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  max-width: 320px;
  overflow: hidden;
  white-space: nowrap;
}
.btn-select:hover {
  background: #3a3a3e;
}
.arrow {
  font-size: 10px;
  color: #888;
}
.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  width: 420px;
  max-height: 360px;
  overflow-y: auto;
  background: #1e1e20;
  border: 1px solid #444;
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.5);
  z-index: 100;
}
.dropdown-header {
  padding: 8px 14px;
  font-size: 11px;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid #333;
}
.dropdown-empty {
  padding: 16px;
  text-align: center;
  color: #555;
  font-size: 13px;
}
.dropdown-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  cursor: pointer;
  border-bottom: 1px solid #222;
  font-size: 13px;
}
.dropdown-item:hover {
  background: #2a2a2e;
}
.dropdown-item.active {
  background: #1a3a5c;
}
.item-name {
  font-weight: 600;
  color: #ddd;
  flex-shrink: 0;
}
.item-path {
  color: #888;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.item-remove {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  font-size: 16px;
  padding: 0 2px;
  line-height: 1;
  flex-shrink: 0;
}
.item-remove:hover {
  color: #f44336;
}
.dropdown-footer {
  padding: 8px 14px;
  font-size: 13px;
  color: #4a9eff;
  cursor: pointer;
  border-top: 1px solid #333;
}
.dropdown-footer:hover {
  background: #2a2a2e;
}
.workspace-info {
  font-size: 12px;
  color: #888;
}
.file-count {
  background: #333;
  padding: 2px 8px;
  border-radius: 10px;
}
</style>
