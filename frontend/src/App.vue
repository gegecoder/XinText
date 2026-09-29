<template>
  <div class="XinText-app" :class="{ 'is-zen': isZen }">
    <BrandBar
      :app-name="appName"
      :app-sub-title="appSubTitle"
      :has-file="!!editorStore.currentTab"
      :pandoc-available="pandocAvailable"
      :is-dark="isDark"
      @new="handleMenuNew"
      @open="handleMenuOpen"
      @save="handleMenuSave"
      @save-as="handleMenuSaveAs"
      @print="handleMenuPrint"
      @export-html="handleExportHTML"
      @export-pdf="handleExportPDF"
      @export-docx="handleExportDOCX"
      @export-txt="handleExportTXT"
      @export-md="handleExportMD"
      @find="handleMenuFind"
      @replace="handleMenuReplace"
      @paragraph="handleMenuParagraph"
      @insert="handleMenuInsert"
      @insert-table="handleMenuInsertTable"
      @format="handleMenuFormat"
      @undo="handleMenuUndo"
      @redo="handleMenuRedo"
      @about="showAbout = true"
      @log="showLog = true"
      @settings="showSettings = true"
      @toggle-theme="toggleTheme"
      :mode="mode"
      :can-undo="editorCanUndo"
      :can-redo="editorCanRedo"
    />
    <div class="XinText-app__body">
      <SideBar
        ref="sideBarRef"
        :root="treeStore.root"
        :current-path="editorStore.currentTab?.path ?? null"
        @open-file="onOpenFile"
        @open-folder="onOpenFolder"
        @renamed="onFileRenamed"
        @show-properties="showFilePropertiesByPath"
      />
      <div class="XinText-app__main">
        <Tabs
          v-if="currentTab"
          :tabs="editorStore.tabs"
          :current-tab-id="editorStore.currentTabId"
          :mode="mode"
          @select="editorStore.switchTab"
          @close="requestCloseTab"
          @new="onNewTab"
          @ctx-action="handleTabCtxAction"
        />
        <Toolbar
          v-if="currentTab && !isBrowserTab"
          :tab-name="currentTab?.name ?? null"
          :tab-path="currentTab?.path ?? null"
          :tab-dirty="currentTab?.dirty ?? false"
          :mode="mode"
          @update:mode="(m) => (mode = m)"
        />
        <main
          class="XinText-app__editor"
          :style="{
            '--zen-zoom': zenZoom,
            '--zen-grow': zenGrow,
            '--zen-top-pad': currentTab && (mode === 'read' || isZen) ? '44px' : '0px',
          }"
          @wheel="onWheel"
        >
          <!-- 左上角热区：阅读模式缩放控件显示 5s 后淡出，鼠标移入唤出、移出重新计时 -->
          <div
            v-if="currentTab && mode === 'read' && !isBrowserTab"
            class="zen-zone"
            @mouseenter="showZenControls"
            @mouseleave="scheduleHideZenControls"
          >
            <div class="zen-zoom" :class="{ 'is-visible': zenBarVisible }" @click="showZenControls">
              <button
                class="zen-zoom__btn"
                :disabled="zenZoomIndex === 0"
                :title="i18n('tabs.zoomOut')"
                @click="zoomOut"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M5 12h14"/></svg>
              </button>
              <button class="zen-zoom__pct" :title="i18n('tabs.zoomReset')" @click="zoomReset">
                {{ zenZoomPercent }}%
              </button>
              <button
                class="zen-zoom__btn"
                :disabled="zenZoomIndex === ZOOM_LEVELS.length - 1"
                :title="i18n('tabs.zoomIn')"
                @click="zoomIn"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
              </button>
              <!-- 独立重置按钮：让"重置为 100%"作为显式动作，与百分比按钮的重置行为并行 -->
              <button
                class="zen-zoom__btn"
                :disabled="zenZoomIndex === ZOOM_DEFAULT_INDEX"
                :title="i18n('tabs.zoomReset')"
                @click="zoomReset"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M3 12a9 9 0 1 0 3-6.7" />
                  <path d="M3 4v5h5" />
                </svg>
              </button>
            </div>
          </div>
          <!-- 全屏编辑：右上角常驻保存按钮，颜色区分保存状态
               （蓝底=有未保存更改，灰色=已保存；Ctrl+S 同样可用；新建文档走另存为） -->
          <div v-if="isZen && mode !== 'read'" class="zen-save">
            <button
              class="zen-save__btn"
              :class="{ 'is-dirty': currentTab?.dirty }"
              :title="i18n('tabs.fullscreenSave')"
              @click="handleMenuSave"
            >
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
                <polyline points="17 21 17 13 7 13 7 21"/>
                <polyline points="7 3 7 8 15 8"/>
              </svg>
            </button>
          </div>
          <Editor
            v-if="currentTab && mode !== 'read' && !isBrowserTab"
            ref="editorCompRef"
            :key="currentTab.id"
            v-model="markdown"
            :mode="mode"
            :doc-path="currentTab.path"
            :theme="isDark ? 'dark' : 'classic'"
            :zen="isZen"
            @open-md="openMdLink"
            @open-browser="openBrowserLink"
            @undo-redo-state="handleUndoRedoState"
          />
          <ReadView
            v-else-if="currentTab && !isBrowserTab"
            ref="readViewRef"
            :key="'read-' + currentTab.id"
            :markdown="markdown"
            :doc-path="currentTab.path"
            :theme="isDark ? 'dark' : 'classic'"
            @open-md="openMdLink"
            @open-browser="openBrowserLink"
          />
          <BrowserView
            v-else-if="currentTab && isBrowserTab"
            :key="'browser-' + currentTab.id"
            :tab-id="currentTab.id"
            :url="currentTab.url ?? ''"
          />
          <div v-else class="XinText-app__welcome">
            <img :src="isDark ? '/logo_dark.png' : '/logo.png'" class="welcome__logo" alt="XinText" />
            <h1 class="welcome__title">{{ appName }}</h1>
            <p class="welcome__desc">{{ appSubTitle }}</p>
            <p class="welcome__hint">{{ i18n('app.welcomeHint') }}</p>
          </div>
        </main>
      </div>
    </div>
    <AboutDialog v-model:visible="showAbout" />
    <UpdateDialog :visible="showUpdateModal" @close="showUpdateModal = false" />
    <SettingsDialog v-model:visible="showSettings" @theme-change="handleThemeSet" @language-change="handleLanguageSet" @saved="refreshPandoc" />
    <LogDialog v-model:visible="showLog" />
    <!-- 全局提示弹窗（单例，替代 window.alert，由 useNotice 驱动） -->
    <NoticeDialog />
    <!-- 文件属性弹窗 -->
    <FilePropertiesDialog v-model:visible="filePropsVisible" :info="filePropsInfo" />
    <!-- 未保存文档关闭确认：保存 / 不保存 / 取消 -->
    <ConfirmDialog
      v-model:visible="closeConfirmVisible"
      mode="save"
      :message="closeConfirmMessage"
      @confirm="onCloseConfirmSave"
      @discard="onCloseConfirmDiscard"
    />
    <ConfirmDialog
      v-model:visible="batchCloseVisible"
      mode="save"
      :message="batchCloseMessage"
      @confirm="onBatchCloseSave"
      @discard="onBatchCloseDiscard"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Events, Browser } from '@wailsio/runtime'
