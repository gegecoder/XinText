import { defineStore } from 'pinia'
import { FileService, WatchService } from '../../bindings/XinText/internal/service'
import type { DirEntry } from '../../bindings/XinText/internal/service/models'

export interface TreeNode {
  name: string
  path: string
  isDir: boolean
  expanded: boolean
  loaded: boolean
  children: TreeNode[]
}

interface FileEventPayload {
  path: string
  name: string
  isDir: boolean
  op: 'created' | 'changed' | 'deleted' | 'renamed'
}

function isMarkdown(name: string) {
  return /\.(md|markdown|mdown|mkd|txt)$/i.test(name)
}

function sortNodes(nodes: TreeNode[]) {
  nodes.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
}

/** Normalize a path for comparison (Windows backslash -> slash). */
export function normPath(p: string) {
  return p.replace(/\\/g, '/')
}

function toNode(e: DirEntry): TreeNode {
  return {
    name: e.name,
    path: e.path,
    isDir: e.isDir,
    expanded: false,
    loaded: false,
    children: []
  }
}

export const useTreeStore = defineStore('tree', {
  state: () => ({
    rootPath: '' as string,
    root: null as TreeNode | null,
    loading: false
  }),
  actions: {
    /** Reload the entire tree, preserving expanded directory states. */
    async reload() {
      if (!this.rootPath || !this.root) return
      // Collect expanded directory paths before reload
      const expandedPaths: string[] = []
      const collect = (node: TreeNode) => {
        if (node.isDir && node.expanded) expandedPaths.push(node.path)
        node.children.forEach(collect)
      }
      collect(this.root)
      // Rebuild root
      this.root = {
        name: this.root.name,
        path: this.rootPath,
        isDir: true,
        expanded: true,
        loaded: false,
        children: []
      }
      await this.loadChildren(this.root)
      // Re-expand previously expanded directories
      for (const p of expandedPaths) {
        if (normPath(p) === normPath(this.rootPath)) continue
        const node = this.findNodeByPath(this.root, p)
        if (node && node.isDir) {
          node.expanded = true
          node.loaded = false
          await this.loadChildren(node)
        }
      }
    },

    /** Set the project root and load its top-level entries. */
    async setRoot(path: string) {
      if (!path) {
        this.rootPath = ''
        this.root = null
        return
      }
      this.rootPath = path
      this.root = {
        name: path.replace(/\\/g, '/').split('/').pop() || path,
        path,
        isDir: true,
        expanded: true,
        loaded: false,
        children: []
      }
      await this.loadChildren(this.root)
      // Start the filesystem watcher for the opened project.
      try {
        await WatchService.Watch(path)
      } catch (e) {
        console.warn('failed to start watcher', e)
      }
    },

    /** Lazily load children of a directory node. */
    async loadChildren(node: TreeNode) {
      if (!node.isDir || node.loaded) return
      this.loading = true
      try {
        const entries = await FileService.ReadDirectory(node.path)
        node.children = (entries || [])
          .filter((e) => e.isDir || isMarkdown(e.name))
          .map(toNode)
        sortNodes(node.children)
        node.loaded = true
      } finally {
        this.loading = false
      }
    },

    /** Toggle a directory node's expanded state. */
    async toggle(node: TreeNode) {
      if (!node.isDir) return
      node.expanded = !node.expanded
      if (node.expanded && !node.loaded) {
        await this.loadChildren(node)
      }
    },

    /** Locate the parent node of `childPath` inside `parent`. */
    findParent(parent: TreeNode, childPath: string): TreeNode | null {
      const target = normPath(childPath)
      for (const child of parent.children) {
        if (normPath(child.path) === target) return parent
        if (child.isDir) {
          const found = this.findParent(child, childPath)
          if (found) return found
        }
      }
      return null
    },

    /** Apply a filesystem event emitted by the Go watcher. */
    async handleEvent(payload: FileEventPayload) {
      if (!this.root) return
      const parentPath = payload.path.replace(/\\/g, '/').split('/').slice(0, -1).join('/')
      const parent = this.findNodeByPath(this.root, parentPath)
      // 父目录不在树中（深层未展开）→ 跳过，展开时会从磁盘重新加载
      if (!parent) return

      if (payload.op === 'created') {
        // 与 loadChildren 的过滤一致：只添加目录和 markdown 文件
        if (!payload.isDir && !isMarkdown(payload.name)) return
        if (parent.loaded && !parent.children.some((c) => normPath(c.path) === normPath(payload.path))) {
          parent.children.push({
            name: payload.name,
            path: payload.path,
            isDir: payload.isDir,
            expanded: false,
            loaded: false,
            children: []
          })
          sortNodes(parent.children)
        }
      } else if (payload.op === 'deleted') {
        const idx = parent.children.findIndex((c) => normPath(c.path) === normPath(payload.path))
        if (idx !== -1) parent.children.splice(idx, 1)
      } else if (payload.op === 'renamed') {
        // Rename handling is best-effort: refresh the parent.
        if (parent.loaded) {
          parent.loaded = false
          await this.loadChildren(parent)
        }
      }
      // 'changed' (write) does not affect tree structure.
    },

    findNodeByPath(node: TreeNode, path: string): TreeNode | null {
      if (normPath(node.path) === normPath(path)) return node
      for (const child of node.children) {
        const found = this.findNodeByPath(child, path)
        if (found) return found
      }
      return null
    }
  }
})
