<template>
  <div class="file-tree-node" :class="{ 'is-renaming': renaming }" :data-node-path="node.path" :data-is-dir="node.isDir ? '1' : '0'">
    <!-- 目录 -->
    <template v-if="node.isDir">
      <div
        class="tree-row"
        :class="{ 'is-active': isActive, 'is-dragging': dragging, 'is-drag-over': dragOverValid, 'is-renaming': renaming }"
        :style="{ paddingLeft: depth * 12 + 8 + 'px' }"
        :title="node.name"
        :draggable="dragEnabled && !renaming && depth > 0"
        @click="emit('toggle', node)"
        @contextmenu.prevent="onContextMenu"
        @dragstart="onDragStart"
        @dragend="onDragEnd"
        @dragover="onDirDragOver"
        @dragleave="onDirDragLeave"
        @drop="onDirDrop"
      >
        <svg class="tree-row__caret" :class="{ 'is-expanded': node.expanded }" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="m9 18 6-6-6-6" />
        </svg>
        <svg class="tree-row__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <path v-if="node.expanded" d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7zM12 11l2 2-2 2" stroke-linecap="round" stroke-linejoin="round" />
          <path v-else d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <input
          v-if="renaming"
          ref="renameInput"
          v-model="renameValue"
          class="tree-row__rename"
          @click.stop
          @dblclick.stop
          @keydown.enter.prevent="confirmRename"
          @keydown.esc.prevent="cancelRename"
          @blur="confirmRename"
        />
        <span v-else class="tree-row__name" @dblclick.stop="startRename">{{ node.name }}</span>
        <span class="tree-row__actions">
          <button class="tree-row__action" :title="i18n('sidebar.menuNewFile')" @click.stop="emit('new-file', node.path)">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><path d="M14 2v6h6M12 12v6M9 15h6" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </button>
          <button class="tree-row__action" :title="i18n('sidebar.menuNewDir')" @click.stop="emit('new-dir', node.path)">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" stroke-linecap="round" stroke-linejoin="round" /><path d="M12 10v6M9 13h6" stroke-linecap="round" />
            </svg>
          </button>
        </span>
      </div>
      <!-- 子节点（vuedraggable 拖拽容器，data-parent-path 标记所属目录）。
           filter 排除重命名中的节点：行内 pointer-events:none 后事件穿透到
           .file-tree-node，需在此拦截；preventOnFilter=false 保证 input 正常聚焦 -->
      <draggable
        v-if="node.expanded"
        class="tree-children"
        :class="{ 'drag-disabled': !dragEnabled }"
        :data-parent-path="node.path"
        :list="node.children"
        :disabled="!dragEnabled"
        :move="onSortableMove"
        :animation="150"
        ghost-class="tree-drag-ghost"
        chosen-class="tree-drag-chosen"
        group="XinText-tree"
        filter=".is-renaming"
        :prevent-on-filter="false"
        item-key="path"
        @end="onListEnd"
      >
        <template #item="{ element }">
          <FileTreeNode
            :key="element.path"
            :node="element"
            :depth="depth + 1"
            :current-path="currentPath"
            :drag-enabled="dragEnabled"
            :rename-request="renameRequest"
            @open-file="(p: string) => emit('open-file', p)"
            @toggle="(n: TreeNodeType) => emit('toggle', n)"
            @new-file="(p: string) => emit('new-file', p)"
            @new-dir="(p: string) => emit('new-dir', p)"
            @delete-file="(p: string, n: string) => emit('delete-file', p, n)"
            @rename="(p: string, n: string) => emit('rename', p, n)"
            @contextmenu="(payload: CtxMenuPayload) => emit('contextmenu', payload)"
            @move="(src: string, dest: string) => emit('move', src, dest)"
            @reorder="() => emit('reorder')"
            @drag-expand="(n: TreeNodeType) => emit('drag-expand', n)"
          />
        </template>
      </draggable>
    </template>

    <!-- 文件 -->
    <div
      v-else
      class="tree-row tree-row--file"
      :class="{ 'is-active': isActive, 'is-dragging': dragging, 'is-renaming': renaming }"
      :style="{ paddingLeft: depth * 12 + 8 + 'px' }"
      :title="node.name"
      :draggable="dragEnabled && !renaming"
      @click="emit('open-file', node.path)"
      @contextmenu.prevent="onContextMenu"
      @dragstart="onDragStart"
      @dragend="onDragEnd"
    >
      <span class="tree-row__caret-spacer" />
      <svg class="tree-row__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
        <path d="M14 2v6h6" />
        <path d="M9 13h6M9 17h6" stroke-width="1.6" />
      </svg>
      <input
        v-if="renaming"
        ref="renameInput"
        v-model="renameValue"
        class="tree-row__rename"
        @click.stop
        @dblclick.stop
        @keydown.enter.prevent="confirmRename"
        @keydown.esc.prevent="cancelRename"
        @blur="cancelRename"
      />
      <span v-else class="tree-row__name" @dblclick.stop="startRename">{{ node.name }}</span>
      <span class="tree-row__actions">
        <button class="tree-row__action tree-row__action--danger" :title="i18n('sidebar.menuDeleteFile')" @click.stop="emit('delete-file', node.path, node.name)">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import type { TreeNode } from '../store/tree'

