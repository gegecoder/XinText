<template>
  <div class="tabs-bar">
    <div class="tabs-bar__nav">
      <button class="tabs__btn" :title="i18n('tabs.newFile')" @click="emit('new')">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
      </button>
      <div class="tabs-bar__file-btn">
        <button class="tabs__btn" :title="i18n('tabs.openFiles')" @click="toggleFileList">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg>
        </button>
      </div>
      <button class="tabs__btn" :title="i18n('tabs.prevFile')" :disabled="!canPrev" @click="goPrev">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
      </button>
      <button class="tabs__btn" :title="i18n('tabs.nextFile')" :disabled="!canNext" @click="goNext">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18l6-6-6-6"/></svg>
      </button>
    </div>

    <div class="tabs" ref="tabsContainer">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tabs__item"
        :class="{ 'is-active': tab.id === currentTabId, 'is-browser': tab.type === 'browser' }"
        @click="emit('select', tab.id)"
        @contextmenu.prevent="onTabContextMenu($event, tab)"
      >
        <span v-if="tab.type === 'browser'" class="tabs__icon" :title="tab.url">
          <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
            <!-- 网页图标 -->
            <circle cx="12" cy="12" r="10" />
            <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
          </svg>
        </span>
        <span v-else class="tabs__icon tabs__icon--md" :title="tab.path || tab.name">
          <!-- 未保存时在图标前显示圆点 -->
          <span v-if="tab.dirty" class="tabs__icon-dot" :title="i18n('tabs.unsaved')"></span>
          <!-- md 文档图标 -->
          <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
            <path d="M14 2v6h6" />
            <path d="M9 13h6M9 17h6" stroke-width="1.6" />
          </svg>
        </span>
        <span class="tabs__name" :title="tab.type === 'browser' ? (tab.url || tab.name) : (tab.path || tab.name)">{{ tab.name }}</span>
        <span class="tabs__indicator" :title="i18n('tabs.close')" @click.stop="emit('close', tab.id)">
          <svg class="tabs__close" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" shape-rendering="geometricPrecision"><path d="M4 4 L12 12 M12 4 L4 12" /></svg>
        </span>
      </div>
    </div>

    <!-- 当前文件下拉浮层 -->
    <Teleport to="body">
      <div v-if="fileListVisible" class="tabs-overlay" @click="fileListVisible = false">
        <div class="tabs-filelist" :style="fileListStyle" @click.stop>
          <input
            ref="fileListInputRef"
            v-model="fileListKeyword"
            class="tabs-filelist__search"
            :placeholder="i18n('tabs.searchPlaceholder')"
            @keydown.enter="onFileListEnter"
            @keydown.escape="fileListVisible = false"
          />
          <div class="tabs-filelist__list">
            <div
              v-for="tab in filteredTabs"
              :key="tab.id"
              class="tabs-filelist__item"
              :class="{ 'is-active': tab.id === currentTabId }"
              @click="emit('select', tab.id); fileListVisible = false"
            >
              <span v-if="tab.type === 'browser'" class="tabs__icon">
                <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
                </svg>
              </span>
              <span v-else-if="tab.dirty" class="tabs__dot" />
              <span class="tabs-filelist__name">{{ tab.name }}</span>
              <span class="tabs-filelist__path" :title="tab.type === 'browser' ? (tab.url || '') : (tab.path || '')">{{ tab.type === 'browser' ? (tab.url || '') : (tab.path || i18n('tabs.unsaved')) }}</span>
            </div>
            <div v-if="filteredTabs.length === 0" class="tabs-filelist__empty">{{ i18n('tabs.noMatch') }}</div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="ctxMenuVisible"
        class="tabs-ctxmenu"
        :style="{ left: ctxMenuX + 'px', top: ctxMenuY + 'px' }"
        @click.stop
      >
        <div class="tabs-ctxmenu__item" @click="ctxAction('close')">{{ i18n('tabs.ctxClose') }}</div>
        <div class="tabs-ctxmenu__item" @click="ctxAction('close-others')">{{ i18n('tabs.ctxCloseOthers') }}</div>
        <div class="tabs-ctxmenu__sep"></div>
        <div class="tabs-ctxmenu__item" @click="ctxAction('close-right')">{{ i18n('tabs.ctxCloseRight') }}</div>
        <div class="tabs-ctxmenu__item" @click="ctxAction('close-left')">{{ i18n('tabs.ctxCloseLeft') }}</div>
        <div class="tabs-ctxmenu__item" @click="ctxAction('close-all')">{{ i18n('tabs.ctxCloseAll') }}</div>
        <template v-if="!ctxTabIsBrowser">
          <div class="tabs-ctxmenu__sep"></div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('copy-fullpath')">{{ i18n('tabs.ctxCopyFullpath') }}</div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('copy-relpath')">{{ i18n('tabs.ctxCopyRelpath') }}</div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('copy-source')">{{ i18n('tabs.ctxCopySource') }}</div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('copy-txt')">{{ i18n('tabs.ctxCopyTxt') }}</div>
          <div class="tabs-ctxmenu__sep"></div>
          <div class="tabs-ctxmenu__item" :class="{ 'is-disabled': !ctxTab?.path }" @click="ctxTab?.path && ctxAction('reveal')">{{ i18n('tabs.ctxReveal') }}</div>
          <div class="tabs-ctxmenu__item" :class="{ 'is-disabled': !ctxTab?.path }" @click="ctxTab?.path && ctxAction('properties')">{{ i18n('tabs.ctxProperties') }}</div>
          <div class="tabs-ctxmenu__sep"></div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('fullscreen')">
            {{ mode === 'read' ? i18n('tabs.ctxFullscreen') : i18n('tabs.ctxFullscreenEdit') }}
          </div>
        </template>
        <template v-else>
          <div class="tabs-ctxmenu__sep"></div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('copy-url')">复制网址</div>
          <div class="tabs-ctxmenu__item" @click="ctxAction('open-external')">在系统浏览器中打开</div>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { i18n } from '../i18n'
