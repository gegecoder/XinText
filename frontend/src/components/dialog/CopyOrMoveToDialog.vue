<template>
  <Teleport to="body">
    <div v-if="visible" class="copy-move-overlay" @click.self="cancel">
      <div class="copy-move-dialog">
        <div class="copy-move-title">{{ titleText }}</div>
        <div class="copy-move-tree">
          <template v-if="tree">
            <DirTreeNode
              v-for="node in tree"
              :key="node.path"
              :node="node"
              :depth="0"
              :selected-path="selectedPath"
              @toggle="toggleNode"
              @select="selectNode"
              @new-dir="handleNewDir"
              @refresh="initTree"
            />
          </template>
          <div v-else class="copy-move-empty">{{ i18n('app.loading') }}</div>
        </div>
        <div class="copy-move-dest" :title="selectedPath ? selectedPath.replace(/\\/g, '/') : ''">
          {{ destText }}{{ selectedPath ? selectedPath.replace(/\\/g, '/') : '' }}
        </div>
        <div class="copy-move-actions">
          <button class="copy-move-btn copy-move-btn--cancel" @click="cancel">{{ i18n('confirm.cancel') }}</button>
          <button class="copy-move-btn copy-move-btn--ok" :disabled="!selectedPath" @click="confirm">{{ i18n('confirm.ok') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, defineComponent, ref, watch } from 'vue'
import { FileService } from '../../../bindings/XinText/internal/service'
import { i18n } from '../../i18n'

export interface DirNode {
  name: string
  path: string
  children: DirNode[] | null // null = 未加载
  expanded: boolean
}

// 递归目录树节点（局部组件，通过 name 自引用）。
// 注意：内联组件的 DOM 不继承本文件的 scoped scopeId，
// 其样式放在下方非 scoped 的 style 块中（copy-move-node__ 前缀避免冲突）。
const DirTreeNode = defineComponent({
  name: 'DirTreeNode',
  props: {
    node: { type: Object, required: true },
    depth: { type: Number, required: true },
    selectedPath: { type: String, default: '' },
  },
  emits: ['toggle', 'select', 'new-dir', 'refresh'],
  setup() {
    return { i18n }
  },
  template: `
    <div class="copy-move-node">
      <div
        class="copy-move-node__row"
        :class="{ 'is-selected': node.path === selectedPath }"
        :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
        @click="$emit('select', node)"
      >
        <span
          class="copy-move-node__arrow"
          :class="{ 'is-expanded': node.expanded, 'is-leaf': node.children !== null && node.children.length === 0 }"
          @click.stop="$emit('toggle', node)"
        >▸</span>
        <svg class="copy-move-node__icon" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
          <path d="M1.5 2.5A1.5 1.5 0 0 1 3 1h3l1.5 2H13a1.5 1.5 0 0 1 1.5 1.5v7A1.5 1.5 0 0 1 13 13H3a1.5 1.5 0 0 1-1.5-1.5v-9Z" />
        </svg>
        <span class="copy-move-node__name">{{ node.name }}</span>
        <!-- 仅根目录（第一层）提供刷新：重建整棵树，兜底目录不同步的情况 -->
        <button
          v-if="depth === 0"
          class="copy-move-node__action"
          :title="i18n('sidebar.refreshTree')"
          @click.stop="$emit('refresh')"
        >
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-2.64-6.36" stroke-linecap="round" /><path d="M21 3v6h-6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
        <button
          class="copy-move-node__action"
          :title="i18n('sidebar.menuNewDir')"
          @click.stop="$emit('new-dir', node)"
        >
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" stroke-linecap="round" stroke-linejoin="round" /><path d="M12 10v6M9 13h6" stroke-linecap="round" />
          </svg>
        </button>
      </div>
      <template v-if="node.expanded && node.children">
        <DirTreeNode
          v-for="child in node.children"
          :key="child.path"
          :node="child"
          :depth="depth + 1"
          :selected-path="selectedPath"
          @toggle="$emit('toggle', $event)"
          @select="$emit('select', $event)"
          @new-dir="$emit('new-dir', $event)"
          @refresh="$emit('refresh')"
        />
      </template>
    </div>
  `,
})

const props = withDefaults(
  defineProps<{ visible: boolean; rootPath: string; mode?: 'copy' | 'move' | 'restore' }>(),
  { mode: 'copy' }
)
const emit = defineEmits<{
  'update:visible': [v: boolean]
  confirm: [destDirPath: string]
}>()

// 模式仅影响标题与目标前缀文案，目录树交互完全一致（回收站「还原到」复用本浮层）
const titleText = computed(() => {
  if (props.mode === 'move') return i18n('sidebar.moveToTitle')
  if (props.mode === 'restore') return i18n('sidebar.restoreToTitle')
  return i18n('sidebar.copyToTitle')
})
const destText = computed(() => {
  if (props.mode === 'move') return i18n('sidebar.moveToDest')
  if (props.mode === 'restore') return i18n('sidebar.restoreToDest')
  return i18n('sidebar.copyToDest')
})

const tree = ref<DirNode[] | null>(null)
const selectedPath = ref('')

// 打开时初始化：默认选中根目录，根目录展开显示第一层子目录
watch(
  () => props.visible,
  async (v) => {
    if (v) await initTree()
  }
)

// 构建整棵树（打开浮层时 / 点击根节点刷新图标时调用）
async function initTree() {
  if (!props.rootPath) return
  selectedPath.value = props.rootPath
  const rootName = props.rootPath.replace(/\\/g, '/').replace(/\/$/, '').split('/').pop() || props.rootPath
  const rootNode: DirNode = { name: rootName, path: props.rootPath, children: null, expanded: true }
  tree.value = [rootNode]
  // 注意必须操作 tree.value[0]（响应式代理）：直接改 rootNode 原始对象
  // 不会触发视图更新，表现为子目录加载完成但界面不渲染
  await loadChildren(tree.value[0])
}

async function loadChildren(node: DirNode) {
  try {
    const entries = await FileService.ReadDirectory(node.path)
    const dirs = (entries || [])
      .filter((e) => e.isDir)
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((e) => ({ name: e.name, path: e.path, children: null, expanded: false }) as DirNode)
    node.children = dirs
  } catch {
    node.children = []
  }
}

function formatTimestamp() {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
}

// 浮层内新建目录：与左侧文件栏逻辑一致（时间戳命名），完成后刷新该节点并选中新目录
async function handleNewDir(node: DirNode) {
  try {
    const name = `${i18n('sidebar.newDirPrefix')}${formatTimestamp()}`
    const fullPath = await FileService.JoinPath(node.path, name)
    await FileService.CreateDirectory(fullPath)
    node.children = null
    node.expanded = true
    await loadChildren(node)
    selectedPath.value = fullPath
  } catch {
    /* 创建失败静默：文件树未变化，用户可重试 */
  }
}

async function toggleNode(node: DirNode) {
  node.expanded = !node.expanded
  if (node.expanded && node.children === null) await loadChildren(node)
}

function selectNode(node: DirNode) {
  selectedPath.value = node.path
}

function cancel() {
  emit('update:visible', false)
}

function confirm() {
  if (!selectedPath.value) return
  emit('update:visible', false)
  emit('confirm', selectedPath.value)
}
</script>

<style scoped>
.copy-move-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: copy-move-fade 0.18s ease-out;
}

