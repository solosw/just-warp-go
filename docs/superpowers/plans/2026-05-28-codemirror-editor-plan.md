# CodeMirror 6 Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace highlight.js-based read-only file preview with CodeMirror 6 editor, adding edit and save capability.

**Architecture:** Add `SaveFile` to Go backend for writing files. Create a reusable `CodeEditor.vue` component wrapping CM6 with two-tier language support (13 official CM6 lang packages + 30+ legacy modes). Update `FilePreviewPanel.vue` and `FileView.vue` to toggle between view (read-only CM6) and edit mode (writable CM6 with save).

**Tech Stack:** Wails v2 (Go), Vue 3 + Composition API + TypeScript, CodeMirror 6, Pinia

---

## File Structure

| Action | Path | Responsibility |
|--------|------|----------------|
| Modify | `app.go` | Add `SaveFile` Go method |
| CREATE | `frontend/src/utils/langMapping.ts` | Map detectLang output → CM6 language extension |
| CREATE | `frontend/src/components/CodeEditor.vue` | Reusable CM6 editor wrapper |
| Modify | `frontend/src/components/FilePreviewPanel.vue` | Add edit/save mode |
| Modify | `frontend/src/components/FileView.vue` | Add edit/save mode |

---

### Task 1: Add SaveFile to Go backend

**Files:**
- Modify: `app.go`

- [ ] **Step 1: Add `SaveFile` method to `app.go`**

Insert after the existing `GetFileContent` method (after line 739):

```go
func (a *App) SaveFile(path, content string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.workspace == "" || a.snapEng == nil {
		return fmt.Errorf("未选择工作区")
	}
	if a.isRemote {
		if a.remoteSFTP == nil {
			return fmt.Errorf("远程连接不可用")
		}
		rp := path.Join(a.remotePath, path)
		f, err := a.remoteSFTP.Create(rp)
		if err != nil {
			return fmt.Errorf("写入远程文件失败: %w", err)
		}
		defer f.Close()
		if _, err := f.Write([]byte(content)); err != nil {
			return fmt.Errorf("写入远程文件失败: %w", err)
		}
		// Update manifest hash for the saved file
		newHash := snapshot.HashBytes([]byte(content))
		_ = a.snapEng.AcceptHashes(map[string]string{path: newHash})
		a.refreshScanLocked()
		a.emitChanges()
		return nil
	}
	fullPath := filepath.Join(a.workspace, path)
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}
	a.refreshScanLocked()
	a.emitChanges()
	return nil
}
```

- [ ] **Step 2: Verify the Go code compiles**

```bash
cd C:/Users/solosw/Desktop/warp-go && go build -o /dev/null ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add SaveFile method to Go backend"
```

---

### Task 2: Install CodeMirror 6 dependencies

**Files:**
- Modify: `frontend/package.json` (npm will update)

- [ ] **Step 1: Run npm install**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npm install codemirror @codemirror/lang-javascript @codemirror/lang-json @codemirror/lang-python @codemirror/lang-html @codemirror/lang-css @codemirror/lang-markdown @codemirror/lang-xml @codemirror/lang-cpp @codemirror/lang-java @codemirror/lang-go @codemirror/lang-rust @codemirror/lang-php @codemirror/lang-sql @codemirror/legacy-modes @codemirror/theme-one-dark
```

Expected: packages installed successfully, `package.json` updated.

- [ ] **Step 2: Verify installation**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && node -e "require('codemirror'); console.log('OK')"
```

Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add frontend/package.json frontend/package-lock.json
git commit -m "chore: install codemirror 6 and language packages"
```

---

### Task 3: Create language mapping utility

**Files:**
- Create: `frontend/src/utils/langMapping.ts`

- [ ] **Step 1: Write `langMapping.ts`**

```typescript
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { python } from '@codemirror/lang-python'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { markdown } from '@codemirror/lang-markdown'
import { xml } from '@codemirror/lang-xml'
import { cpp } from '@codemirror/lang-cpp'
import { java } from '@codemirror/lang-java'
import { go } from '@codemirror/lang-go'
import { rust } from '@codemirror/lang-rust'
import { php } from '@codemirror/lang-php'
import { sql, PostgreSQL } from '@codemirror/lang-sql'
import { StreamLanguage } from '@codemirror/language'
// Legacy modes (static imports for Vite ESM compatibility)
import { yaml } from '@codemirror/legacy-modes/mode/yaml'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { ruby } from '@codemirror/legacy-modes/mode/ruby'
import { swift } from '@codemirror/legacy-modes/mode/swift'
import { objectiveC } from '@codemirror/legacy-modes/mode/objective-c'
import { kotlin } from '@codemirror/legacy-modes/mode/kotlin'
import { scala } from '@codemirror/legacy-modes/mode/scala'
import { groovy } from '@codemirror/legacy-modes/mode/groovy'
import { cSharp } from '@codemirror/legacy-modes/mode/c-sharp'
import { lua } from '@codemirror/legacy-modes/mode/lua'
import { r } from '@codemirror/legacy-modes/mode/r'
import { dart } from '@codemirror/legacy-modes/mode/dart'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import { powershell } from '@codemirror/legacy-modes/mode/powershell'
import { mscgen } from '@codemirror/legacy-modes/mode/mscgen'
import { dockerfile } from '@codemirror/legacy-modes/mode/dockerfile'
import { cmake } from '@codemirror/legacy-modes/mode/cmake'
import { graphql } from '@codemirror/legacy-modes/mode/graphql'
import { protobuf } from '@codemirror/legacy-modes/mode/protobuf'
import { stex } from '@codemirror/legacy-modes/mode/stex'
import { elm } from '@codemirror/legacy-modes/mode/elm'
import { erlang } from '@codemirror/legacy-modes/mode/erlang'
import { elixir } from '@codemirror/legacy-modes/mode/elixir'
import { haskell } from '@codemirror/legacy-modes/mode/haskell'
import { clojure } from '@codemirror/legacy-modes/mode/clojure'
import { fortran } from '@codemirror/legacy-modes/mode/fortran'
import { perl } from '@codemirror/legacy-modes/mode/perl'
import { scheme } from '@codemirror/legacy-modes/mode/scheme'
import { vim } from '@codemirror/legacy-modes/mode/vim'
import type { Extension } from '@codemirror/state'

