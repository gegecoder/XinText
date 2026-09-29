<template>
  <aside class="sidebar" :style="{ width: sidebarWidth + 'px' }">
    <header class="sidebar__header">
      <span
        class="sidebar__title"
        :title="root ? root.path.replace(/\\/g, '/') : ''"
      >{{ root ? root.name : i18n('sidebar.selectFolder') }}</span>
      <span class="sidebar__header-actions">
        <button class="sidebar__icon-btn" :title="i18n('sidebar.openFolder')" @click="openFolder">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" />
          </svg>
        </button>
        <button class="sidebar__icon-btn" :disabled="!root" :title="i18n('sidebar.refreshTree')" @click="refreshTree">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-2.64-6.36L21 8" /><path d="M21 3v5h-5" />
          </svg>
        </button>
        <button class="sidebar__icon-btn" :title="i18n('sidebar.history')" @click="toggleHistory">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
          </svg>
        </button>
        <button
          class="sidebar__icon-btn"
          :disabled="!root"
          :title="root ? i18n('sidebar.searchFolder') : i18n('sidebar.selectFolderFirst')"
          @click="searchVisible = true"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
          </svg>
        </button>
      </span>
    </header>

    <div v-if="!root" class="sidebar__empty">
      <p>{{ i18n('sidebar.empty') }}</p>
      <button class="sidebar__open" @click="openFolder">{{ i18n('sidebar.openFolderBtn') }}</button>
    </div>

    <template v-else>
      <!-- 搜索 -->
      <div class="sidebar__search">
        <svg class="sidebar__search-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
        </svg>
        <input v-model="searchQuery" class="sidebar__search-input" type="text" :placeholder="i18n('sidebar.searchFiles')" />
      </div>

      <!-- 文件树 -->
      <div class="sidebar__tree">
        <FileTreeNode
          v-if="filteredRoot"
          :node="filteredRoot"
          :depth="0"
          :current-path="currentPath"
          :drag-enabled="!searchQuery.trim()"
          :rename-request="renameRequest"
          @open-file="(p: string) => emit('open-file', p)"
          @toggle="handleToggle"
          @new-file="handleNewFile"
          @new-dir="handleNewDir"
          @delete-file="handleDeleteFile"
          @rename="handleRename"
          @contextmenu="openNodeMenu"
          @move="handleMove"
          @reorder="refreshTree"
          @drag-expand="handleDragExpand"
        />
      </div>

      <!-- 底部按钮区（仅选择目录后显示） -->
      <div class="sidebar__footer">
        <button class="sidebar__footer-btn" @click="recycleVisible = true">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 6h18" />
            <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
            <path d="M10 11v6M14 11v6" />
          </svg>
          <span>{{ i18n('sidebar.recycleBin') }}</span>
        </button>
      </div>
    </template>

    <ConfirmDialog v-model:visible="confirmVisible" :message="confirmMessage" @confirm="doDelete" />
    <!-- 复制/移动到目录浮层 -->
    <CopyOrMoveToDialog
      v-model:visible="transferDialogVisible"
      :root-path="root?.path ?? ''"
      :mode="transferMode"
      @confirm="handleTransferConfirm"
    />
    <!-- 回收站浮层 -->
    <RecycleDialog
      v-model:visible="recycleVisible"
      :root-path="root?.path ?? ''"
      @restored="refreshTree"
    />
    <!-- 全文检索浮层 -->
    <SearchDialog
      v-model:visible="searchVisible"
      :root-path="root?.path ?? null"
      @open-file="(p: string) => emit('open-file', p)"
    />
    <!-- 历史路径浮层 -->
    <Teleport to="body">
      <div v-if="historyVisible" class="hist-overlay" @click="historyVisible = false">
        <div class="hist-panel" :style="historyPanelStyle" @click.stop>
          <div class="hist-panel__header">
            <input
              ref="historyInputRef"
              v-model="historyKeyword"
              class="hist-panel__search"
              :placeholder="i18n('sidebar.searchPaths')"
              @keydown.escape="historyVisible = false"
            />
            <button
              v-if="recentFolders.length > 0"
              class="hist-panel__clear"
              :title="i18n('sidebar.clearHistory')"
              @click="clearHistory"
            >{{ i18n('sidebar.clear') }}</button>
          </div>
          <div class="hist-panel__list">
            <div
              v-for="item in filteredHistory"
              :key="item.path"
              class="hist-item"
              :class="{ 'is-missing': !item.exists }"
            >
              <span class="hist-item__icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" />
                </svg>
              </span>
              <span
                class="hist-item__name"
                :title="item.path.replace(/\\/g, '/')"
                :class="{ 'has-strikethrough': !item.exists }"
                @click="item.exists && openFromHistory(item.path)"
              >{{ folderName(item.path) }}</span>
              <span class="hist-item__path">{{ item.path.replace(/\\/g, '/') }}</span>
              <button
                class="hist-item__action"
                :class="{ 'is-disabled': !item.exists }"
                :title="item.exists ? i18n('sidebar.open') : i18n('sidebar.pathMissing')"
                :disabled="!item.exists"
                @click.stop="item.exists && openFromHistory(item.path)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14M13 6l6 6-6 6"/></svg>
              </button>
              <button class="hist-item__del" :title="i18n('sidebar.removeFromHistory')" @click.stop="removeHistory(item.path)">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/></svg>
              </button>
            </div>
            <div v-if="filteredHistory.length === 0" class="hist-empty">
              {{ recentFolders.length === 0 ? i18n('sidebar.noHistory') : i18n('sidebar.noMatch') }}
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 文件树节点右键菜单 -->
    <Teleport to="body">
      <div v-if="nodeMenuVisible" class="node-menu-overlay" @click="closeNodeMenu" @contextmenu.prevent="closeNodeMenu">
        <div class="node-menu" :style="{ left: nodeMenuX + 'px', top: nodeMenuY + 'px' }" @click.stop>
          <template v-if="nodeMenuTarget?.isDir">
            <div class="node-menu__item" @click="menuNewFile">{{ i18n('sidebar.menuNewFile') }}</div>
            <div class="node-menu__item" @click="menuNewDir">{{ i18n('sidebar.menuNewDir') }}</div>
            <div v-if="!isRootDirTarget" class="node-menu__item" @click="menuRename">{{ i18n('sidebar.menuRename') }}</div>
            <div v-if="!isRootDirTarget" class="node-menu__item node-menu__item--danger" @click="menuDelete">{{ i18n('sidebar.menuDeleteDir') }}</div>
            <div class="node-menu__sep"></div>
            <div class="node-menu__item" @click="menuReveal">{{ i18n('sidebar.menuReveal') }}</div>
          </template>
          <template v-else>
            <div class="node-menu__item" @click="menuOpen">{{ i18n('sidebar.menuOpenFile') }}</div>
            <div class="node-menu__item" @click="menuCopy">{{ i18n('sidebar.menuCopy') }}</div>
            <div class="node-menu__item" @click="menuMove">{{ i18n('sidebar.menuMove') }}</div>
            <div class="node-menu__item" @click="menuRename">{{ i18n('sidebar.menuRename') }}</div>
            <div class="node-menu__item node-menu__item--danger" @click="menuDelete">{{ i18n('sidebar.menuDeleteFile') }}</div>
            <div class="node-menu__sep"></div>
            <div class="node-menu__item" @click="menuReveal">{{ i18n('sidebar.menuReveal') }}</div>
            <div class="node-menu__item" @click="menuProperties">{{ i18n('sidebar.menuProperties') }}</div>
          </template>
        </div>
      </div>
    </Teleport>

    <!-- 右侧可拖拽宽度调整手柄 -->
    <div
      class="sidebar__resizer"
      :class="{ 'is-dragging': resizing }"
      @mousedown="startResize"
    ></div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { FileService, RecycleService } from '../../bindings/XinText/internal/service'