import appConfig from '@app-config'
import BrandBar from './components/BrandBar.vue'
import Toolbar from './components/Toolbar.vue'
import SideBar from './components/SideBar.vue'
import Tabs from './components/Tabs.vue'
import Editor from './components/Editor.vue'
import ReadView from './components/ReadView.vue'
import BrowserView from './components/BrowserView.vue'
import AboutDialog from './components/dialog/AboutDialog.vue'
import UpdateDialog from './components/dialog/UpdateDialog.vue'
import SettingsDialog from './components/dialog/SettingsDialog.vue'
import LogDialog from './components/dialog/LogDialog.vue'
import ConfirmDialog from './components/dialog/ConfirmDialog.vue'
import FilePropertiesDialog from './components/dialog/FilePropertiesDialog.vue'
import NoticeDialog from './components/dialog/NoticeDialog.vue'
import { useUpdate } from './composables/useUpdate'
import { useNotice } from './composables/useNotice'
import { printDocument, renderMarkdownToHTML } from './composables/usePrint'
import { FileService, ExportService } from '../bindings/XinText/internal/service'
import type { FileInfo } from '../bindings/XinText/internal/service/models'
import { useEditorStore, type Tab } from './store/editor'
import { useTreeStore } from './store/tree'
import { usePreferencesStore } from './store/preferences'
import { i18n, setLocale, resolveLocale, type Locale } from './i18n'

const appName = appConfig.name
const appSubTitle = appConfig.branding?.brandSub ?? ''

const editorStore = useEditorStore()
const sideBarRef = ref<{ refreshTree: () => Promise<void> } | null>(null)
const treeStore = useTreeStore()
const prefsStore = usePreferencesStore()
const { checkForUpdate, showUpdateModal } = useUpdate()

const mode = ref<'wysiwyg' | 'source' | 'read'>('read')
// 全屏阅读：隐藏品牌栏/侧栏/tabs/工具栏并请求浏览器原生全屏，Esc 退出
const isZen = ref(false)
// 全屏内容缩放：离散档位（避免浮点累计漂移），默认第 4 档 = 100%
const ZOOM_LEVELS = [0.5, 0.67, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2] as const
const ZOOM_DEFAULT_INDEX = 4
const zenZoomIndex = ref(ZOOM_DEFAULT_INDEX)
const zenZoom = computed(() => ZOOM_LEVELS[zenZoomIndex.value])
// 宽度增长因子：仅 100%~200% 区间从 0 线性增到 1，供内容区 860px→80% 宽度过渡
const zenGrow = computed(() => Math.max(0, zenZoom.value - 1))
const zenZoomPercent = computed(() => Math.round(zenZoom.value * 100))
function zoomIn() {
  zenZoomIndex.value = Math.min(ZOOM_LEVELS.length - 1, zenZoomIndex.value + 1)
}
function zoomOut() {
  zenZoomIndex.value = Math.max(0, zenZoomIndex.value - 1)
}
function zoomReset() {
  zenZoomIndex.value = ZOOM_DEFAULT_INDEX
}
// Ctrl + 鼠标滚轮缩放（仅阅读模式，避免与编辑器内部滚动冲突）
function onWheel(e: WheelEvent) {
  if (mode.value !== 'read' || isBrowserTab.value || !e.ctrlKey) return
  e.preventDefault()
  // deltaY < 0：滚轮向上 → 放大；> 0：向下 → 缩小
  if (e.deltaY < 0) zoomIn()
  else if (e.deltaY > 0) zoomOut()
  // 唤出缩放控件 5s，给用户视觉反馈
  showZenControls()
}