import type { Tab } from '../store/editor'

const props = defineProps<{
  tabs: Tab[]
  currentTabId: string
  // 当前编辑模式：仅阅读模式显示「全屏模式」右键菜单
  mode: 'wysiwyg' | 'source' | 'read'
}>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'close', id: string): void
  (e: 'new'): void
  (e: 'ctx-action', action: string, tab: Tab): void
}>()

// —— 上一个 / 下一个 ——
const currentIndex = computed(() => props.tabs.findIndex((t) => t.id === props.currentTabId))
const canPrev = computed(() => currentIndex.value > 0)
const canNext = computed(() => currentIndex.value >= 0 && currentIndex.value < props.tabs.length - 1)

// 当前 tab 变化时自动滚动到可见位置
const tabsContainer = ref<HTMLElement | null>(null)
watch(
  () => props.currentTabId,
  () => {
    nextTick(() => {
      const el = tabsContainer.value?.querySelector('.tabs__item.is-active') as HTMLElement | null
      if (!el) return
      const c = tabsContainer.value!
      const left = el.offsetLeft
      const right = left + el.offsetWidth
      if (left < c.scrollLeft) {
        c.scrollLeft = left - 10
      } else if (right > c.scrollLeft + c.clientWidth) {
        c.scrollLeft = right - c.clientWidth + 10
      }
    })
  }
)

function goPrev() {
  if (canPrev.value) emit('select', props.tabs[currentIndex.value - 1].id)
}
function goNext() {
  if (canNext.value) emit('select', props.tabs[currentIndex.value + 1].id)
}

// —— 当前文件下拉 ——
const fileListVisible = ref(false)
const fileListKeyword = ref('')
const fileListInputRef = ref<HTMLInputElement | null>(null)
const fileListStyle = ref({})

const filteredTabs = computed(() => {
  const kw = fileListKeyword.value.toLowerCase().trim()
  if (!kw) return props.tabs
  return props.tabs.filter((t) => t.name.toLowerCase().includes(kw))
})