import { useTreeStore, type TreeNode as TreeNodeType, normPath } from '../store/tree'
import { useEditorStore } from '../store/editor'
import { usePreferencesStore } from '../store/preferences'
import FileTreeNode, { type CtxMenuPayload, type RenameRequest } from './FileTreeNode.vue'
import ConfirmDialog from './dialog/ConfirmDialog.vue'
import CopyOrMoveToDialog from './dialog/CopyOrMoveToDialog.vue'
import RecycleDialog from './dialog/RecycleDialog.vue'
import SearchDialog from './dialog/SearchDialog.vue'
import { useNotice } from '../composables/useNotice'
import { i18n } from '../i18n'

const props = defineProps<{
  root: TreeNodeType | null
  currentPath: string | null
}>()

const emit = defineEmits<{
  (e: 'open-file', path: string): void
  (e: 'open-folder', path: string): void
  (e: 'renamed', oldPath: string, newPath: string): void
  (e: 'show-properties', path: string): void
}>()

const treeStore = useTreeStore()
const editorStore = useEditorStore()
const prefsStore = usePreferencesStore()

// —— 侧栏宽度拖拽 ——
const SIDEBAR_WIDTH_KEY = 'XinText.sidebar.width'
const SIDEBAR_MIN = 180
const SIDEBAR_MAX = 400
const sidebarWidth = ref(
  Math.min(SIDEBAR_MAX, Math.max(SIDEBAR_MIN, Number(localStorage.getItem(SIDEBAR_WIDTH_KEY)) || 280))
)
const resizing = ref(false)
let resizeStartX = 0
let resizeStartWidth = 0

