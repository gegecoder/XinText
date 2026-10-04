<template>
  <Teleport to="body">
    <div v-if="visible" class="toc-file-overlay" @click.self="cancel">
      <div class="toc-file-dialog" @keydown.esc="cancel" @keydown.enter="confirm">
        <div class="toc-file-title">{{ i18n('tocFile.title') }}</div>
        <div class="toc-file-search">
          <input
            ref="searchInputRef"
            v-model="filter"
            class="toc-file-search__input"
            type="text"
            :placeholder="i18n('tocFile.searchPlaceholder')"
          />
        </div>
        <div class="toc-file-tree">
          <!-- 无过滤词：目录树（目录可展开，md 文件单选） -->
          <template v-if="!filter.trim()">
            <template v-if="tree">
              <FileTreeNode
                v-for="node in tree"
                :key="node.path"
                :node="node"
                :depth="0"
                :selected-path="selectedPath"
                @toggle="toggleNode"
                @select="selectNode"
              />
              <div v-if="isTreeEmpty" class="toc-file-empty">{{ i18n('tocFile.empty') }}</div>
            </template>
            <div v-else class="toc-file-empty">{{ i18n('app.loading') }}</div>
          </template>
          <!-- 有过滤词：全量加载后按文件名匹配的扁平列表 -->
          <template v-else>
            <div
              v-for="node in filteredFiles"
              :key="node.path"
              class="toc-file-match"
              :class="{ 'is-selected': node.path === selectedPath }"
              @click="selectNode(node)"
            >
              <svg class="toc-file-match__icon" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
                <path d="M3 1.5h6L13 5.5v9H3v-13Zm6 .7V5h2.8L9 2.2Z" />
              </svg>
              <span class="toc-file-match__name">{{ node.name }}</span>
              <span class="toc-file-match__path">{{ relativeToRoot(node.path) }}</span>
            </div>
            <div v-if="!filteredFiles.length" class="toc-file-empty">{{ i18n('tocFile.noMatch') }}</div>
          </template>
        </div>
        <div class="toc-file-dest" :title="selectedPath.replace(/\\/g, '/')">
          {{ selectedPath.replace(/\\/g, '/') }}
        </div>
        <div class="toc-file-path-type">
          <span class="toc-file-path-type__label">{{ i18n('tocFile.pathType') }}</span>
          <label class="toc-file-path-type__option" :class="{ 'is-disabled': !canRelative }">
            <input v-model="pathType" type="radio" value="relative" :disabled="!canRelative" />
            {{ i18n('tocFile.relative') }}
          </label>
          <label class="toc-file-path-type__option">
            <input v-model="pathType" type="radio" value="absolute" />
            {{ i18n('tocFile.absolute') }}
          </label>
        </div>
        <div class="toc-file-actions">
          <button class="toc-file-btn toc-file-btn--cancel" @click="cancel">{{ i18n('confirm.cancel') }}</button>
          <button class="toc-file-btn toc-file-btn--ok" :disabled="!selectedPath" @click="confirm">{{ i18n('confirm.ok') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, defineComponent, nextTick, ref, watch } from 'vue'
import { FileService } from '../../../bindings/XinText/internal/service'
import { i18n } from '../../i18n'

export interface TocFileNode {
  name: string
  path: string
  isDir: boolean
  children: TocFileNode[] | null // null = 未加载
  expanded: boolean
}

// 仅 Markdown 文件可选中插入（与左侧文件栏的 md/markdown/mdown/mkd 过滤一致，去掉 txt）
function isMarkdownFile(name: string) {
  return /\.(md|markdown|mdown|mkd)$/i.test(name)
}

function normPath(p: string) {
  return p.replace(/\\/g, '/')
}

// 递归文件树节点（局部组件，通过 name 自引用）。
// 注意：内联组件的 DOM 不继承本文件的 scoped scopeId，
// 其样式放在下方非 scoped 的 style 块中（toc-file-node__ 前缀避免冲突）。
const FileTreeNode = defineComponent({
  name: 'FileTreeNode',
  props: {
    node: { type: Object, required: true },
    depth: { type: Number, required: true },
    selectedPath: { type: String, default: '' },
  },
  emits: ['toggle', 'select'],
  template: `
    <div class="toc-file-node">
      <div
        class="toc-file-node__row"
        :class="{ 'is-selected': !node.isDir && node.path === selectedPath, 'is-dir': node.isDir }"
        :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
        @click="node.isDir ? $emit('toggle', node) : $emit('select', node)"
      >
        <span
          class="toc-file-node__arrow"
          :class="{ 'is-expanded': node.expanded, 'is-leaf': !node.isDir }"
          @click.stop="node.isDir && $emit('toggle', node)"
        >▸</span>
        <svg v-if="node.isDir" class="toc-file-node__icon" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
          <path d="M1.5 2.5A1.5 1.5 0 0 1 3 1h3l1.5 2H13a1.5 1.5 0 0 1 1.5 1.5v7A1.5 1.5 0 0 1 13 13H3a1.5 1.5 0 0 1-1.5-1.5v-9Z" />
        </svg>
        <svg v-else class="toc-file-node__icon" viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
          <path d="M3 1.5h6L13 5.5v9H3v-13Zm6 .7V5h2.8L9 2.2Z" />
        </svg>
        <span class="toc-file-node__name">{{ node.name }}</span>
      </div>
      <template v-if="node.isDir && node.expanded && node.children">
        <FileTreeNode
          v-for="child in node.children"
          :key="child.path"
          :node="child"
          :depth="depth + 1"
          :selected-path="selectedPath"
          @toggle="$emit('toggle', $event)"
          @select="$emit('select', $event)"
        />
      </template>
    </div>
  `,
})

const props = defineProps<{ visible: boolean; rootPath: string; currentFilePath: string }>()
const emit = defineEmits<{
  'update:visible': [v: boolean]
  /** 确认时携带已组装好的 markdown 链接（[名称](相对/全路径)），由调用方插入光标处 */
  confirm: [markdown: string]
}>()

const tree = ref<TocFileNode[] | null>(null)
const selectedPath = ref('')
const selectedName = ref('')
const filter = ref('')
const pathType = ref<'relative' | 'absolute'>('relative')
const searchInputRef = ref<HTMLInputElement | null>(null)
// 过滤时首次触发全量加载的标记（避免每个按键重复深加载）
let allLoaded = false
let loadingAll = false

/** 当前编辑文件位于根目录内时才能生成相对路径，否则只能全路径 */
const canRelative = computed(() => {
  if (!props.currentFilePath || !props.rootPath) return false
  const cur = normPath(props.currentFilePath).toLowerCase()
  const root = normPath(props.rootPath).replace(/\/+$/, '').toLowerCase()
  return cur.startsWith(root + '/')
})

// 打开时初始化：重置状态，构建树根，聚焦搜索框
watch(
  () => props.visible,
  async (v) => {
    if (!v) return
    filter.value = ''
    selectedPath.value = ''
    selectedName.value = ''
    allLoaded = false
    pathType.value = canRelative.value ? 'relative' : 'absolute'
    await initTree()
    nextTick(() => searchInputRef.value?.focus())
  }
)

// 输入过滤词时确保整棵树已加载（深加载只发生一次）
watch(filter, (v) => {
  if (v.trim()) void ensureAllLoaded()
})

async function initTree() {
  if (!props.rootPath) {
    tree.value = []
    return
  }
  const rootName = normPath(props.rootPath).replace(/\/$/, '').split('/').pop() || props.rootPath
  const rootNode: TocFileNode = { name: rootName, path: props.rootPath, isDir: true, children: null, expanded: true }
  tree.value = [rootNode]
  // 注意必须操作 tree.value[0]（响应式代理）：直接改 rootNode 原始对象
  // 不会触发视图更新，表现为子目录加载完成但界面不渲染
  await loadChildren(tree.value[0])
}

async function loadChildren(node: TocFileNode) {
  try {
    const entries = await FileService.ReadDirectory(node.path)
    node.children = (entries || [])
      .filter((e) => e.isDir || isMarkdownFile(e.name))
      .sort((a, b) => {
        if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
        return a.name.localeCompare(b.name)
      })
      .map((e) => ({ name: e.name, path: e.path, isDir: e.isDir, children: null, expanded: false }) as TocFileNode)
  } catch {
    node.children = []
  }
}

/** 递归加载所有目录（过滤检索需要全量数据） */
async function ensureAllLoaded() {
  if (allLoaded || loadingAll) return
  loadingAll = true
  const walk = async (nodes: TocFileNode[]) => {
    for (const n of nodes) {
      if (!n.isDir) continue
      if (n.children === null) await loadChildren(n)
      await walk(n.children || [])
    }
  }
  if (tree.value) await walk(tree.value)
  allLoaded = true
  loadingAll = false
}

/** 按文件名过滤（不区分大小写）的扁平 md 文件列表 */
const filteredFiles = computed(() => {
  const kw = filter.value.trim().toLowerCase()
  if (!kw || !tree.value) return []
  const result: TocFileNode[] = []
  const walk = (nodes: TocFileNode[]) => {
    for (const n of nodes) {
      if (n.isDir) {
        if (n.children) walk(n.children)
      } else if (n.name.toLowerCase().includes(kw)) {
        result.push(n)
      }
    }
  }
  walk(tree.value)
  return result
})

/** 树为空（根目录下既没有子目录也没有 md 文件） */
const isTreeEmpty = computed(() => {
  const root = tree.value?.[0]
  return !!root && root.children !== null && root.children.length === 0
})

/** 匹配列表中显示的相对根目录路径（去掉文件名） */
function relativeToRoot(p: string) {
  const root = normPath(props.rootPath).replace(/\/+$/, '')
  const rel = normPath(p).replace(new RegExp('^' + root.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '/', 'i'), '')
  const dir = rel.split('/').slice(0, -1).join('/')
  return dir
}

async function toggleNode(node: TocFileNode) {
  if (!node.isDir) return
  node.expanded = !node.expanded
  if (node.expanded && node.children === null) await loadChildren(node)
}

function selectNode(node: TocFileNode) {
  if (node.isDir) return
  selectedPath.value = node.path
  selectedName.value = node.name
}

/** 计算当前编辑文件目录 → 目标文件的相对路径 */
function relativeLink(fromFile: string, toFile: string): string {
  const fromDir = normPath(fromFile).split('/').slice(0, -1)
  const target = normPath(toFile).split('/')
  let i = 0
  // Windows/macOS 路径不区分大小写，公共前缀比较用小写
  while (i < fromDir.length && i < target.length && fromDir[i].toLowerCase() === target[i].toLowerCase()) i++
  const ups = fromDir.length - i
  const rest = target.slice(i).join('/')
  return ups > 0 ? '../'.repeat(ups) + rest : './' + rest
}

function cancel() {
  emit('update:visible', false)
}

function confirm() {
  if (!selectedPath.value) return
  // 链接文本用去掉扩展名的文件名；路径含空格时按 markdown 规范用 <> 包裹
  const linkText = selectedName.value.replace(/\.(md|markdown|mdown|mkd)$/i, '')
  let linkPath =
    pathType.value === 'relative' && canRelative.value
      ? relativeLink(props.currentFilePath, selectedPath.value)
      : normPath(selectedPath.value)
  if (/\s/.test(linkPath)) linkPath = `<${linkPath}>`
  emit('update:visible', false)
  emit('confirm', `[${linkText}](${linkPath})`)
}
</script>

<style scoped>
.toc-file-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: toc-file-fade 0.18s ease-out;
}