// 缩放控件显隐：阅读模式（含全屏阅读）显示 5s 后淡出，鼠标移入左上角热区唤出
const ZEN_CONTROLS_DELAY = 5_000
const zenControlsVisible = ref(true)
let zenControlsTimer: ReturnType<typeof setTimeout> | null = null
const zenBarVisible = computed(() => zenControlsVisible.value)
function showZenControls() {
  zenControlsVisible.value = true
  if (zenControlsTimer) clearTimeout(zenControlsTimer)
  // 阅读模式（含全屏）统一 5s 自动淡出
  if (mode.value === 'read') {
    zenControlsTimer = setTimeout(() => { zenControlsVisible.value = false }, ZEN_CONTROLS_DELAY)
  } else {
    zenControlsTimer = null
  }
}
function scheduleHideZenControls() {
  if (mode.value !== 'read') return
  if (zenControlsTimer) clearTimeout(zenControlsTimer)
  zenControlsTimer = setTimeout(() => { zenControlsVisible.value = false }, ZEN_CONTROLS_DELAY)
}
const markdown = ref('')
// 自动保存：编辑后 10s 无操作自动写盘（仅对已保存文件生效，新建文档走另存为）
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
const AUTO_SAVE_DELAY = 10_000
const isDark = ref(false)
const showAbout = ref(false)
const showSettings = ref(false)
const showLog = ref(false)
const pandocAvailable = ref(true)

// 文件属性弹窗
const filePropsVisible = ref(false)
const filePropsInfo = ref<FileInfo | null>(null)

// 未保存文档关闭确认状态
const closeConfirmVisible = ref(false)
const closeConfirmMessage = ref('')
const closeTabId = ref('')

/** 关闭 tab：有未保存更改时先弹保存确认 */
function requestCloseTab(id: string) {
  const tab = editorStore.tabs.find((t) => t.id === id)
  if (!tab) return
  if (!tab.dirty) {
    editorStore.closeTab(id)
    return
  }
  closeTabId.value = id
  closeConfirmMessage.value = i18n('app.unsavedChanges', { name: tab.name })
  closeConfirmVisible.value = true
}

/** 确认框选「保存」：保存成功后关闭（新建文档会先走另存为，取消另存则不关闭） */
async function onCloseConfirmSave() {
  closeConfirmVisible.value = false
  const id = closeTabId.value
  const ok = await editorStore.saveTab(id)
  if (ok) {
    editorStore.closeTab(id)
    sideBarRef.value?.refreshTree()
  }
}

/** 确认框选「不保存」：直接关闭，丢弃更改 */
function onCloseConfirmDiscard() {
  closeConfirmVisible.value = false
  editorStore.closeTab(closeTabId.value)
}

// 全局提示（由 App.vue 单例挂载的 NoticeDialog 渲染）
const { showNotice } = useNotice()

// —— 批量关闭未保存确认 ——
const batchCloseIds = ref<string[]>([])
const batchCloseVisible = ref(false)
const batchCloseMessage = ref('')

/** 批量关闭 tabs：非脏的直接关闭，脏的汇总后一次性询问 */
function batchCloseTabs(ids: string[]) {
  const dirty = ids.filter((id) => editorStore.tabs.find((t) => t.id === id)?.dirty)
  const clean = ids.filter((id) => !editorStore.tabs.find((t) => t.id === id)?.dirty)
  // 先关闭不脏的
  for (const id of clean) editorStore.closeTab(id)
  if (dirty.length === 0) return
  const names = dirty.map((id) => editorStore.tabs.find((t) => t.id === id)?.name).filter(Boolean)
  batchCloseIds.value = dirty
  batchCloseMessage.value = i18n('app.batchUnsaved', { count: dirty.length, names: names.join('\n') })
  batchCloseVisible.value = true
}

async function onBatchCloseSave() {
  batchCloseVisible.value = false
  for (const id of batchCloseIds.value) {
    await editorStore.saveTab(id)
    editorStore.closeTab(id)
  }
  sideBarRef.value?.refreshTree()
}

function onBatchCloseDiscard() {
  batchCloseVisible.value = false
  for (const id of batchCloseIds.value) editorStore.closeTab(id)
}

// —— 右键菜单操作 ——
async function handleTabCtxAction(action: string, tab: Tab) {
  switch (action) {
    case 'close':
      requestCloseTab(tab.id)
      break
    case 'close-others': {
      const others = editorStore.tabs.filter((t) => t.id !== tab.id).map((t) => t.id)
      batchCloseTabs(others)
      break
    }
    case 'close-right': {
      const idx = editorStore.tabs.findIndex((t) => t.id === tab.id)
      const right = editorStore.tabs.slice(idx + 1).map((t) => t.id)
      batchCloseTabs(right)
      break
    }
    case 'close-left': {
      const idx = editorStore.tabs.findIndex((t) => t.id === tab.id)
      const left = editorStore.tabs.slice(0, idx).map((t) => t.id)
      batchCloseTabs(left)
      break
    }
    case 'close-all':
      batchCloseTabs(editorStore.tabs.map((t) => t.id))
      break
    case 'copy-fullpath':
      if (tab.path) await navigator.clipboard.writeText(tab.path.replace(/\\/g, '/'))
      else showNotice(i18n('app.noPath'))
      break
    case 'copy-relpath': {
      if (!tab.path) { showNotice(i18n('app.noPath')); break }
      const root = treeStore.rootPath
      if (root) {
        const abs = tab.path.replace(/\\/g, '/')
        const r = root.replace(/\\/g, '/').replace(/\/$/, '')
        const rel = abs.startsWith(r + '/') ? abs.slice(r.length + 1) : abs
        await navigator.clipboard.writeText(rel)
      } else {
        await navigator.clipboard.writeText(tab.path.replace(/\\/g, '/'))
      }
      break
    }
    case 'copy-source':
      await navigator.clipboard.writeText(tab.markdown)
      break
    case 'copy-txt':
      await navigator.clipboard.writeText(stripMarkdown(tab.markdown))
      break
    case 'reveal':
      if (tab.path) {
        try { await FileService.RevealInExplorer(tab.path) }
        catch { showNotice(i18n('app.revealFailed')) }
      } else showNotice(i18n('app.noPath'))
      break
    case 'properties':
      await showFileProperties(tab)
      break
    case 'fullscreen':
      await enterFullscreen(tab)
      break
    case 'copy-url':
      if (tab.url) await navigator.clipboard.writeText(tab.url)
      break
    case 'open-external':
      if (tab.url) {
        try {
          await Browser.OpenURL(tab.url)
        } catch {
          window.open(tab.url, '_blank')
        }
      }
      break
  }
}