function startResize(e: MouseEvent) {
  if (e.button !== 0) return
  resizing.value = true
  resizeStartX = e.clientX
  resizeStartWidth = sidebarWidth.value
  document.addEventListener('mousemove', onResizeMove)
  document.addEventListener('mouseup', stopResize)
  // 拖拽期间禁止文本选中与鼠标指针错乱
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'col-resize'
  e.preventDefault()
}

function onResizeMove(e: MouseEvent) {
  const delta = e.clientX - resizeStartX
  const next = Math.min(SIDEBAR_MAX, Math.max(SIDEBAR_MIN, resizeStartWidth + delta))
  if (next !== sidebarWidth.value) sidebarWidth.value = next
}

function stopResize() {
  resizing.value = false
  document.removeEventListener('mousemove', onResizeMove)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  localStorage.setItem(SIDEBAR_WIDTH_KEY, String(sidebarWidth.value))
}

const searchQuery = ref('')
const confirmVisible = ref(false)
const confirmMessage = ref('')
const pendingDeletePath = ref('')
const searchVisible = ref(false)
// 回收站浮层
const recycleVisible = ref(false)

// 全局提示（NoticeDialog 由 App.vue 单例挂载）
const { showNotice } = useNotice()

// —— 历史路径 ——
const historyVisible = ref(false)
const historyKeyword = ref('')
const historyInputRef = ref<HTMLInputElement | null>(null)
const historyPanelStyle = ref({})
const folderExistsMap = ref<Record<string, boolean>>({})

const recentFolders = computed(() => prefsStore.config.recentFolders || [])

const filteredHistory = computed(() => {
  const kw = historyKeyword.value.toLowerCase().trim()
  return recentFolders.value
    .filter((p) => !kw || p.toLowerCase().includes(kw) || folderName(p).toLowerCase().includes(kw))
    .map((p) => ({ path: p, exists: folderExistsMap.value[p] ?? true }))
})

function folderName(path: string) {
  return path.replace(/\\/g, '/').replace(/\/$/, '').split('/').pop() || path
}

async function toggleHistory(e: MouseEvent) {
  if (historyVisible.value) {
    historyVisible.value = false
    return
  }
  const btn = e.currentTarget as HTMLElement
  const rect = btn.getBoundingClientRect()
  historyPanelStyle.value = { left: rect.left + 'px', top: rect.bottom + 4 + 'px' }
  historyKeyword.value = ''
  historyVisible.value = true
  // 每次打开都重新校验所有路径
  folderExistsMap.value = {}
  for (const p of recentFolders.value) {
    folderExistsMap.value[p] = await prefsStore.pathExists(p)
  }
  nextTick(() => historyInputRef.value?.focus())
}

async function openFromHistory(path: string) {
  historyVisible.value = false
  await treeStore.setRoot(path)
  prefsStore.setProjectPath(path)
  await prefsStore.addRecentFolder(path)
  emit('open-folder', path)
}

async function removeHistory(path: string) {
  await prefsStore.removeRecentFolder(path)
  delete folderExistsMap.value[path]
}

async function clearHistory() {
  await prefsStore.clearRecentFolders()
  folderExistsMap.value = {}
}

// —— 节点右键菜单 ——
const nodeMenuVisible = ref(false)
const nodeMenuX = ref(0)
const nodeMenuY = ref(0)
const nodeMenuTarget = ref<TreeNodeType | null>(null)
// 根目录（当前打开目录，文件树第一层节点）不提供「删除目录」
const isRootDirTarget = computed(() => {
  const t = nodeMenuTarget.value
  const r = props.root
  return !!t?.isDir && !!r && normPath(t.path) === normPath(r.path)
})
// 触发节点行内重命名：nonce 递增作为信号
const renameRequest = ref<RenameRequest | null>(null)
let renameNonce = 0