// Official CM6 language packages (tree-sitter grammars)
const officialMap: Record<string, () => Extension> = {
  javascript,
  typescript: javascript,
  json,
  python,
  html,
  css,
  scss: css,
  less: css,
  markdown,
  xml,
  c: cpp,
  cpp,
  java,
  go,
  rust,
  php,
  sql,
  pgsql: () => sql({ dialect: PostgreSQL }),
}

// Legacy mode mapping (StreamLanguage wrappers, static imports above)
const legacyMap: Record<string, any> = {
  yaml,
  toml,
  ini: properties,
  ruby,
  swift,
  objectivec: objectiveC,
  kotlin,
  scala,
  groovy,
  csharp: cSharp,
  lua,
  r,
  dart,
  bash: shell,
  fish: shell,
  powershell,
  dos: mscgen,
  dockerfile,
  makefile: cmake,
  cmake,
  graphql,
  protobuf,
  latex: stex,
  elm,
  erlang,
  elixir,
  haskell,
  clojure,
  fortran,
  perl,
  scheme,
  vim,
}

const legacyCache: Record<string, Extension> = {}

function loadLegacy(lang: string): Extension | null {
  if (legacyCache[lang]) return legacyCache[lang]
  const mode = legacyMap[lang]
  if (!mode) return null
  const ext = StreamLanguage.define(mode)
  legacyCache[lang] = ext
  return ext
}