// —— 全屏（阅读 / 编辑）——
/** 进入全屏：切到目标 tab，保持当前阅读/编辑模式，隐藏应用 chrome 并请求原生全屏 */
async function enterFullscreen(tab: Tab) {
  if (tab.id !== editorStore.currentTabId) editorStore.switchTab(tab.id)
  isZen.value = true
  try {
    // 原生全屏下 Esc 由浏览器处理，触发 fullscreenchange 后自动恢复 UI
    await document.documentElement.requestFullscreen()
  } catch {
    // 环境不支持原生全屏时退化为窗口内沉浸模式，由自注册的 Esc 监听退出
  }
  // 控件先展示 5s，随后淡出（编辑模式无缩放控件，此调用仅影响阅读模式）
  showZenControls()
  showNotice(mode.value === 'read'
    ? i18n('tabs.fullscreenExitHint')
    : i18n('tabs.fullscreenEditExitHint'))
}

/** 退出全屏阅读（Esc 或当前文档被关闭时调用） */
function exitFullscreen() {
  if (!isZen.value) return
  isZen.value = false
  zoomReset()
  // 退出后仍在阅读模式则恢复 5s 淡出计时；编辑模式无缩放控件
  showZenControls()
  if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
}

// 进入阅读模式或切换阅读文档：显示缩放控件并启动 5s 淡出；离开阅读时清理计时
watch(
  () => [mode.value, editorStore.currentTabId] as const,
  () => {
    if (mode.value === 'read' && editorStore.currentTabId) {
      showZenControls()
    } else if (zenControlsTimer) {
      clearTimeout(zenControlsTimer)
      zenControlsTimer = null
    }
  }
)

