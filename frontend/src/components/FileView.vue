<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import { Diff } from 'vue-diff/dist/index.es.js'
import 'vue-diff/dist/index.css'
import { GetFileContent, GetFileDiff } from '../../wailsjs/go/main/App'
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

const isChanged = computed(() =>
  fc.changes.some(c => c.path === props.filePath)
)

const noDifference = computed(() =>
  showDiff.value && oldContent.value === newContent.value
)

function detectLang(filePath: string): string {
  const ext = (filePath.split('.').pop() || '').toLowerCase()
  const map: Record<string, string> = {
    ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
    vue: 'html', go: 'go', rs: 'rust', py: 'python', rb: 'ruby',
    css: 'css', scss: 'scss', html: 'xml', json: 'json', xml: 'xml',
    yaml: 'yaml', yml: 'yaml', md: 'markdown', sql: 'sql',
    sh: 'bash', bat: 'dos', c: 'c', cpp: 'cpp', java: 'java',
    kt: 'kotlin', swift: 'swift', php: 'php', lua: 'lua',
  }
  return map[ext] || 'plaintext'
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

async function loadContent() {
  loading.value = true
  showDiff.value = false
  diffError.value = ''
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

watch(() => props.filePath, loadContent, { immediate: true })
</script>

<template>
  <div class="file-view">
    <div class="file-toolbar">
      <span class="file-path">{{ filePath }}</span>
      <button class="btn-diff" :class="{ active: isChanged }" @click="toggleDiff">
        {{ showDiff ? '隐藏差异' : '查看差异' }}
      </button>
    </div>
    <div v-if="loading" class="file-loading">加载中...</div>
    <div v-else-if="diffError" class="file-error">{{ diffError }}</div>
    <div v-else-if="noDifference" class="file-no-diff">
      文件内容与快照一致，无差异
    </div>
    <div v-else-if="showDiff" class="diff-wrap">
      <Diff
        mode="unified"
        theme="dark"
        :language="detectLang(filePath)"
        :prev="oldContent"
        :current="newContent"
        :folding="true"
      />
    </div>
    <pre v-else class="file-content" v-html="highlightedHtml"></pre>
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
.file-loading, .file-error, .file-no-diff {
  padding: 20px;
  color: #8b949e;
}
.file-error { color: #f85149; }
.file-content {
  flex: 1;
  overflow: auto;
  padding: 0;
  margin: 0;
}
.file-content :deep(code) {
  font-family: Consolas, "Courier New", monospace;
  font-size: 13px;
  line-height: 1.5;
  padding: 12px;
  display: block;
}
.diff-wrap {
  flex: 1;
  overflow: auto;
  min-height: 0;
}
.diff-wrap :deep(.vue-diff-wrapper) {
  height: 100%;
}
.diff-wrap :deep(.vue-diff-viewer) {
  height: 100%;
}
</style>