const NODE_MENU_W = 200
const NODE_MENU_H = 170

function openNodeMenu(payload: CtxMenuPayload) {
  nodeMenuTarget.value = payload.node
  // 防止菜单超出视口
  nodeMenuX.value = Math.min(payload.x, window.innerWidth - NODE_MENU_W - 4)
  nodeMenuY.value = Math.min(payload.y, window.innerHeight - NODE_MENU_H - 4)
  nodeMenuVisible.value = true
}

function closeNodeMenu() {
  nodeMenuVisible.value = false
  nodeMenuTarget.value = null
}

function onNodeMenuKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') closeNodeMenu()
}

onMounted(() => window.addEventListener('keydown', onNodeMenuKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onNodeMenuKeydown))

function menuOpen() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node && !node.isDir) emit('open-file', node.path)
}

// —— 复制/移动到目录 ——
const transferDialogVisible = ref(false)
const transferSrcPath = ref('')
const transferMode = ref<'copy' | 'move'>('copy')

function menuCopy() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node && !node.isDir) {
    transferSrcPath.value = node.path
    transferMode.value = 'copy'
    transferDialogVisible.value = true
  }
}

function menuMove() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node && !node.isDir) {
    transferSrcPath.value = node.path
    transferMode.value = 'move'
    transferDialogVisible.value = true
  }
}

async function handleTransferConfirm(destDirPath: string) {
  const isMove = transferMode.value === 'move'
  try {
    const newPath = isMove
      ? await FileService.MovePath(transferSrcPath.value, destDirPath)
      : await FileService.CopyFile(transferSrcPath.value, destDirPath)
    await refreshTree()
    showNotice(i18n(isMove ? 'sidebar.moveSuccess' : 'sidebar.copySuccess', { path: newPath }))
  } catch (e: any) {
    showNotice(e?.message || i18n(isMove ? 'sidebar.moveFailed' : 'sidebar.copyFailed'))
  }
}

function menuNewFile() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node?.isDir) handleNewFile(node.path)
}

function menuNewDir() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node?.isDir) handleNewDir(node.path)
}

function menuRename() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node) renameRequest.value = { path: node.path, nonce: ++renameNonce }
}

function menuDelete() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node) handleDeleteFile(node.path, node.name)
}

async function menuReveal() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (!node) return
  try {
    await FileService.RevealInExplorer(node.path)
  } catch (e: any) {
    showNotice(e?.message || i18n('app.revealFailed'))
  }
}

function menuProperties() {
  const node = nodeMenuTarget.value
  closeNodeMenu()
  if (node && !node.isDir) emit('show-properties', node.path)
}

async function openFolder() {
  const path = await FileService.PickOpenDirectory()
  if (!path) return
  await treeStore.setRoot(path)
  await prefsStore.addRecentFolder(path)
  emit('open-folder', path)
}

async function refreshTree() {
  await treeStore.reload()
}

// 供父组件调用
defineExpose({ refreshTree })

// ---- 搜索过滤 ----
function filterNode(node: TreeNodeType, query: string): TreeNodeType | null {
  if (!query) return node
  const q = query.toLowerCase()
  if (node.isDir) {
    const children = node.children
      .map((c) => filterNode(c, query))
      .filter(Boolean) as TreeNodeType[]
    if (children.length > 0 || node.name.toLowerCase().includes(q)) {
      return { ...node, children, expanded: true }
    }
    return null
  }
  return node.name.toLowerCase().includes(q) ? node : null
}

const filteredRoot = computed(() => {
  if (!treeStore.root) return null as TreeNodeType | null
  return filterNode(treeStore.root, searchQuery.value) || treeStore.root
})

// ---- 展开/折叠（含懒加载） ----
async function handleToggle(node: TreeNodeType) {
  await treeStore.toggle(node)
}

// ---- 新建文件/目录 ----
function formatTimestamp() {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
}

async function handleNewFile(parentPath: string) {
  const name = `${i18n('sidebar.newDocPrefix')}${formatTimestamp()}.md`
  const fullPath = await FileService.JoinPath(parentPath, name)
  await FileService.WriteFile(fullPath, '')
  await refreshTree()
}