/** 简单 Markdown 转纯文本 */
function stripMarkdown(md: string): string {
  return md
    .replace(/```[\s\S]*?```/g, '')   // 代码块
    .replace(/`([^`]+)`/g, '$1')      // 行内代码
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '') // 图片
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // 链接
    .replace(/^#{1,6}\s+/gm, '')       // 标题
    .replace(/[*_~]{1,3}([^*_~]+)[*_~]{1,3}/g, '$1') // 强调
    .replace(/^>\s+/gm, '')            // 引用
    .replace(/^[-*+]\s+/gm, '')        // 无序列表
    .replace(/^\d+\.\s+/gm, '')        // 有序列表
    .replace(/^\|.*\|$/gm, '')         // 表格
    .replace(/^[-*_]{3,}$/gm, '')      // 分割线
    .replace(/\n{3,}/g, '\n\n')        // 多余空行
    .trim()
}

// —— 文件属性弹窗 ——
async function showFileProperties(tab: Tab) {
  if (!tab.path) {
    showNotice(i18n('app.noPath'))
    return
  }
  await showFilePropertiesByPath(tab.path)
}

/** 按路径展示文件属性（SideBar 右键菜单入口） */
async function showFilePropertiesByPath(path: string) {
  if (!path) return
  try {
    filePropsInfo.value = await FileService.GetFileInfo(path)
    filePropsVisible.value = true
  } catch {
    showNotice(i18n('app.fileInfoFailed'))
  }
}


function applyTheme(theme: string) {
  isDark.value = theme === 'dark'
  document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
}

/** 设置浮层中选择主题：立即应用并持久化 */
function handleThemeSet(theme: string) {
  applyTheme(theme)
  prefsStore.setTheme(theme)
}

/** 设置浮层中选择语言：立即切换界面语言并持久化 */
function handleLanguageSet(language: Locale) {
  setLocale(language)
  document.documentElement.lang = language
  prefsStore.setLanguage(language)
}

/** 检测 pandoc 可用性，决定 HTML/DOCX/TXT 导出菜单是否置灰（PDF 已改走 msedge，无需检测引擎） */
async function refreshPandoc() {
  try {
    const info = await ExportService.CheckPandoc()
    pandocAvailable.value = !!info.available
  } catch (e) {
    console.error('check pandoc failed', e)
    pandocAvailable.value = false
  }
}

async function toggleTheme() {
  const next = isDark.value ? 'light' : 'dark'
  applyTheme(next)
  await prefsStore.setTheme(next)
}

const currentTab = computed(() => editorStore.currentTab)
// 浏览器标签页：当前 tab 是 browser 类型时，主区域渲染 BrowserView
const isBrowserTab = computed(() => currentTab.value?.type === 'browser')

/**
 * 链接点击回调：打开本地 .md 文件。
 * 去重比较由 editorStore.openFile 内部完成（路径规范化，避免重复开 tab）。
 */
async function openMdLink(path: string) {
  await editorStore.openFile(path)
  mode.value = 'read'
}

/**
 * 链接点击回调：用新标签页打开 web 链接。
 */
function openBrowserLink(url: string) {
  editorStore.openBrowserTab(url)
}

watch(
  () => editorStore.currentTabId,
  () => {
    markdown.value = currentTab.value?.markdown ?? ''
    // 切换 tab 取消未触发的自动保存计时，切回该 tab 重新编辑才会重新计时
    if (autoSaveTimer) { clearTimeout(autoSaveTimer); autoSaveTimer = null }
    // 全屏阅读时当前文档被关闭（如快捷键），退出全屏回到常规界面
    if (!currentTab.value && isZen.value) exitFullscreen()
  }
)

// 自动保存：编辑已保存文件后 10s 无操作即写盘；编辑中持续重置计时（debounce）。
// 新建文档无 path，不调度，需手动另存为。
function scheduleAutoSave(tabId: string) {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(async () => {
    autoSaveTimer = null
    const tab = editorStore.tabs.find((t) => t.id === tabId)
    // tab 已关闭 / 未变脏 / 无路径 → 跳过
    if (!tab || !tab.dirty || !tab.path) return
    // 捕获保存前内容：保存期间用户若继续编辑，tab.markdown 会变化，
    // saveTab 完成后会把 dirty 置 false，需恢复脏标记并重新计时
    const snapshot = tab.markdown
    await editorStore.saveTab(tabId)
    sideBarRef.value?.refreshTree()
    const latest = editorStore.tabs.find((t) => t.id === tabId)
    if (latest && latest.markdown !== snapshot) {
      latest.dirty = true
      scheduleAutoSave(tabId)
    }
  }, AUTO_SAVE_DELAY)
}

watch(markdown, (value) => {
  const tab = currentTab.value
  if (tab && tab.markdown !== value) {
    editorStore.updateMarkdown(tab.id, value)
    // 仅已保存文件（有 path）且开启自动保存时调度；新建文档需手动另存为
    if (tab.path && (prefsStore.config.autoSave ?? false)) scheduleAutoSave(tab.id)
  }
})

// 设置中关闭自动保存时，取消未触发的计时，立即停止
watch(
  () => prefsStore.config.autoSave,
  (enabled) => {
    if (!enabled && autoSaveTimer) {
      clearTimeout(autoSaveTimer)
      autoSaveTimer = null
    }
  }
)

function onNewTab() {
  const tab = editorStore.newTab()
  markdown.value = tab.markdown
  // 新建空白文档直接进入编辑模式
  mode.value = 'wysiwyg'
}

async function onOpenFile(path: string) {
  await editorStore.openFile(path)
  markdown.value = currentTab.value?.markdown ?? ''
  // 打开已有文档默认进入阅读模式
  mode.value = 'read'
}

function onOpenFolder(path: string) {
  prefsStore.setProjectPath(path)
}

/** 文件树内重命名后，同步更新已打开 tab 的路径与标题 */
function onFileRenamed(oldPath: string, newPath: string) {
  editorStore.renameTabPath(oldPath, newPath)
}

function handleMenuNew() {
  onNewTab()
}

async function handleMenuOpen() {
  const path = await FileService.PickOpenFile()
  if (!path) return
  await onOpenFile(path)
}

async function handleMenuSave() {
  if (!currentTab.value || isBrowserTab.value) return
  await editorStore.saveTab(currentTab.value.id)
  sideBarRef.value?.refreshTree()
}

async function handleMenuSaveAs() {
  if (!currentTab.value || isBrowserTab.value) return
  const path = await FileService.PickSaveFile(currentTab.value.name || i18n('app.untitled'))
  if (!path) return
  await editorStore.saveTab(currentTab.value.id, path)
  sideBarRef.value?.refreshTree()
}

async function doExport(format: 'html' | 'pdf' | 'docx' | 'txt' | 'md') {
  const tab = currentTab.value
  if (!tab || tab.type === 'browser') return
  // PDF 导出已切换为 msedge headless 渲染 Vditor HTML，与 window.print() 路径
  // 共享渲染产物，公式（KaTeX）/代码高亮/表格与打印效果完全一致，不再依赖
  // pandoc 或任何第三方 PDF 引擎
  if (format === 'pdf') {
    return doExportPDF()
  }
  const baseName = tab.name.replace(/\.(md|markdown|mdown|mkd)$/i, '')
  // MD（内嵌图片）默认文件名加后缀，避免直接覆盖源文件
  const defaultName = format === 'md' ? baseName + i18n('app.mdExportSuffix') + '.md' : baseName + '.' + format
  const outPath = await FileService.PickSaveFile(defaultName)
  if (!outPath) return
  const label = format === 'docx' ? 'DOCX' : format === 'txt' ? 'TXT' : format === 'md' ? 'MD' : 'HTML'
  try {
    if (tab.path) {
      await editorStore.saveTab(tab.id)
      await ExportService.ExportFile(tab.path, outPath, format)
    } else if (format === 'html') {
      await ExportService.ExportHTML(tab.markdown, '', outPath)
    } else if (format === 'docx') {
      await ExportService.ExportDOCX(tab.markdown, '', outPath)
    } else if (format === 'txt') {
      await ExportService.ExportTXT(tab.markdown, '', outPath)
    } else if (format === 'md') {
      await ExportService.ExportMD(tab.markdown, '', outPath)
    }
    showNotice(i18n('app.exportSuccess', { label, path: outPath }))
  } catch (e: any) {
    showNotice(i18n('app.exportFailed', { label, reason: e?.message || String(e) }))
  }
}

/**
 * 导出 PDF：复用 Vditor.preview 渲染产物（与 window.print() 共享同一份 HTML 字符串），
 * 后端 ExportPDFFromHTML 用 msedge --headless --print-to-pdf 渲染为 PDF。
 * 公式（KaTeX）、代码高亮、表格样式与打印完全一致。
 */
async function doExportPDF() {
  const tab = currentTab.value
  if (!tab) return
  const baseName = tab.name.replace(/\.(md|markdown|mdown|mkd)$/i, '')
  const outPath = await FileService.PickSaveFile(baseName + '.pdf')
  if (!outPath) return
  try {
    const html = await renderMarkdownToHTML(tab.markdown, tab.path, isDark.value ? 'dark' : 'classic')
    await ExportService.ExportPDFFromHTML(html, outPath)
    showNotice(i18n('app.exportSuccess', { label: 'PDF', path: outPath }))
  } catch (e: any) {
    showNotice(i18n('app.exportFailed', { label: 'PDF', reason: e?.message || String(e) }))
  }
}

function handleExportHTML() {
  return doExport('html')
}

function handleExportPDF() {
  return doExport('pdf')
}

function handleExportDOCX() {
  return doExport('docx')
}

function handleExportTXT() {
  return doExport('txt')
}

function handleExportMD() {
  return doExport('md')
}

/** 模态弹窗打开时，快捷键不抢占（查找/替换等） */
function isModalOpen() {
  return !!document.querySelector(
    '.settings-overlay, .about-overlay, .update-overlay, .confirm-overlay, .notice-overlay, .search-overlay, .fp-overlay, .log-overlay'
  )
}

/** 暴露给 ReadView / Editor 组件实例的方法类型 */
interface FindReplaceCapable {
  openFind: () => void
  openReplace?: () => void
  setHeading?: (level: 0 | 1 | 2 | 3 | 4 | 5 | 6) => void
  insertBlock?: (kind: 'insert-before' | 'insert-after' | 'image' | 'link' | 'quote' | 'table' | 'line' | 'inline-code' | 'code' | 'footnote' | 'formula') => void
  insertTable?: (rows: number, cols: number) => void
  format?: (type: 'bold' | 'italic' | 'strike' | 'list' | 'ordered-list' | 'check' | 'outdent' | 'indent') => void
  undo?: () => void
  redo?: () => void
}

const editorCompRef = ref<FindReplaceCapable | null>(null)
const readViewRef = ref<FindReplaceCapable | null>(null)
// undo/redo 可用状态（从 Editor 的 Vditor 工具栏按钮同步）
const editorCanUndo = ref(true)
const editorCanRedo = ref(true)
function handleUndoRedoState(canUndo: boolean, canRedo: boolean) {
  editorCanUndo.value = canUndo
  editorCanRedo.value = canRedo
}

/** 菜单/快捷键：查找。阅读模式走 ReadView，编辑模式走 Editor */
function handleMenuFind() {
  if (!currentTab.value || isBrowserTab.value || isModalOpen()) return
  if (mode.value === 'read') {
    readViewRef.value?.openFind()
  } else {
    editorCompRef.value?.openFind()
  }
}

/** 菜单/快捷键：替换，仅预览编辑/源码编辑可用 */
function handleMenuReplace() {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.openReplace?.()
}

/** 菜单/快捷键：段落/标题级别（Ctrl+0~6），仅编辑模式可用 */
function handleMenuParagraph(level: 0 | 1 | 2 | 3 | 4 | 5 | 6) {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.setHeading?.(level)
}

/** 菜单：插入块（段落/图片/链接/引用/脚注/表格/分割线/代码/公式），仅编辑模式可用 */
function handleMenuInsert(kind: 'insert-before' | 'insert-after' | 'image' | 'link' | 'quote' | 'table' | 'line' | 'inline-code' | 'code' | 'footnote' | 'formula') {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  // 段落(上方/下方) 在源码编辑模式下不可用（依赖 Vditor IR/WYSIWYG toolbar）
  if ((kind === 'insert-before' || kind === 'insert-after') && mode.value === 'source') return
  editorCompRef.value?.insertBlock?.(kind)
}

/** 菜单：插入 n*n 表格（来自表格菜单的 10x10 网格浮层），仅编辑模式可用 */
function handleMenuInsertTable(rows: number, cols: number) {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.insertTable?.(rows, cols)
}

/** 菜单/快捷键：格式化（加粗/斜体/删除线/列表/缩进），仅编辑模式可用 */
function handleMenuFormat(type: 'bold' | 'italic' | 'strike' | 'list' | 'ordered-list' | 'check' | 'outdent' | 'indent') {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.format?.(type)
}

/** 菜单/快捷键：撤销，仅编辑模式可用 */
function handleMenuUndo() {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.undo?.()
}

/** 菜单/快捷键：重做，仅编辑模式可用 */
function handleMenuRedo() {
  if (!currentTab.value || isBrowserTab.value || isModalOpen() || mode.value === 'read') return
  editorCompRef.value?.redo?.()
}

/** 菜单/快捷键：打印当前文档（Vditor 渲染 + WebView2 打印对话框） */
async function handleMenuPrint() {
  const tab = currentTab.value
  if (!tab || tab.type === 'browser' || isModalOpen()) return
  try {
    await printDocument(tab.markdown, tab.path, isDark.value ? 'dark' : 'classic')
  } catch (e: any) {
    showNotice(i18n('app.printFailed', { reason: e?.message || String(e) }))
  }
}

function onWebviewKeydown(e: KeyboardEvent) {
  const ctrl = e.ctrlKey || e.metaKey
  const key = e.key.toLowerCase()
  // Alt+ 快捷键：插入菜单项（仅编辑模式生效，由 handleMenuInsert/handleMenuInsertTable 自行守卫）
  if (e.altKey && !ctrl && !e.shiftKey) {
    // 用 e.code（物理键）匹配，规避 Alt-code 导致 e.key 变成特殊字符的情况
    const insertMap: Record<string, 'insert-before' | 'insert-after' | 'image' | 'link' | 'quote' | 'footnote' | 'line' | 'inline-code' | 'code' | 'formula'> = {
      'Digit1': 'insert-before',
      'Digit2': 'insert-after',
      'Digit3': 'image',
      'Digit4': 'link',
      'Digit5': 'quote',
      'Digit6': 'footnote',
      'Digit8': 'line',
      'KeyQ': 'inline-code',
      'KeyW': 'code',
      'KeyE': 'formula',
    }
    if (insertMap[e.code]) {
      e.preventDefault()
      handleMenuInsert(insertMap[e.code])
      return
    }
    if (e.code === 'Digit7') {
      // Alt+7：表格，默认 3x3
      e.preventDefault()
      handleMenuInsertTable(3, 3)
      return
    }
  }
  if (!ctrl) return
  // Ctrl+Shift+ 组合
  if (e.shiftKey) {
    if (key === 's') { e.preventDefault(); handleMenuSaveAs(); return }
    if (key === 'n') { e.preventDefault(); handleMenuOpen(); return }
    // Ctrl+Shift+I/O（列表缩进/反向缩进）由 Vditor 原生处理，不在此重复触发
    return
  }
  // Ctrl+ 单键组合
  if (key === 's') {
    e.preventDefault()
    handleMenuSave()
  } else if (key === 'n') {
    e.preventDefault()
    handleMenuNew()
  } else if (key === 'f') {
    e.preventDefault()
    handleMenuFind()
  } else if (key === 'h') {
    e.preventDefault()
    handleMenuReplace()
  } else if (key === 'p') {
    // 阻止 WebView2 默认打印整个主界面，改用自定义文档打印
    e.preventDefault()
    handleMenuPrint()
  } else if (e.code === 'Backquote') {
    // Ctrl+·：段落正文（仅编辑模式生效，函数内自行守卫）
    e.preventDefault()
    handleMenuParagraph(0)
  } else if (/^[1-6]$/.test(e.key)) {
    // Ctrl+1~6：标题级别（仅编辑模式生效，函数内自行守卫）
    e.preventDefault()
    handleMenuParagraph(Number(e.key) as 1 | 2 | 3 | 4 | 5 | 6)
  }
}

function handleFileEvent(payload: any) {
  // Wails v3 事件回调 payload 格式为 {name, data: {path, name, isDir, op}}
  const data = payload?.data || payload
  treeStore.handleEvent(data)
  // 重命名由 SideBar @renamed 主动同步 tab（watcher 的 renamed 事件携带旧路径，
  // 文件已不存在，不能再 openFile）
  if (currentTab.value?.path && currentTab.value.path === data.path && data.op === 'changed') {
    editorStore.openFile(data.path)
  }
}

const unlisten: Array<() => void> = []

onMounted(async () => {
  await prefsStore.load()
  applyTheme(prefsStore.config.theme || 'light')
  // 应用界面语言：用户显式选择优先，否则按系统语言（后端已解析，前端再兜底）
  const lang = resolveLocale(prefsStore.config.language)
  setLocale(lang)
  document.documentElement.lang = lang
  refreshPandoc()

  // 启动后延迟自动检测更新，有新版本才弹窗
  setTimeout(() => { checkForUpdate().catch(() => {}) }, 2000)

  if (prefsStore.config.projectPath) {
    try {
      await treeStore.setRoot(prefsStore.config.projectPath)
    } catch (e) {
      console.warn('failed to restore project', e)
    }
  }

  unlisten.push(await Events.On('menu:new', handleMenuNew))
  unlisten.push(await Events.On('menu:open', handleMenuOpen))
  unlisten.push(await Events.On('menu:save', handleMenuSave))

  window.addEventListener('keydown', onWebviewKeydown)

  const onEscKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') showAbout.value = false
  }
  window.addEventListener('keydown', onEscKeydown)

  // 原生全屏下按 Esc：浏览器退出全屏并触发本事件，恢复应用 chrome
  const onFullscreenChange = () => {
    if (!document.fullscreenElement) isZen.value = false
  }
  document.addEventListener('fullscreenchange', onFullscreenChange)
  // 退化沉浸模式（无原生全屏）下由 Esc 退出；页内查找栏打开时 Esc 先关查找栏
  const onZenEscKeydown = (e: KeyboardEvent) => {
    if (e.key !== 'Escape' || !isZen.value) return
    // 阅读视图查找栏 / 编辑器查找替换栏打开时，Esc 优先关闭它们
    if (document.querySelector('.read-view__findbar, .ot-fr')) return
    e.preventDefault()
    exitFullscreen()
  }
  window.addEventListener('keydown', onZenEscKeydown, true)

  unlisten.push(await Events.On('file:created', handleFileEvent))
  unlisten.push(await Events.On('file:changed', handleFileEvent))
  unlisten.push(await Events.On('file:deleted', handleFileEvent))
  unlisten.push(await Events.On('file:renamed', handleFileEvent))

  let resizeTimer: ReturnType<typeof setTimeout> | null = null
  const onResize = () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      prefsStore.setWindowSize(window.innerWidth, window.innerHeight)
    }, 400)
  }
  window.addEventListener('resize', onResize)
  unlisten.push(() => {
    window.removeEventListener('resize', onResize)
    window.removeEventListener('keydown', onWebviewKeydown)
    window.removeEventListener('keydown', onEscKeydown)
    window.removeEventListener('keydown', onZenEscKeydown, true)
    document.removeEventListener('fullscreenchange', onFullscreenChange)
  })
})

onBeforeUnmount(() => {
  unlisten.forEach((off) => off())
  if (autoSaveTimer) { clearTimeout(autoSaveTimer); autoSaveTimer = null }
  if (zenControlsTimer) clearTimeout(zenControlsTimer)
})
</script>

<style>
* {
  box-sizing: border-box;
}
html,
body,
#app {
  margin: 0;
  height: 100%;
}
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
    'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei',
    'Noto Sans CJK SC', 'WenQuanYi Micro Hei', sans-serif;
  color: var(--app-text);
  /* 不设 background：body 保持透明，Teleport 到 body 的遮罩层 backdrop-filter
     才能在 WebView2 中捕获 #app 内容（.XinText-app 提供应用背景）。
     参考 SuperSender：body/#app 均无 background，毛玻璃遮罩才能生效。 */
}

/* Teleport 到 body 的弹窗层（overlay/dialog）：
   1. 禁用 CSS animation：动画期间合成层处于 animated 状态，backdrop texture 无法稳定捕获。
   2. transform: translateZ(0) 强制提升为 GPU 合成层：
      Chrome 中 backdrop-filter 无需此属性即可生效（已验证），但 WebView2 需要元素被显式
      提升为合成层才能正确捕获背景内容。参考 SuperSender 的 el-overlay（Element Plus 隐式合成）。 */
body > [class$="-overlay"],
body > [class$="-overlay"] > [class$="-dialog"] {
  animation: none !important;
  transform: translateZ(0);
}

/* 全屏编辑：编辑器查找/替换栏下移，避开右上角保存按钮 */
.XinText-app.is-zen .ot-fr {
  top: 54px;
}
</style>

<style scoped>
.XinText-app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--app-bg);
}

/* 全屏阅读：隐藏品牌栏 / 侧栏 / tabs / 工具栏，阅读视图占满整个窗口 */
.XinText-app.is-zen .brand-bar,
.XinText-app.is-zen .sidebar,
.XinText-app.is-zen .tabs-bar,
.XinText-app.is-zen .toolbar {
  display: none;
}

/* 左上角透明热区：比控件略大，保证鼠标移到左上角即可唤出 */
.zen-zone {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 20;
  padding: 10px 16px 12px 12px;
}

/* 全屏缩放控件：悬浮胶囊条，5s 无操作淡出（热区仍可接收鼠标） */
.zen-zoom {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 3px;
  background: var(--app-bar-bg);
  border: 1px solid var(--app-border);
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.18);
  opacity: 0;
  pointer-events: none;
  transform: translateY(-4px);
  transition: opacity 0.25s, transform 0.25s;
}
.zen-zoom.is-visible {
  opacity: 0.92;
  pointer-events: auto;
  transform: translateY(0);
}
.zen-zoom.is-visible:hover {
  opacity: 1;
}

.zen-zoom__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.zen-zoom__btn:hover:not(:disabled) {
  background: var(--app-hover);
  color: var(--app-active-text);
}
.zen-zoom__btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.zen-zoom__pct {
  min-width: 52px;
  height: 26px;
  padding: 0 6px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--app-text);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s;
}
.zen-zoom__pct:hover {
  background: var(--app-hover);
}

/* 全屏编辑保存按钮：右上角悬浮（常驻，不自动隐藏） */
.zen-save {
  position: absolute;
  top: 4px;
  right: 6px;
  z-index: 20;
}

.zen-save__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 27px;
  height: 27px;
  padding: 0;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-bar-bg);
  color: var(--app-text-muted);
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.18);
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.zen-save__btn:hover {
  background: var(--app-hover);
  color: var(--app-text);
}
.zen-save__btn:active {
  transform: scale(0.95);
}

/* 有未保存更改：强调蓝底白图标，提示用户可点击保存 */
.zen-save__btn.is-dirty {
  background: var(--app-active-text);
  border-color: var(--app-active-text);
  color: #fff;
}
.zen-save__btn.is-dirty:hover {
  filter: brightness(1.08);
  background: var(--app-active-text);
  color: #fff;
}

.XinText-app__body {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.XinText-app__main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.XinText-app__editor {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.XinText-app__welcome {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 8px;
}

.welcome__logo {
  width: 48px;
  height: 48px;
  object-fit: contain;
  margin-bottom: 8px;
  opacity: 0.8;
}

.welcome__title {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--app-text);
}

.welcome__desc {
  margin: 0;
  font-size: 14px;
  color: var(--app-text-muted);
}

.welcome__hint {
  margin: 0;
  margin-top: 12px;
  font-size: 13px;
  color: var(--app-text-muted);
  opacity: 0.7;
}
</style>
