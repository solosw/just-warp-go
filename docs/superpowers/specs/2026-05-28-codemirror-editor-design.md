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
npm install codemirror @codemirror/state @codemirror/view @codemirror/commands @codemirror/language @codemirror/theme-one-dark \
  @codemirror/lang-javascript @codemirror/lang-json @codemirror/lang-python @codemirror/lang-html @codemirror/lang-css \
  @codemirror/lang-markdown @codemirror/lang-xml @codemirror/lang-cpp @codemirror/lang-java @codemirror/lang-go \
  @codemirror/lang-rust @codemirror/lang-php @codemirror/lang-sql @codemirror/legacy-modes
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

Reuse existing `detectLang()` utility. Two-tier mapping:

**Official CM6 packages** (first-class support, tree-sitter grammar):
| detectLang output | CM6 extension |
|---|---|
| `javascript`, `typescript` | `@codemirror/lang-javascript` |
| `json` | `@codemirror/lang-json` |
| `python` | `@codemirror/lang-python` |
| `html` | `@codemirror/lang-html` |
| `css`, `scss`, `less` | `@codemirror/lang-css` |
| `markdown` | `@codemirror/lang-markdown` |
| `xml` | `@codemirror/lang-xml` |
| `c`, `cpp` | `@codemirror/lang-cpp` |
| `java` | `@codemirror/lang-java` |
| `go` | `@codemirror/lang-go` |
| `rust` | `@codemirror/lang-rust` |
| `php` | `@codemirror/lang-php` |
| `sql`, `pgsql` | `@codemirror/lang-sql` |

**Legacy modes** (StreamLanguage wrapper, covers the rest):
`yaml`, `toml`, `ruby`, `swift`, `objectivec`, `kotlin`, `scala`, `groovy`,
`csharp`, `lua`, `r`, `dart`, `bash`, `fish`, `powershell`, `dos`,
`dockerfile`, `makefile`, `cmake`, `ini`, `graphql`, `protobuf`, `latex`,
`elm`, `erlang`, `elixir`, `haskell`, `clojure`, `fortran`, `perl`,
`scheme`, `vim`

**Fallback**: any unrecognized language → plain text (no syntax highlighting, editing still works)

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