function toggleFileList(e: MouseEvent) {
  if (fileListVisible.value) {
    fileListVisible.value = false
    return
  }
  const btn = e.currentTarget as HTMLElement
  const rect = btn.getBoundingClientRect()
  fileListStyle.value = { left: rect.left + 'px', top: rect.bottom + 4 + 'px' }
  fileListKeyword.value = ''
  fileListVisible.value = true
  nextTick(() => fileListInputRef.value?.focus())
}

function onFileListEnter() {
  if (filteredTabs.value.length > 0) {
    emit('select', filteredTabs.value[0].id)
    fileListVisible.value = false
  }
}

// —— 右键菜单 ——
const ctxMenuVisible = ref(false)
const ctxMenuX = ref(0)
const ctxMenuY = ref(0)
const ctxTab = ref<Tab | null>(null)
const ctxTabIsBrowser = computed(() => ctxTab.value?.type === 'browser')

function onTabContextMenu(e: MouseEvent, tab: Tab) {
  ctxTab.value = tab
  const x = Math.min(e.clientX, window.innerWidth - 220)
  const y = Math.min(e.clientY, window.innerHeight - 400)
  ctxMenuX.value = x
  ctxMenuY.value = y
  ctxMenuVisible.value = true
}

function ctxAction(action: string) {
  ctxMenuVisible.value = false
  if (ctxTab.value) {
    emit('ctx-action', action, ctxTab.value)
  }
}

// 点击外部关闭菜单
if (typeof document !== 'undefined') {
  document.addEventListener('mousedown', (e) => {
    if (ctxMenuVisible.value) {
      const el = e.target as HTMLElement
      if (!el.closest('.tabs-ctxmenu')) ctxMenuVisible.value = false
    }
  })
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      ctxMenuVisible.value = false
      fileListVisible.value = false
    }
  })
}
</script>

<style scoped>
.tabs-bar {
  display: flex;
  align-items: stretch;
  height: 37px;
  background: var(--app-bar-bg);
  /* 用 inset 阴影代替 border-bottom：活动标签的背景色会覆盖阴影，
     形成与下方 toolbar 无缝衔接的效果；非活动标签透明背景让阴影透出 */
  box-shadow: inset 0 -1px 0 var(--app-border);
  flex-shrink: 0;
}

.tabs-bar__nav {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 6px;
  flex-shrink: 0;
  border-right: 1px solid var(--app-border);
}

.tabs-bar__file-btn {
  flex-shrink: 0;
}

.tabs {
  display: flex;
  align-items: flex-start;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
  min-width: 0;
  position: relative;
  scrollbar-width: thin;
  /* 翻转滚动容器：水平滚动条从底部移到顶部；
     flex-start 在翻转后对应视觉底部，使标签贴住底边 */
  transform: scaleY(-1);
}

.tabs::-webkit-scrollbar { height: 3px; }
.tabs::-webkit-scrollbar-track { background: transparent; }
.tabs::-webkit-scrollbar-thumb { background: var(--app-border); border-radius: 2px; }
.tabs::-webkit-scrollbar-thumb:hover { background: var(--app-text-muted); }

.tabs__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  border-radius: 5px;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}
.tabs__btn:hover:not(:disabled) {
  background: var(--app-hover);
  color: var(--app-active-text);
}
.tabs__btn:active:not(:disabled) {
  transform: scale(0.95);
}
.tabs__btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.tabs__item {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 6px 0 12px;
  margin: 0px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px 6px 0 0;
  cursor: pointer;
  font-size: 13px;
  color: var(--app-text-muted);
  flex-shrink: 0;
  max-width: 200px;
  position: relative;
  /* 容器翻转后，每个 tab 翻回来保持内容正常 */
  transform: scaleY(-1);
  transition: background 0.12s, color 0.12s;
}
.tabs__item:hover {
  background: var(--app-hover);
  color: var(--app-text);
}
.tabs__item:hover::after {
  opacity: 0;
}
.tabs__item.is-active {
  background: var(--app-bg);
  color: var(--app-text);
  border-color: var(--app-border);
  border-bottom-color: var(--app-bg);
  overflow: hidden;
}
/* 非活动标签之间的分隔线 */
.tabs__item:not(.is-active)::after {
  content: '';
  position: absolute;
  right: -2px;
  top: 50%;
  margin-top: -7px;
  width: 1px;
  height: 14px;
  background: var(--app-tab-sep);
  opacity: 0.8;
  transition: opacity 0.12s;
  pointer-events: none;
}
/* 活动标签顶部强调线，overflow:hidden 使其跟随父级圆角 */
.tabs__item.is-active::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--app-active-text);
}

