<script setup lang="ts">
import { ref, watch } from 'vue'
import { useWorkspaceStore } from '../stores/workspace'
import { useTerminalStore } from '../stores/terminal'

const ws = useWorkspaceStore()
const ts = useTerminalStore()

interface TreeNode {
  name: string
  path: string
  isDir: boolean
  children: TreeNode[]
}

interface FlatNode {
  node: TreeNode
  depth: number
  padding: number
}

const tree = ref<TreeNode[]>([])
const expanded = ref<Set<string>>(new Set())

function sortChildren(nodes: TreeNode[]) {
  nodes.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
  for (const node of nodes) {
    if (node.isDir) sortChildren(node.children)
  }
}

function buildTree(files: string[]): TreeNode[] {
  const root: TreeNode = { name: '', path: '', isDir: true, children: [] }
  for (const file of files) {
    const parts = file.replace(/\\/g, '/').split('/')
    let current = root
    let currentPath = ''
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      currentPath = currentPath ? currentPath + '/' + part : part
      const isLast = i === parts.length - 1
      let child = current.children.find(c => c.name === part)
      if (!child) {
        child = { name: part, path: currentPath, isDir: !isLast, children: [] }
        current.children.push(child)
      }
      if (!isLast) child.isDir = true
      current = child
    }
  }
  sortChildren(root.children)
  return root.children
}

watch(() => ws.info?.files, (files) => {
  if (files) tree.value = buildTree(files)
  else tree.value = []
}, { immediate: true })

function handleClick(node: TreeNode) {
  if (node.isDir) {
    if (expanded.value.has(node.path)) {
      expanded.value.delete(node.path)
    } else {
      expanded.value.add(node.path)
    }
  } else {
    ts.openFile(node.path)
  }
}

function isExpanded(node: TreeNode): boolean {
  return expanded.value.has(node.path)
}

function getIcon(node: TreeNode): string {
  if (!node.isDir) return '\u{1F4C4}'
  return isExpanded(node) ? '\u{1F4C2}' : '\u{1F4C1}'
}

function renderTree(nodes: TreeNode[], depth: number = 0): FlatNode[] {
  const result: FlatNode[] = []
  for (const node of nodes) {
    result.push({ node, depth, padding: depth * 14 + 8 })
    if (node.isDir && isExpanded(node)) {
      result.push(...renderTree(node.children, depth + 1))
    }
  }
  return result
}
</script>

<template>
  <div class="file-tree-panel">
    <div class="panel-header">文件目录</div>
    <div class="tree-body">
      <div v-if="!ws.hasWorkspace" class="tree-empty">未选择工作区</div>
      <div v-else-if="tree.length === 0" class="tree-empty">无文件</div>
      <div
        v-for="item in renderTree(tree)"
        :key="item.node.path"
        class="tree-node"
        :style="{ paddingLeft: item.padding + 'px' }"
        @click="handleClick(item.node)"
      >
        <span class="node-icon">{{ getIcon(item.node) }}</span>
        <span class="node-name">{{ item.node.name }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-tree-panel {
  width: 220px;
  background: #141416;
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
  letter-spacing: 0.5px;
  border-bottom: 1px solid #2a2a2e;
  height: 32px;
  display: flex;
  align-items: center;
}
.tree-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}
.tree-empty {
  padding: 20px 12px;
  text-align: center;
  color: #555;
  font-size: 12px;
}
.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  cursor: pointer;
  font-size: 12px;
  color: #aaa;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  user-select: none;
}
.tree-node:hover {
  background: #1e1e22;
  color: #ddd;
}
.node-icon {
  flex-shrink: 0;
  font-size: 11px;
}
.node-name {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