async function handleNewDir(parentPath: string) {
  const name = `${i18n('sidebar.newDirPrefix')}${formatTimestamp()}`
  const fullPath = await FileService.JoinPath(parentPath, name)
  await FileService.CreateDirectory(fullPath)
  await refreshTree()
}

async function refreshParent(parentPath: string) {
  if (!treeStore.root) return
  const node = treeStore.findNodeByPath(treeStore.root, parentPath)
  if (node && node.isDir) {
    node.loaded = false
    node.expanded = true
    await treeStore.loadChildren(node)
  }
}

// ---- 删除 ----
function handleDeleteFile(path: string, name: string) {
  pendingDeletePath.value = path
  confirmMessage.value = i18n('sidebar.confirmDelete', { name })
  confirmVisible.value = true
}

async function doDelete() {
  if (!pendingDeletePath.value) return
  try {
    // 删除即移入应用回收站（物理移动 + file_recycle 记录），可在回收站中还原
    await RecycleService.MoveToRecycle(pendingDeletePath.value)
    await refreshTree()
  } catch (e: any) {
    console.error('move to recycle failed', e)
    showNotice(e?.message || String(e))
  }
  pendingDeletePath.value = ''
}

// ---- 重命名（双击节点名，回车保存） ----
async function handleRename(path: string, newName: string) {
  // 目录重命名前校验：其下若有已打开的文件（含子目录），不允许改名，
  // 否则已打开 tab 的路径会全部失效
  const node = treeStore.root ? treeStore.findNodeByPath(treeStore.root, path) : null
  if (node?.isDir) {
    const dirPrefix = normPath(path).replace(/\/+$/, '') + '/'
    const occupied = editorStore.tabs.some((t) => t.path && normPath(t.path).startsWith(dirPrefix))
    if (occupied) {
      showNotice(i18n('sidebar.renameDirOccupied'))
      return
    }
  }
  try {
    const newPath = await FileService.RenamePath(path, newName)
    await refreshTree()
    emit('renamed', path, newPath)
  } catch (e: any) {
    // 重名校验失败、非法字符等错误信息来自后端
    showNotice(e?.message || String(e))
  }
}

// ---- 拖拽移动 ----
// 拖到折叠目录上悬停 700ms：自动展开（懒加载由 treeStore.toggle 处理）
async function handleDragExpand(node: TreeNodeType) {
  if (!node.expanded) await treeStore.toggle(node)
}

async function handleMove(srcPath: string, destDirPath: string) {
  // vuedraggable 已乐观改写树数组；先读取源节点类型（path 尚未变化）
  const srcNode = treeStore.root ? treeStore.findNodeByPath(treeStore.root, srcPath) : null
  const isDir = srcNode?.isDir ?? false

  // 同目录移动 = 无操作，刷新还原 SortableJS 的排序改动
  const srcParent = normPath(srcPath).split('/').slice(0, -1).join('/')
  if (normPath(srcParent) === normPath(destDirPath)) {
    await refreshTree()
    return
  }

  // 与重命名一致的约束：目录下有已打开文件时禁止移动
  if (isDir) {
    const dirPrefix = normPath(srcPath).replace(/\/+$/, '') + '/'
    const occupied = editorStore.tabs.some((t) => t.path && normPath(t.path).startsWith(dirPrefix))
    if (occupied) {
      showNotice(i18n('sidebar.moveDirOccupied'))
      await refreshTree()
      return
    }
  }

  try {
    const newPath = await FileService.MovePath(srcPath, destDirPath)
    // 同步已打开 tab 的路径（单文件精确匹配 / 目录前缀批量改写）
    editorStore.moveTabPaths(srcPath, newPath, isDir)
    await refreshTree()
  } catch (e: any) {
    showNotice(e?.message || String(e))
    // 移动失败：刷新树，还原 SortableJS 的乐观改动
    await refreshTree()
  }
}
</script>

<style scoped>
.sidebar {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--app-bar-bg);
  flex-shrink: 0;
}

.sidebar__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 37px;
  padding: 0 10px;
  border-bottom: 1px solid var(--app-border);
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
  flex-shrink: 0;
}

.sidebar__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.sidebar__header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.sidebar__icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  color: var(--app-text-muted);
  transition: background 0.12s, color 0.12s;
}

.sidebar__icon-btn:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

.sidebar__icon-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.sidebar__icon-btn:disabled:hover {
  background: none;
  color: var(--app-text-muted);
}

.sidebar__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex: 1;
  color: var(--app-text-muted);
  font-size: 13px;
}

