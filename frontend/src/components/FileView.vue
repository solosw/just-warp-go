<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import DiffView from './DiffView.vue'
import CodeEditor from './CodeEditor.vue'
import { detectLang } from '../utils/detectLang'
import { GetFileContent, GetFileDiff, SaveFile } from '../../wailsjs/go/main/App'
import { useFileChangesStore } from '../stores/fileChanges'

const props = defineProps<{ filePath: string }>()

const fc = useFileChangesStore()
const content = ref('')
const loading = ref(true)
const showDiff = ref(false)
const oldContent = ref('')
const newContent = ref('')
const highlightedHtml = ref('')
const diffError = ref('')
const isEditing = ref(false)
const editContent = ref('')
const saveError = ref('')

const isChanged = computed(() =>
  fc.changes.some(c => c.path === props.filePath)
)

const noDifference = computed(() =>
  showDiff.value && oldContent.value === newContent.value
)

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

async function loadContent() {
  loading.value = true
  showDiff.value = false
  diffError.value = ''
  isEditing.value = false
  saveError.value = ''
  try {
    const raw = await GetFileContent(props.filePath) || ''
    content.value = raw
    highlightedHtml.value = highlight(raw, props.filePath)
  } catch {
    content.value = '[无法读取文件]'
    highlightedHtml.value = '<span style="color:#f85149">[无法读取文件]</span>'
  }
  loading.value = false
}

async function toggleDiff() {
  if (!showDiff.value) {
    diffError.value = ''
    try {
      const diff = await GetFileDiff(props.filePath)
      oldContent.value = diff?.old ?? ''
      newContent.value = diff?.new ?? ''
      showDiff.value = true
    } catch (e: any) {
      diffError.value = '加载差异失败: ' + (e?.message || e)
    }
  } else {
    showDiff.value = false
    diffError.value = ''
  }
}

function enterEdit() {
  editContent.value = content.value
  isEditing.value = true
  saveError.value = ''
}

function cancelEdit() {
  isEditing.value = false
  editContent.value = ''
  saveError.value = ''
}

async function handleSave() {
  saveError.value = ''
  try {
    await SaveFile(props.filePath, editContent.value)
    content.value = editContent.value
    highlightedHtml.value = highlight(editContent.value, props.filePath)
    isEditing.value = false
    editContent.value = ''
  } catch (e: any) {
    saveError.value = '保存失败: ' + (e?.message || e)
  }
}

watch(() => props.filePath, loadContent, { immediate: true })
</script>

<template>
  <div class="file-view">
    <div class="file-toolbar">
      <span class="file-path">{{ filePath }}</span>
      <template v-if="isEditing">
        <button class="btn-save" @click="handleSave">保存</button>
        <button class="btn-cancel" @click="cancelEdit">取消</button>
        <span v-if="saveError" class="save-error">{{ saveError }}</span>
      </template>
      <template v-else>
        <button class="btn-edit" @click="enterEdit">编辑</button>
        <button class="btn-diff" :class="{ active: isChanged }" @click="toggleDiff">
          {{ showDiff ? '隐藏差异' : '查看差异' }}
        </button>
      </template>
    </div>
    <div v-if="loading" class="file-loading">加载中...</div>
    <div v-else-if="diffError" class="file-error">{{ diffError }}</div>
    <div v-else-if="noDifference" class="file-no-diff">
      文件内容与快照一致，无差异
    </div>
    <div v-else-if="isEditing" class="editor-wrap">
      <CodeEditor
        :model-value="editContent"
        :language="detectLang(filePath)"
        :read-only="false"
        @update:model-value="val => editContent = val"
        @save="handleSave"
      />
    </div>
    <div v-else-if="showDiff" class="diff-wrap">
      <DiffView
        :old-string="oldContent"
        :new-string="newContent"
        :language="detectLang(filePath)"
        :file-path="filePath"
      />
    </div>
    <CodeEditor
      v-else
      :model-value="content"
      :language="detectLang(filePath)"
      :read-only="true"
    />
  </div>
</template>

<style scoped>
.file-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #0d1117;
}
.file-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  height: 30px;
  flex-shrink: 0;
}
.file-path {
  flex: 1;
  font-size: 12px;
  color: #8b949e;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.btn-edit, .btn-save, .btn-cancel {
  background: #21262d;
  border: 1px solid #30363d;
  color: #8b949e;
  font-size: 11px;
  padding: 2px 10px;
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}
.btn-edit:hover, .btn-save:hover { color: #58a6ff; border-color: #58a6ff; }
.btn-cancel:hover { color: #f85149; border-color: #f85149; }
.btn-diff {
  background: #21262d;
  border: 1px solid #30363d;
  color: #d2991d;
  font-size: 11px;
  padding: 2px 10px;
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}
.btn-diff:hover { background: #30363d; }
.save-error {
  font-size: 10px;
  color: #f85149;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.file-loading, .file-error, .file-no-diff { padding: 20px; color: #8b949e; }
.file-error { color: #f85149; }
.editor-wrap {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.diff-wrap { flex: 1; overflow: auto; min-height: 0; }
</style>
