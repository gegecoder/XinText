import { defineStore } from 'pinia'
import { FileService } from '../../bindings/XinText/internal/service'
import { usePreferencesStore } from './preferences'
import { i18n } from '../i18n'

export interface Tab {
  id: string
  name: string
  path: string | null
  markdown: string
  dirty: boolean
  type: 'md' | 'browser'
  url?: string
  history?: string[]
  historyIndex?: number
}

let tabSeq = 0
const nextId = () => `tab-${Date.now()}-${++tabSeq}`

/**
 * 路径规范化（仅用于去重比较，不修改存储的 path）：
 * - 统一分隔符为反斜杠
 * - 转小写（Windows 文件系统不区分大小写）
 *
 * 这样同一文件的不同写法（C:\a\b.md、C:/a/b.md、c:\A\B.md）能被识别为同一个。
 */
function normalizePathForCompare(p: string | null | undefined): string {
  if (!p) return ''
  return p.replace(/\//g, '\\').toLowerCase()
}

export const useEditorStore = defineStore('editor', {
  state: () => ({
    tabs: [] as Tab[],
    currentTabId: '' as string
  }),
  getters: {
    currentTab(state): Tab | undefined {
      return state.tabs.find((t) => t.id === state.currentTabId)
    }
  },
  actions: {
    /** Create a new empty tab and make it active. */
    newTab(markdown = '', name = '') {
      // Default name: localized "new document" prefix + timestamp
      if (!name) {
        const now = new Date()
        const pad = (n: number) => String(n).padStart(2, '0')
        const ts = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
        name = `${i18n('sidebar.newDocPrefix')}${ts}.md`
      }
      const tab: Tab = {
        id: nextId(),
        name,
        path: null,
        markdown,
        dirty: markdown.length > 0,
        type: 'md'
      }
      this.tabs.push(tab)
      this.currentTabId = tab.id
      return tab
    },

    /** Open a file by path. If already open, switch to it; otherwise create a tab. */
    async openFile(path: string) {
      // 统一存储为正斜杠，保证 tab 显示风格一致（C:/... 而非 C:\...）
      const normalized = path.replace(/\\/g, '/')
      // 去重比较时再小写 + 反斜杠，避免大小写差异漏匹配
      const cmp = normalizePathForCompare(normalized)
      const existing = this.tabs.find(
        (t) => t.type === 'md' && normalizePathForCompare(t.path) === cmp
      )
      if (existing) {
        this.currentTabId = existing.id
        return existing
      }
      // 读取时用规范化后的路径，Go 端 filepath/os 兼容正斜杠
      const content = await FileService.ReadFile(normalized)
      const tab: Tab = {
        id: nextId(),
        name: normalized.split('/').pop() || path,
        path: normalized,
        markdown: content,
        dirty: false,
        type: 'md'
      }
      this.tabs.push(tab)
      this.currentTabId = tab.id

      const prefs = usePreferencesStore()
      prefs.addRecentFile(normalized)
      return tab
    },

    /**
     * 打开浏览器标签页：以 url 去重，已存在则切换过去；否则新建并切到前台。
     * name 取 hostname，方便在 Tabs 栏展示。
     */
    openBrowserTab(url: string) {
      const normalized = url.trim()
      const existing = this.tabs.find((t) => t.type === 'browser' && t.url === normalized)
      if (existing) {
        this.currentTabId = existing.id
        return existing
      }
      let name = normalized
      // 404 兜底 URL 用友好名称，否则取 hostname
      if (/^XinText:notfound\?/i.test(normalized)) {
        name = '404'
      } else {
        try {
          name = new URL(normalized).hostname || normalized
        } catch {
          // 非标准 URL 时直接用原串
        }
      }
      const tab: Tab = {
        id: nextId(),
        name,
        path: null,
        markdown: '',
        dirty: false,
        type: 'browser',
        url: normalized,
        history: [normalized],
        historyIndex: 0
      }
      this.tabs.push(tab)
      this.currentTabId = tab.id
      return tab
    },

    /**
     * 更新浏览器标签页的 URL（地址栏导航 / 内部跳转）。
     * pushHistory=true 时压入历史栈，前进历史会被截断；false 时仅刷新地址栏。
     */
    updateBrowserUrl(id: string, url: string, pushHistory = true) {
      const tab = this.tabs.find((t) => t.id === id && t.type === 'browser')
      if (!tab) return
      const normalized = url.trim()
      if (pushHistory && tab.url !== normalized) {
        const hist = tab.history ? tab.history.slice(0, (tab.historyIndex ?? -1) + 1) : []
        hist.push(normalized)
        tab.history = hist
        tab.historyIndex = hist.length - 1
      }
      tab.url = normalized
      try {
        tab.name = new URL(normalized).hostname || normalized
      } catch {
        tab.name = normalized
      }
    },

    /**
     * 浏览器后退：移动 historyIndex 并返回目标 URL；不可后退时返回 null。
     */
    browserGoBack(id: string): string | null {
      const tab = this.tabs.find((t) => t.id === id && t.type === 'browser')
      if (!tab) return null
      const idx = tab.historyIndex ?? 0
      if (idx <= 0) return null
      const newIdx = idx - 1
      tab.historyIndex = newIdx
      const target = tab.history?.[newIdx] ?? null
      if (target) {
        tab.url = target
        try {
          tab.name = new URL(target).hostname || target
        } catch {
          tab.name = target
        }
      }
      return target
    },

    /**
     * 浏览器前进：移动 historyIndex 并返回目标 URL；不可前进时返回 null。
     */
    browserGoForward(id: string): string | null {
      const tab = this.tabs.find((t) => t.id === id && t.type === 'browser')
      if (!tab) return null
      const hist = tab.history ?? []
      const idx = tab.historyIndex ?? 0
      if (idx >= hist.length - 1) return null
      const newIdx = idx + 1
      tab.historyIndex = newIdx
      const target = hist[newIdx] ?? null
      if (target) {
        tab.url = target
        try {
          tab.name = new URL(target).hostname || target
        } catch {
          tab.name = target
        }
      }
      return target
    },

    /** Close a tab; falls back to the nearest neighbour. */
    closeTab(id: string) {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs.splice(idx, 1)
      if (this.currentTabId === id) {
        const next = this.tabs[idx] || this.tabs[idx - 1]
        this.currentTabId = next ? next.id : ''
      }
    },

    /** Update open tabs after a file/folder was renamed on disk. */
    renameTabPath(oldPath: string, newPath: string) {
      const oldN = normalizePathForCompare(oldPath)
      // 统一存储为正斜杠，与 openFile 行为一致
      const normalizedNew = newPath.replace(/\\/g, '/')
      for (const tab of this.tabs) {
        if (tab.type !== 'md') continue
        if (normalizePathForCompare(tab.path) === oldN) {
          tab.path = normalizedNew
          tab.name = normalizedNew.split('/').pop() || tab.name
        }
      }
    },

    /**
     * 拖拽移动后同步 tab 路径：
     * - 精确匹配移动的文件本身
     * - 移动目录时，批量重写其下所有已打开文件的路径前缀
     */
    moveTabPaths(oldPath: string, newPath: string, isDir: boolean) {
      // 比较时统一规范化（小写 + 正斜杠），与原行为保持一致：移动后 tab.path 写成正斜杠
      const norm = (p: string) => p.replace(/\\/g, '/').toLowerCase().replace(/\/+$/, '')
      const oldN = norm(oldPath)
      const newN = norm(newPath)
      for (const tab of this.tabs) {
        if (tab.type !== 'md') continue
        if (!tab.path) continue
        const p = norm(tab.path)
        if (p === oldN) {
          tab.path = newN
        } else if (isDir && p.startsWith(oldN + '/')) {
          tab.path = newN + p.slice(oldN.length)
        } else {
          continue
        }
        tab.name = tab.path.split('/').pop() || tab.name
      }
    },

    switchTab(id: string) {
      if (this.tabs.some((t) => t.id === id)) {
        this.currentTabId = id
      }
    },

    /** Called by the editor on every content change. */
    updateMarkdown(id: string, markdown: string) {
      const tab = this.tabs.find((t) => t.id === id)
      if (tab && tab.type === 'md' && tab.markdown !== markdown) {
        tab.markdown = markdown
        tab.dirty = true
      }
    },

    /** Save the given tab. If it has no path, prompt via Save As first. */
    /** 保存 tab；新建文档无路径时弹另存为对话框。返回是否保存成功。 */
    async saveTab(id: string, path?: string): Promise<boolean> {
      const tab = this.tabs.find((t) => t.id === id)
      if (!tab) return false
      // 浏览器标签页没有 markdown 内容，不需要保存
      if (tab.type === 'browser') return false
      let target = path || tab.path
      if (!target) {
        target = await FileService.PickSaveFile(tab.name || i18n('app.untitled'))
        if (!target) return false
      }
      await FileService.WriteFile(target, tab.markdown)
      // 统一存储为正斜杠，与 openFile 行为一致
      const normalizedTarget = target.replace(/\\/g, '/')
      tab.path = normalizedTarget
      tab.name = normalizedTarget.split('/').pop() || tab.name
      tab.dirty = false

      const prefs = usePreferencesStore()
      prefs.addRecentFile(normalizedTarget)
      return true
    }
  }
})
