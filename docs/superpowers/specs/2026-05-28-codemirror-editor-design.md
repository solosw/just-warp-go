# CodeMirror 6 Editor — Design Spec

## Goal

Replace highlight.js-based read-only file preview with CodeMirror 6, adding edit and save capability for basic code writing.

## Scope

- Syntax highlighting, text editing, save (line numbers, bracket matching, indentation included via CM6 defaults)
- No autocomplete, lint, or advanced IDE features in this iteration

## Backend

### New method: `SaveFile`

Add to `app.go`:

```
SaveFile(path, content string) error
```

- **Local workspace**: `os.WriteFile(filepath.Join(a.workspace, path), []byte(content), 0644)`, then refresh scan + emit changes
- **Remote workspace**: write via SFTP, then refresh scan + emit changes
- Frontend binding auto-generated at `wailsjs/go/main/App.js`

## Frontend

### New dependency

Install codemirror 6 packages:

```
npm install codemirror @codemirror/state @codemirror/view @codemirror/commands @codemirror/language @codemirror/lang-javascript @codemirror/lang-json @codemirror/lang-python @codemirror/lang-html @codemirror/lang-css @codemirror/lang-markdown @codemirror/lang-xml @codemirror/theme-one-dark
```

### New component: `CodeEditor.vue`

Props:
- `modelValue: string` — file content (v-model)
- `language: string` — language ID for syntax highlighting (from existing `detectLang`)
- `readOnly: boolean` — toggle edit mode

Emits:
- `update:modelValue` — content changes
- `save` — Ctrl+S or equivalent

Behavior:
- Create CM6 EditorView with one-dark theme
- Map `language` prop to appropriate `@codemirror/lang-*` extension (fallback: plain text)
- Bind `readOnly` to `EditorView.editable`
- Ctrl+S emits `save` event

### Modified components

**`FilePreviewPanel.vue`:**
- Add `isEditing` ref and `editContent` ref
- Toolbar: add "编辑" / "取消" / "保存" buttons
- View mode: replace `<pre v-html>` with `<CodeEditor :readOnly="true">`
- Edit mode: `<CodeEditor v-model="editContent" @save="handleSave">`
- `handleSave`: call `SaveFile(path, content)`, refresh preview, exit edit mode

**`FileView.vue`:**
- Same pattern as FilePreviewPanel

### Language mapping

Reuse existing `detectLang()` utility. Map its output to CM6 language extensions:
- `js/ts` → `javascript()`
- `json` → `json()`
- `py` → `python()`
- `html` → `html()`
- `css/scss/less` → `css()`
- `md` → `markdown()`
- `xml/svg` → `xml()`
- default → plain text (no language extension)

### Data flow

```
User clicks "编辑" → isEditing=true, editContent=currentContent
User edits in CM6 → v-model updates editContent
User presses Ctrl+S or clicks "保存" → emit 'save' → parent calls SaveFile(path, editContent) → isEditing=false, reload preview
User clicks "取消" → isEditing=false, discard changes

### Error handling

- Save failure: show error message inline in toolbar, keep editor open so user doesn't lose work
- File read failure: show error in preview area (existing behavior)
```
