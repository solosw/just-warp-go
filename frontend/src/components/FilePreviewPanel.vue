<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import DiffView from './DiffView.vue'
import CodeEditor from './CodeEditor.vue'
import { GetFileContent, GetFileDiff, SaveFile } from '../../wailsjs/go/main/App'
import { useWorkspaceStore } from '../stores/workspace'
import { useFileChangesStore } from '../stores/fileChanges'
import { detectLang } from '../utils/detectLang'

const ws = useWorkspaceStore()
const fc = useFileChangesStore()

const cache = ref<Record<string, {
  content: string
  highlightedHtml: string
  loading: boolean
  showDiff: boolean
  oldContent: string
  newContent: string
  isEditing: boolean
  editContent: string
  saveError: string
}>>({})

const activeFile = computed(() => ws.activePreviewFile)
const activeState = computed(() => activeFile.value ? cache.value[activeFile.value] : null)
const isChanged = computed(() => !!activeFile.value && fc.changes.some(c => c.path === activeFile.value))

function getOrCreate(path: string) {
  if (!cache.value[path]) {
    cache.value[path] = {
      content: '', highlightedHtml: '', loading: false, showDiff: false,
      oldContent: '', newContent: '', isEditing: false, editContent: '', saveError: '',
    }
  }
  return cache.value[path]
}

function getFileName(path: string) {
  return path.replace(/\\/g, '/').split('/').pop() || path
}

function highlight(code: string, filePath: string): string {
  if (!code) return ''
  const lang = detectLang(filePath)
  try {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value
    }
    return hljs.highlightAuto(code).value
  } catch {
    return code.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  }
}

async function loadFile(path: string) {
  const st = getOrCreate(path)
  st.loading = true
  st.showDiff = false
  st.isEditing = false
  st.saveError = ''
  try {
    const raw = await GetFileContent(path) || ''
    st.content = raw
    st.highlightedHtml = highlight(raw, path)
  } catch {
    st.highlightedHtml = '<span style="color:#f85149">[无法读取文件]</span>'
  }
  st.loading = false
}

async function toggleDiff() {
  const st = activeState.value
  if (!st || !activeFile.value) return
  if (!st.showDiff) {
    try {
      const diff = await GetFileDiff(activeFile.value)
      st.oldContent = diff?.old ?? ''
      st.newContent = diff?.new ?? ''
      st.showDiff = true
    } catch { }
  } else {
    st.showDiff = false
  }
}

function enterEdit() {
  const st = activeState.value
  if (!st) return
  st.editContent = st.content
  st.isEditing = true
  st.saveError = ''
}

function cancelEdit() {
  const st = activeState.value
  if (!st) return
  st.isEditing = false
  st.editContent = ''
  st.saveError = ''
}

async function handleSave() {
  const st = activeState.value
  const path = activeFile.value
  if (!st || !path) return
  st.saveError = ''
  try {
    await SaveFile(path, st.editContent)
    st.content = st.editContent
    st.highlightedHtml = highlight(st.editContent, path)
    st.isEditing = false
    st.editContent = ''
  } catch (e: any) {
    st.saveError = '保存失败: ' + (e?.message || e)
  }
}

watch(activeFile, (path) => {
  if (path && !cache.value[path]) loadFile(path)
}, { immediate: true })
</script>

<template>
  <div class="preview-panel">
    <div class="panel-header">文件预览</div>
    <div class="tab-bar" v-if="ws.previewFiles.length > 0">
      <div
        v-for="path in ws.previewFiles"
        :key="path"
        class="tab"
        :class="{ active: path === ws.activePreviewFile }"
        @click="ws.activePreviewFile = path"
      >
        <span>{{ getFileName(path) }}</span>
        <button class="tab-close" @click.stop="ws.closePreviewFile(path)">×</button>
      </div>
    </div>

    <div v-if="!activeFile" class="panel-empty">点击文件树查看</div>
    <template v-else>
      <div class="preview-toolbar">
        <span class="preview-path">{{ activeFile }}</span>
        <template v-if="activeState?.isEditing">
          <button class="btn-save" @click="handleSave">保存</button>
          <button class="btn-cancel" @click="cancelEdit">取消</button>
          <span v-if="activeState?.saveError" class="save-error">{{ activeState.saveError }}</span>
        </template>
        <template v-else>
          <button class="btn-edit" @click="enterEdit">编辑</button>
          <button class="btn-diff" :class="{ active: isChanged }" @click="toggleDiff">
            {{ activeState?.showDiff ? '隐藏差异' : '查看差异' }}
          </button>
        </template>
      </div>
      <div v-if="activeState?.loading" class="preview-loading">加载中...</div>
      <div v-else-if="activeState?.isEditing" class="editor-wrap">
        <CodeEditor
          :model-value="activeState!.editContent"
          :language="detectLang(activeFile)"
          :read-only="false"
          @update:model-value="val => activeState && (activeState.editContent = val)"
          @save="handleSave"
        />
      </div>
      <div v-else-if="activeState?.showDiff" class="diff-wrap">
        <DiffView
          :old-string="activeState!.oldContent"
          :new-string="activeState!.newContent"
          :language="detectLang(activeFile)"
          :file-path="activeFile"
        />
      </div>
      <CodeEditor
        v-else
        :model-value="activeState?.content || ''"
        :language="detectLang(activeFile)"
        :read-only="true"
      />
    </template>
  </div>
</template>

<style scoped>
.preview-panel {
  width: 320px;
  background: #0d1117;
  border-left: 1px solid #30363d;
  border-right: 1px solid #2a2a2e;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  flex-shrink: 0;
}
.panel-header {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 600;
  color: #888;
  text-transform: uppercase;
  border-bottom: 1px solid #2a2a2e;
  height: 32px;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.tab-bar {
  display: flex;
  align-items: center;
  background: #1a1a1c;
  border-bottom: 1px solid #333;
  height: 28px;
  padding: 0 4px;
  gap: 2px;
  overflow-x: auto;
  flex-shrink: 0;
}
.tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 4px 4px 0 0;
  cursor: pointer;
  font-size: 11px;
  color: #999;
  white-space: nowrap;
  user-select: none;
}
.tab.active { background: #0d1117; color: #fff; }
.tab:hover { background: #2a2a2e; }
.tab-close {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 14px;
  padding: 0;
  line-height: 1;
}
.tab-close:hover { color: #f44336; }
.panel-empty {
  padding: 20px;
  text-align: center;
  color: #555;
  font-size: 12px;
}
.preview-toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  height: 28px;
  flex-shrink: 0;
}
.preview-path {
  flex: 1;
  font-size: 11px;
  color: #8b949e;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.btn-edit, .btn-save, .btn-cancel {
  background: #21262d;
  border: 1px solid #30363d;
  color: #8b949e;
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 3px;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.btn-edit:hover, .btn-save:hover { color: #58a6ff; border-color: #58a6ff; }
.btn-cancel:hover { color: #f85149; border-color: #f85149; }
.btn-diff {
  background: #21262d;
  border: 1px solid #30363d;
  color: #8b949e;
  font-size: 10px;
  padding: 1px 8px;
  border-radius: 3px;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.btn-diff:hover { color: #d2991d; border-color: #d2991d; }
.btn-diff.active { color: #d2991d; }
.save-error {
  font-size: 10px;
  color: #f85149;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.preview-loading { padding: 20px; color: #8b949e; font-size: 12px; }
.editor-wrap {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.diff-wrap { flex: 1; overflow: auto; min-height: 0; }
</style>