@keyframes toc-file-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.toc-file-dialog {
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
  animation: toc-file-pop 0.2s ease-out;
}

@keyframes toc-file-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.toc-file-title {
  padding: 16px 20px 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.toc-file-search {
  padding: 0 20px 10px;
}

.toc-file-search__input {
  width: 100%;
  height: 30px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--app-text, #303133);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border, #dcdfe6);
  border-radius: 6px;
  outline: none;
  box-sizing: border-box;
}

.toc-file-search__input:focus {
  border-color: var(--app-active-text, #409eff);
}

.toc-file-tree {
  max-height: 300px;
  min-height: 120px;
  overflow-y: auto;
  padding: 4px 12px;
  border-top: 1px solid var(--app-border, #dcdfe6);
  border-bottom: 1px solid var(--app-border, #dcdfe6);
}

.toc-file-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted, #909399);
}

.toc-file-match {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 8px;
  border-radius: 5px;
  cursor: pointer;
  font-size: 13px;
  color: var(--app-text, #303133);
  user-select: none;
}

.toc-file-match:hover {
  background: var(--app-hover, #f5f7fa);
}

.toc-file-match.is-selected {
  background: var(--app-active-bg, #e6f0ff);
  color: var(--app-active-text, #409eff);
}

.toc-file-match__icon {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
}

.toc-file-match.is-selected .toc-file-match__icon {
  color: var(--app-active-text, #409eff);
}

.toc-file-match__name {
  flex-shrink: 0;
  max-width: 45%;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.toc-file-match__path {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}

.toc-file-dest {
  padding: 10px 20px 4px;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis; /* 头部保留、尾部省略 */
}

.toc-file-dest:empty::before {
  content: ' ';
}

.toc-file-path-type {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 4px 20px 12px;
  font-size: 12px;
  color: var(--app-text, #606266);
}

.toc-file-path-type__label {
  color: var(--app-text-muted, #909399);
}

.toc-file-path-type__option {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  user-select: none;
}

.toc-file-path-type__option.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toc-file-actions {
  display: flex;
  gap: 10px;
  padding: 0 20px 16px;
  justify-content: flex-end;
}

.toc-file-btn {
  min-width: 72px;
  height: 32px;
  font-size: 13px;
  border: 1px solid var(--app-border, #dcdfe6);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.toc-file-btn--cancel {
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  color: var(--app-text, #606266);
}

.toc-file-btn--cancel:hover {
  background: var(--app-hover, #f5f7fa);
}

.toc-file-btn--ok {
  background: var(--app-active-text, #409eff);
  color: #fff;
  border-color: var(--app-active-text, #409eff);
}

.toc-file-btn--ok:hover:not(:disabled) {
  opacity: 0.9;
}

.toc-file-btn--ok:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

<!-- 内联递归组件 FileTreeNode 的样式：其 DOM 不继承本组件 scopeId，
     故放在非 scoped 块中；toc-file-node__ 前缀保证不与全局冲突 -->
<style>
.toc-file-node__row {
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

.toc-file-node__row:hover {
  background: var(--app-hover, #f5f7fa);
}

.toc-file-node__row.is-selected {
  background: var(--app-active-bg, #e6f0ff);
  color: var(--app-active-text, #409eff);
}

.toc-file-node__arrow {
  width: 14px;
  flex-shrink: 0;
  text-align: center;
  font-size: 10px;
  color: var(--app-text-muted, #909399);
  transition: transform 0.12s;
}

.toc-file-node__arrow.is-expanded {
  transform: rotate(90deg);
}

.toc-file-node__arrow.is-leaf {
  visibility: hidden;
}

.toc-file-node__icon {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
}

.toc-file-node__row.is-selected .toc-file-node__icon {
  color: var(--app-active-text, #409eff);
}

.toc-file-node__name {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