/** 右键菜单请求：父组件要求指定路径的节点进入重命名态 */
export interface RenameRequest {
  path: string
  nonce: number
}

/** 冒泡给父组件的右键菜单信息 */
export interface CtxMenuPayload {
  x: number
  y: number
  node: TreeNode
}
</script>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import draggable from 'vuedraggable'
import type { TreeNode as TreeNodeType } from '../store/tree'
import { i18n } from '../i18n'

interface DragPayload {
  path: string
  isDir: boolean
  name: string
}

const DRAG_MIME = 'application/x-XinText-path'

/**
 * 模块级拖拽源状态。
 * 注意：dragover/dragenter 阶段 dataTransfer.getData() 受规范限制返回空串，
 * 无法用 dataTransfer 在悬停时取拖拽源，因此 dragstart 时同步记录在此。
 */
let currentDrag: DragPayload | null = null

const props = withDefaults(defineProps<{
  node: TreeNodeType
  depth: number
  currentPath: string | null
  dragEnabled?: boolean
  renameRequest?: RenameRequest | null
}>(), {
  dragEnabled: true,
  renameRequest: null
})

const emit = defineEmits<{
  (e: 'open-file', path: string): void
  (e: 'toggle', node: TreeNodeType): void
  (e: 'new-file', path: string): void
  (e: 'new-dir', path: string): void
  (e: 'delete-file', path: string, name: string): void
  (e: 'rename', path: string, newName: string): void
  (e: 'contextmenu', payload: CtxMenuPayload): void
  (e: 'move', srcPath: string, destDirPath: string): void
  (e: 'reorder'): void
  (e: 'drag-expand', node: TreeNodeType): void
}>()

const isActive = computed(() => props.currentPath === props.node.path)

// ---- 双击重命名（行内编辑） ----
const renaming = ref(false)
const renameValue = ref('')
const renameInput = ref<HTMLInputElement | null>(null)

function startRename() {
  renaming.value = true
  renameValue.value = props.node.name
  nextTick(() => {
    renameInput.value?.focus()
    renameInput.value?.select()
  })
}

function confirmRename() {
  if (!renaming.value) return
  const newName = renameValue.value.trim()
  renaming.value = false
  // 空名或未变化时直接取消，不发事件
  if (!newName || newName === props.node.name) return
  emit('rename', props.node.path, newName)
}

function cancelRename() {
  renaming.value = false
}

// 右键菜单触发的重命名：父组件更新 renameRequest（nonce 变化），
// 匹配本节点路径时进入行内编辑态
watch(
  () => props.renameRequest?.nonce,
  (n) => {
    if (n && props.renameRequest && norm(props.renameRequest.path) === norm(props.node.path)) {
      startRename()
    }
  }
)

// ---- 右键菜单 ----
function onContextMenu(e: MouseEvent) {
  if (!props.dragEnabled) return
  emit('contextmenu', { x: e.clientX, y: e.clientY, node: props.node })
}

// ---- 拖拽移动 ----
const dragging = ref(false)
const dragOverValid = ref(false)
let expandTimer: ReturnType<typeof setTimeout> | undefined
let leaveGrace: ReturnType<typeof setTimeout> | undefined
// 目录行"放入文件夹"命中区域：行高的中间 50%
const INTO_BAND = 0.5