@keyframes copy-move-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.copy-move-dialog {
  width: 380px;
  max-width: 90vw;
  display: flex;
  flex-direction: column;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: copy-move-pop 0.2s ease-out;
}

@keyframes copy-move-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.copy-move-title {
  padding: 16px 20px 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.copy-move-tree {
  max-height: 300px;
  overflow-y: auto;
  padding: 4px 12px;
  border-top: 1px solid var(--app-border, #dcdfe6);
  border-bottom: 1px solid var(--app-border, #dcdfe6);
}

.copy-move-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted, #909399);
}

.copy-move-dest {
  padding: 10px 20px;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis; /* 头部保留、尾部省略 */
}

.copy-move-actions {
  display: flex;
  gap: 10px;
  padding: 0 20px 16px;
  justify-content: flex-end;
}

.copy-move-btn {
  min-width: 72px;
  height: 32px;
  font-size: 13px;
  border: 1px solid var(--app-border, #dcdfe6);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.copy-move-btn--cancel {
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  color: var(--app-text, #606266);
}

.copy-move-btn--cancel:hover {
  background: var(--app-hover, #f5f7fa);
}

.copy-move-btn--ok {
  background: var(--app-active-text, #409eff);
  color: #fff;
  border-color: var(--app-active-text, #409eff);
}

.copy-move-btn--ok:hover:not(:disabled) {
  opacity: 0.9;
}

.copy-move-btn--ok:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

<!-- 内联递归组件 DirTreeNode 的样式：其 DOM 不继承本组件 scopeId，
     故放在非 scoped 块中；copy-move-node__ 前缀保证不与全局冲突 -->
<style>
.copy-move-node__row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  border-radius: 5px;
  cursor: pointer;
  font-size: 13px;
  color: var(--app-text, #303133);
  user-select: none;
}

.copy-move-node__row:hover {
  background: var(--app-hover, #f5f7fa);
}

.copy-move-node__row.is-selected {
  background: var(--app-active-bg, #e6f0ff);
  color: var(--app-active-text, #409eff);
}

.copy-move-node__arrow {
  width: 14px;
  flex-shrink: 0;
  text-align: center;
  font-size: 10px;
  color: var(--app-text-muted, #909399);
  transition: transform 0.12s;
}

.copy-move-node__arrow.is-expanded {
  transform: rotate(90deg);
}

.copy-move-node__arrow.is-leaf {
  visibility: hidden;
}

.copy-move-node__icon {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
}

.copy-move-node__row.is-selected .copy-move-node__icon {
  color: var(--app-active-text, #409eff);
}

.copy-move-node__name {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.copy-move-node__action {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  margin-right: 4px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.12s, background 0.12s;
}

.copy-move-node__row:hover .copy-move-node__action {
  opacity: 1;
}

.copy-move-node__action:hover {
  background: var(--app-hover, #f0f2f5);
  color: var(--app-active-text, #409eff);
}
</style>