.sidebar__open {
  padding: 6px 14px;
  font-size: 12px;
  color: var(--app-active-text);
  background: var(--app-hover);
  border: 1px solid var(--app-border);
  border-radius: 4px;
  cursor: pointer;
}

.sidebar__search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.sidebar__search-icon {
  color: var(--app-text-muted);
  flex-shrink: 0;
}

.sidebar__search-input {
  flex: 1;
  height: 23px;
  padding: 0 4px;
  font-size: 12px;
  color: var(--app-text);
  background: transparent;
  border: none;
  outline: none;
}

.sidebar__search-input::placeholder {
  color: var(--app-text-muted);
}

.sidebar__tree {
  flex: 1;
  overflow-y: auto;
  padding: 2px 0;
}

/* 底部按钮区 */
.sidebar__footer {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding: 6px 10px;
  border-top: 1px solid var(--app-border);
}

.sidebar__footer-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 28px;
  padding: 0 8px;
  font-size: 12px;
  color: var(--app-text-muted);
  background: none;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.sidebar__footer-btn:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

/* 右侧宽度拖拽手柄：1px 可见线 + 6px 透明命中区 */
.sidebar__resizer {
  position: absolute;
  top: 0;
  right: 0;
  width: 6px;
  height: 100%;
  cursor: col-resize;
  background: transparent;
  z-index: 1;
}

.sidebar__resizer::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 1px;
  height: 100%;
  background: var(--app-border);
  transition: background 0.15s;
}

.sidebar__resizer:hover::before,
.sidebar__resizer.is-dragging::before {
  background: var(--app-active-text, #4285f4);
}
</style>

<style>
/* 文件树节点右键菜单（Teleport 到 body） */
.node-menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 10001;
}

.node-menu {
  position: fixed;
  min-width: 200px;
  padding: 4px 0;
  background: var(--app-bg, #fff);
  border: 1px solid var(--app-border, #e0e0e0);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  font-size: 13px;
}

.node-menu__item {
  padding: 6px 16px;
  cursor: pointer;
  color: var(--app-text, #333);
  white-space: nowrap;
}

.node-menu__item:hover {
  background: var(--app-hover, #f0f0f0);
  color: var(--app-active-text, #4285f4);
}

.node-menu__item--danger:hover {
  background: #fef0f0;
  color: #f56c6c;
}

.node-menu__sep {
  height: 1px;
  margin: 4px 0;
  background: var(--app-border, #e0e0e0);
}

/* 历史路径浮层（Teleport 到 body） */
.hist-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
}

.hist-panel {
  position: fixed;
  width: 340px;
  max-height: 400px;
  background: var(--app-bg, #fff);
  border: 1px solid var(--app-border, #e0e0e0);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.hist-panel__header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--app-border, #e0e0e0);
}

.hist-panel__search {
  flex: 1;
  height: 24px;
  border: none;
  font-size: 13px;
  background: transparent;
  color: var(--app-text, #333);
  outline: none;
}

.hist-panel__clear {
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
}
.hist-panel__clear:hover {
  background: var(--app-hover, #f0f0f0);
  color: var(--app-active-text, #4285f4);
}

.hist-panel__list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  max-height: 350px;
}

.hist-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 13px;
}

.hist-item__icon {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
}

.hist-item__name {
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  color: var(--app-text, #333);
}
.hist-item:not(.is-missing) .hist-item__name:hover {
  color: var(--app-active-text, #4285f4);
}

.hist-item__path {
  font-size: 11px;
  opacity: 0.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.hist-item__action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  border-radius: 4px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.12s;
}
.hist-item:hover .hist-item__action {
  opacity: 1;
}
.hist-item__action:hover:not(.is-disabled) {
  background: var(--app-hover, #f0f0f0);
  color: var(--app-active-text, #4285f4);
}
.hist-item__action.is-disabled {
  opacity: 0.25 !important;
  cursor: not-allowed;
}

.hist-item__del {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  border-radius: 4px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.12s;
}
.hist-item:hover .hist-item__del {
  opacity: 1;
}
.hist-item__del:hover {
  background: #fef0f0;
  color: #f56c6c;
}

.hist-item.is-missing .hist-item__name {
  text-decoration: line-through;
  cursor: not-allowed;
}
.hist-item.is-missing .hist-item__icon {
  opacity: 0.4;
}

.hist-empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  opacity: 0.5;
}
</style>