.tabs__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: none;
}

/* 未保存圆点，名称前显示 */
.tabs__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--app-text-muted);
  flex-shrink: 0;
  transition: opacity 0.12s;
}
.tabs__item.is-active .tabs__dot {
  background: var(--app-active-text);
}

/* browser tab 的地球图标 / md tab 的文档图标 */
.tabs__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--app-text-muted);
  flex-shrink: 0;
}
.tabs__item.is-active .tabs__icon {
  color: var(--app-active-text);
}

/* md tab 文档图标容器：横向布局，让前置 dirty 圆点与 SVG 并排 */
.tabs__icon--md {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

/* 未保存时图标前侧（左侧）显示的圆点：
   非激活标签灰色弱化，激活标签用主色突出 */
.tabs__icon-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--app-text-muted);
  pointer-events: none;
}
.tabs__item.is-active .tabs__icon-dot {
  background: var(--app-active-text);
}

/* 关闭按钮 */
.tabs__indicator {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  flex-shrink: 0;
  cursor: pointer;
  transition: background 0.15s;
}
.tabs__indicator:hover {
  background: rgba(245, 108, 108, 0.12);
}

.tabs__close {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 12px;
  height: 12px;
  margin: -6px 0 0 -6px;
  opacity: 0;
  color: var(--app-text-muted);
  transition: opacity 0.15s ease 0.05s, color 0.15s;
  pointer-events: none;
}
/* hover 标签时显示关闭按钮，加 50ms 延迟避免快速划过闪烁 */
.tabs__item:hover .tabs__close {
  opacity: 1;
  pointer-events: auto;
}
.tabs__indicator:hover .tabs__close {
  color: #f56c6c;
}
</style>

<style>
/* 全局样式：下拉浮层和右键菜单（Teleport 到 body） */
.tabs-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
}

.tabs-filelist {
  position: fixed;
  width: 320px;
  max-height: 360px;
  background: var(--app-bg, #fff);
  border: 1px solid var(--app-border, #e0e0e0);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tabs-filelist__search {
  padding: 8px 10px;
  border: none;
  border-bottom: 1px solid var(--app-border, #e0e0e0);
  font-size: 13px;
  background: transparent;
  color: var(--app-text, #333);
  outline: none;
}

.tabs-filelist__list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  max-height: 320px;
}

.tabs-filelist__item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
}
.tabs-filelist__item:hover {
  background: var(--app-hover, #f0f0f0);
}
.tabs-filelist__item.is-active {
  color: var(--app-active-text, #4285f4);
}

.tabs-filelist__name {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tabs-filelist__path {
  font-size: 11px;
  opacity: 0.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.tabs-filelist__empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  opacity: 0.5;
}

.tabs-ctxmenu {
  position: fixed;
  z-index: 10001;
  min-width: 200px;
  padding: 4px 0;
  background: var(--app-bg, #fff);
  border: 1px solid var(--app-border, #e0e0e0);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  font-size: 13px;
}

.tabs-ctxmenu__item {
  padding: 6px 16px;
  cursor: pointer;
  color: var(--app-text, #333);
  white-space: nowrap;
}
.tabs-ctxmenu__item:hover {
  background: var(--app-hover, #f0f0f0);
  color: var(--app-active-text, #4285f4);
}

.tabs-ctxmenu__item.is-disabled {
  opacity: 0.35;
  cursor: not-allowed;
  pointer-events: none;
}

.tabs-ctxmenu__sep {
  height: 1px;
  margin: 4px 0;
  background: var(--app-border, #e0e0e0);
}
</style>
