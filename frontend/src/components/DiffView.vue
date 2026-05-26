<script setup lang="ts">
import { computed } from 'vue'
import hljs from 'highlight.js'

const props = defineProps<{
  oldString: string
  newString: string
  language: string
  filePath: string
}>()

interface DiffLine {
  type: 'add' | 'del' | 'same'
  oldNum: number
  newNum: number
  text: string
}

const diffLines = computed<DiffLine[]>(() => {
  const oldLines = props.oldString.split('\n')
  const newLines = props.newString.split('\n')
  const m = oldLines.length
  const n = newLines.length

  // LCS
  const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0))
  for (let i = 1; i <= m; i++)
    for (let j = 1; j <= n; j++)
      dp[i][j] = oldLines[i - 1] === newLines[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])

  // Backtrack
  const lines: DiffLine[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && oldLines[i - 1] === newLines[j - 1]) {
      lines.unshift({ type: 'same', oldNum: i, newNum: j, text: oldLines[i - 1] })
      i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      lines.unshift({ type: 'add', oldNum: 0, newNum: j, text: newLines[j - 1] })
      j--
    } else {
      lines.unshift({ type: 'del', oldNum: i, newNum: 0, text: oldLines[i - 1] })
      i--
    }
  }
  return lines
})

function highlightLine(text: string): string {
  if (!text) return '&nbsp;'
  const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
  try {
    if (props.language && hljs.getLanguage(props.language)) {
      return hljs.highlight(text, { language: props.language }).value
    }
  } catch {}
  return escaped
}

const renderHtml = computed(() => {
  const parts: string[] = []
  for (const line of diffLines.value) {
    const cls = line.type === 'add' ? 'diff-add' : line.type === 'del' ? 'diff-del' : 'diff-same'
    const oldNum = line.oldNum || ''
    const newNum = line.newNum || ''
    const sign = line.type === 'add' ? '+' : line.type === 'del' ? '-' : ' '
    const code = highlightLine(line.text)
    parts.push(`<tr class="${cls}"><td class="diff-ln">${oldNum}</td><td class="diff-ln">${newNum}</td><td class="diff-sign">${sign}</td><td class="diff-code">${code}</td></tr>`)
  }
  return parts.join('')
})

const addCount = computed(() => diffLines.value.filter(l => l.type === 'add').length)
const delCount = computed(() => diffLines.value.filter(l => l.type === 'del').length)
</script>

<template>
  <div class="diff-view">
    <div class="diff-header">
      <span class="diff-file">{{ filePath }}</span>
      <span class="diff-stat"><span class="dv-add">+{{ addCount }}</span> <span class="dv-del">-{{ delCount }}</span></span>
    </div>
    <div class="diff-table-wrap">
      <table class="diff-table">
        <tbody v-html="renderHtml"></tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.diff-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #0d1117;
  font-family: Consolas, "Courier New", monospace;
  font-size: 12px;
  line-height: 20px;
}
.diff-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  flex-shrink: 0;
}
.diff-file { color: #8b949e; }
.diff-stat { font-size: 11px; }
.dv-add { color: #3fb950; }
.dv-del { color: #f85149; }
.diff-table-wrap {
  flex: 1;
  overflow: auto;
}
.diff-table {
  width: 100%;
  border-collapse: collapse;
}
.diff-table :deep(tr) { height: 20px; }
.diff-table :deep(.diff-same) { background: #0d1117; }
.diff-table :deep(.diff-add) { background: rgba(46,160,67,.15); }
.diff-table :deep(.diff-del) { background: rgba(248,81,73,.15); }
.diff-table :deep(td) {
  padding: 0;
  vertical-align: top;
}
.diff-table :deep(.diff-ln) {
  width: 1%;
  min-width: 48px;
  padding-right: 10px;
  text-align: right;
  color: #484f58;
  user-select: none;
}
.diff-table :deep(.diff-add .diff-ln) { color: #3fb950; }
.diff-table :deep(.diff-del .diff-ln) { color: #f85149; }
.diff-table :deep(.diff-sign) {
  width: 20px;
  text-align: center;
  color: #484f58;
  user-select: none;
}
.diff-table :deep(.diff-add .diff-sign) { color: #3fb950; }
.diff-table :deep(.diff-del .diff-sign) { color: #f85149; }
.diff-table :deep(.diff-code) {
  padding-left: 8px;
  white-space: pre;
  color: #c9d1d9;
}
</style>