export function getLanguageExtension(lang: string): Extension | null {
  if (!lang || lang === 'plaintext') return null
  const fn = officialMap[lang]
  if (fn) return fn()
  return loadLegacy(lang)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npx vue-tsc --noEmit src/utils/langMapping.ts
```

Expected: no type errors. (May show unrelated project errors; ensure `langMapping.ts` itself has none.)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/utils/langMapping.ts
git commit -m "feat: add CM6 language mapping utility with official + legacy mode support"
```

---

### Task 4: Create CodeEditor.vue component

**Files:**
- Create: `frontend/src/components/CodeEditor.vue`

- [ ] **Step 1: Write `CodeEditor.vue`**

```vue
<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, drawSelection, rectangularSelection } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { closeBrackets } from '@codemirror/autocomplete'
import { oneDark } from '@codemirror/theme-one-dark'
import { getLanguageExtension } from '../utils/langMapping'

const props = defineProps<{
  modelValue: string
  language: string
  readOnly: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'save'): void
}>()

const editorRef = ref<HTMLElement>()
const view = shallowRef<EditorView>()

onMounted(() => {
  const langExt = getLanguageExtension(props.language)

  const saveHandler = keymap.of([{
    key: 'Mod-s',
    run: () => {
      emit('save')
      return true
    },
    preventDefault: true,
  }])

  const updateListener = EditorView.updateListener.of(update => {
    if (update.docChanged) {
      emit('update:modelValue', update.state.doc.toString())
    }
  })

  const state = EditorState.create({
    doc: props.modelValue,
    extensions: [
      lineNumbers(),
      highlightActiveLine(),
      drawSelection(),
      rectangularSelection(),
      history(),
      bracketMatching(),
      closeBrackets(),
      indentOnInput(),
      defaultKeymap,
      historyKeymap,
      indentWithTab,
      saveHandler,
      updateListener,
      syntaxHighlighting(defaultHighlightStyle),
      oneDark,
      EditorView.editable.of(!props.readOnly),
      EditorState.tabSize.of(2),
      ...(langExt ? [langExt] : []),
    ],
  })

  const editor = new EditorView({
    state,
    parent: editorRef.value!,
  })

  view.value = editor
})

watch(() => props.modelValue, (newVal) => {
  const editor = view.value
  if (!editor) return
  const current = editor.state.doc.toString()
  if (newVal !== current) {
    editor.dispatch({
      changes: { from: 0, to: current.length, insert: newVal },
    })
  }
})

watch(() => props.readOnly, (val) => {
  view.value?.dispatch({
    effects: EditorView.editable.reconfigure(EditorView.editable.of(!val)),
  })
})

onUnmounted(() => {
  view.value?.destroy()
})
</script>

<template>
  <div ref="editorRef" class="cm-editor-wrap"></div>
</template>

<style scoped>
.cm-editor-wrap {
  flex: 1;
  overflow: auto;
}
.cm-editor-wrap :deep(.cm-editor) {
  height: 100%;
}
.cm-editor-wrap :deep(.cm-scroller) {
  font-family: Consolas, "Courier New", monospace;
  font-size: 13px;
  line-height: 1.5;
}
.cm-editor-wrap :deep(.cm-content) {
  padding: 8px 0;
}
</style>
```

- [ ] **Step 2: Verify compilation**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npx vue-tsc --noEmit src/components/CodeEditor.vue
```

Expected: no type errors from this file.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/CodeEditor.vue
git commit -m "feat: add CodeEditor.vue — reusable CM6 editor component"
```

---

### Task 5: Update FilePreviewPanel.vue with edit mode

**Files:**
- Modify: `frontend/src/components/FilePreviewPanel.vue`

- [ ] **Step 1: Replace the script section**

Replace lines 1-85 (the entire `<script setup>` block) with:

```vue
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
})
</script>
```

- [ ] **Step 2: Replace the template section**

Replace lines 87-123 (the entire `<template>` block) with:

```vue
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
```

- [ ] **Step 3: Replace the style section**

Replace lines 126-236 (the entire `<style scoped>` block) with:

```css
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
```

- [ ] **Step 4: Verify compilation**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npx vue-tsc --noEmit
```

Expected: no type errors. Fix any that appear.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/FilePreviewPanel.vue
git commit -m "feat: add edit/save mode to FilePreviewPanel using CodeEditor"
```

---

### Task 6: Update FileView.vue with edit mode

**Files:**
- Modify: `frontend/src/components/FileView.vue`

- [ ] **Step 1: Replace the script section**

Replace lines 1-75 (the entire `<script setup>` block) with:

```vue
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
```

- [ ] **Step 2: Replace the template section**

Replace lines 77-99 (the entire `<template>` block) with:

```vue
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
```

- [ ] **Step 3: Replace the style section**

Replace lines 102-155 (the entire `<style scoped>` block) with:

```css
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
```

- [ ] **Step 4: Verify full project compilation**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npx vue-tsc --noEmit
```

Expected: no type errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/FileView.vue
git commit -m "feat: add edit/save mode to FileView using CodeEditor"
```

---

### Task 7: Build and integration test

**Files:** None (build verification only)

- [ ] **Step 1: Build the frontend**

```bash
cd C:/Users/solosw/Desktop/warp-go/frontend && npm run build
```

Expected: build succeeds, no errors.

- [ ] **Step 2: Build the Wails app**

```bash
cd C:/Users/solosw/Desktop/warp-go && wails build
```

Expected: build succeeds, binary produced.

- [ ] **Step 3: Manual smoke test**

Launch the app and verify:
1. Open a workspace → click a file in the tree → file content renders with CM6 syntax highlighting (not highlight.js)
2. Click "编辑" → editor becomes editable, "保存" and "取消" buttons appear
3. Type code, Ctrl+S → file saves, returns to read-only view
4. Click "取消" → discards changes, returns to read-only view
5. Click "查看差异" → diff view still works
6. Edit a file with unsaved changes → "保存" → verify content persists by re-opening

- [ ] **Step 4: Commit any final fixes**

```bash
git add -A
git commit -m "chore: final adjustments after integration test"
```
