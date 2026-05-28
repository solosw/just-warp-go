<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine, drawSelection, rectangularSelection } from '@codemirror/view'
import { Compartment, EditorState } from '@codemirror/state'
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
const editableCompartment = new Compartment()

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
      keymap.of(defaultKeymap),
      keymap.of(historyKeymap),
      keymap.of([indentWithTab]),
      saveHandler,
      updateListener,
      syntaxHighlighting(defaultHighlightStyle),
      oneDark,
      editableCompartment.of(EditorView.editable.of(!props.readOnly)),
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
    effects: editableCompartment.reconfigure(EditorView.editable.of(!val)),
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