// 目录行原生 drop 已处理移动时，抑制随后 Sortable end 的二次处理
let suppressNextEnd = false

function onDragStart(e: DragEvent) {
  if (!props.dragEnabled || renaming.value || props.depth === 0) {
    e.preventDefault()
    return
  }
  const payload: DragPayload = {
    path: props.node.path,
    isDir: props.node.isDir,
    name: props.node.name
  }
  currentDrag = payload
  e.dataTransfer?.setData(DRAG_MIME, JSON.stringify(payload))
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
  dragging.value = true
}

function onDragEnd() {
  dragging.value = false
  dragOverValid.value = false
  clearExpandTimer()
  clearLeaveGrace()
  currentDrag = null
}

function clearLeaveGrace() {
  if (leaveGrace) {
    clearTimeout(leaveGrace)
    leaveGrace = undefined
  }
}

/**
 * 是否为文件树内拖拽。
 * dragover 阶段 dataTransfer 数据不可读、且 WebView2 下自定义 MIME 在
 * types 中不稳定，直接使用 dragstart 时记录的模块级状态即可（同窗口）。
 */
function isInternalDrag(): boolean {
  return currentDrag !== null
}

/** 规范化路径：反斜杠转正斜杠、去除尾部斜杠 */
function norm(p: string) {
  return p.replace(/\\/g, '/').replace(/\/+$/, '')
}

/** 判断把 payload 拖入当前目录节点是否合法 */
function canDropHere(payload: DragPayload): boolean {
  const src = norm(payload.path)
  const dest = norm(props.node.path)
  if (src === dest) return false
  // 目录不能拖入自身或其后代目录
  if (payload.isDir && (dest + '/').startsWith(src + '/')) return false
  return true
}

function clearExpandTimer() {
  if (expandTimer) {
    clearTimeout(expandTimer)
    expandTimer = undefined
  }
}

/**
 * 指针是否位于目录行的"放入文件夹"区域（中间 50%）。
 * 中间 → 放入该文件夹（高亮 + 自动展开，Sortable 不插占位符）
 * 上下边缘 → 作为同级插入（Sortable 插入占位符，节点让位指示位置）
 */
function isIntoBand(e: DragEvent): boolean {
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const y = e.clientY - rect.top
  const margin = rect.height * ((1 - INTO_BAND) / 2)
  return y >= margin && y <= rect.height - margin
}

// 拖到目录行上：按垂直区域分流
function onDirDragOver(e: DragEvent) {
  if (!isInternalDrag() || !canDropHere(currentDrag!)) return
  e.preventDefault()
  clearLeaveGrace()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'

  if (isIntoBand(e)) {
    // 中间区域：放入文件夹
    dragOverValid.value = true
    if (!props.node.expanded && !expandTimer) {
      expandTimer = setTimeout(() => {
        expandTimer = undefined
        emit('drag-expand', props.node)
      }, 500)
    }
  } else {
    // 上下边缘：交给 Sortable 显示插入指示（节点上下让位）
    dragOverValid.value = false
    clearExpandTimer()
  }
}

// dragleave 在行内子元素间也会频繁触发，用 120ms 宽限判定真正离开
// （dragover 持续触发会不断取消该宽限定时器）
function onDirDragLeave() {
  if (!isInternalDrag()) return
  clearLeaveGrace()
  leaveGrace = setTimeout(() => {
    dragOverValid.value = false
    clearExpandTimer()
  }, 120)
}

function onDirDrop(e: DragEvent) {
  if (!isInternalDrag()) return
  const payload = currentDrag
  if (!payload || !canDropHere(payload) || !isIntoBand(e)) return
  e.preventDefault()
  e.stopPropagation()
  dragOverValid.value = false
  clearExpandTimer()
  clearLeaveGrace()
  // 抑制 Sortable 随后的 end 处理，避免二次/错误移动
  suppressNextEnd = true
  emit('move', payload.path, props.node.path)
}

/**
 * SortableJS onMove：悬停在目录行中间区域时返回 false，禁止占位符插入
 * （目录保持不动、等待自动展开）；其余位置返回 true，保留节点让位的
 * 插入位置指示。
 */
function onSortableMove(evt: {
  related: HTMLElement | null
  willInsertAfter: boolean
  originalEvent: MouseEvent
}): boolean {
  const related = evt.related
  if (!related) return true
  // 命中的列表项即 .file-tree-node 根节点
  const isDir = related.dataset.isDir === '1'
  if (!isDir) return true
  const rect = related.getBoundingClientRect()
  const y = evt.originalEvent.clientY - rect.top
  const margin = rect.height * ((1 - INTO_BAND) / 2)
  const intoFolder = y >= margin && y <= rect.height - margin
  return !intoFolder
}

// Sortable 拖拽结束（边缘插入或拖入展开的列表区域）
function onListEnd(evt: { from: HTMLElement; to: HTMLElement; item: HTMLElement }) {
  // 目录行中间区域的 drop 已由原生事件处理
  if (suppressNextEnd) {
    suppressNextEnd = false
    return
  }
  const srcPath = evt.item?.dataset?.nodePath
  const destDir = evt.to?.dataset?.parentPath
  if (!srcPath || !destDir) return
  if (evt.to === evt.from) {
    // 同目录排序：文件系统不维护顺序，刷新还原
    emit('reorder')
    return
  }
  emit('move', srcPath, destDir)
}
</script>

<style scoped>
.tree-row {
  display: flex;
  align-items: center;
  gap: 5px;
  height: 26px;
  padding-right: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--app-text);
  user-select: none;
  transition: background 0.1s;
}

.tree-row:hover {
  background: var(--app-hover);
}

.tree-row.is-active {
  background: var(--app-active-bg);
  color: var(--app-active-text);
  font-weight: 500;
}

.tree-row__caret {
  color: var(--app-text-muted);
  flex-shrink: 0;
  transition: transform 0.15s;
  transform: rotate(0deg);
}

.tree-row__caret.is-expanded {
  transform: rotate(90deg);
}

.tree-row__caret-spacer {
  width: 10px;
  flex-shrink: 0;
}

.tree-row__icon {
  color: var(--app-text-muted);
  flex-shrink: 0;
}

.tree-row.is-active .tree-row__icon {
  color: var(--app-active-text);
}

.tree-row__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.tree-row__rename {
  flex: 1;
  min-width: 0;
  height: 20px;
  padding: 0 4px;
  font-size: 12px;
  font-family: inherit;
  color: var(--app-text);
  background: var(--app-bg);
  border: 1px solid var(--app-active-text);
  border-radius: 3px;
  outline: none;
}

.tree-row__actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.1s;
}

.tree-row:hover .tree-row__actions {
  opacity: 1;
}

.tree-row__action {
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  cursor: pointer;
  padding: 3px;
  border-radius: 3px;
  color: var(--app-text-muted);
  transition: background 0.1s, color 0.1s;
}

.tree-row__action:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

.tree-row__action--danger:hover {
  color: #f56c6c;
}

/* ---- 拖拽视觉反馈 ---- */
.tree-row.is-dragging {
  opacity: 0.4;
}

.tree-row.is-drag-over {
  background: var(--app-active-bg);
  outline: 1px solid var(--app-active-text);
  outline-offset: -1px;
  border-radius: 3px;
}

/* 重命名期间：屏蔽整行的指针事件，避免误触拖动 / 折叠 / 打开；
   input 单独放行以保留输入与失焦保存能力 */
.tree-row.is-renaming {
  pointer-events: none;
}

.tree-row.is-renaming .tree-row__rename {
  pointer-events: auto;
}

.tree-children {
  min-height: 4px;
}

/* 空目录也要有可投放区域 */
.tree-children:empty {
  min-height: 22px;
}

.tree-children.drag-disabled {
  pointer-events: none;
}
</style>

<style>
/* SortableJS 拖拽占位/选中样式（非 scoped，ghost 元素脱离组件作用域） */
.tree-drag-ghost {
  opacity: 0.35;
  background: var(--app-active-bg, rgba(64, 158, 255, 0.12));
  border-radius: 3px;
}

.tree-drag-chosen {
  cursor: grabbing;
}
</style>
