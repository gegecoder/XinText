<template>
  <header class="brand-bar">
    <div class="brand-bar__left">
      <div class="brand-bar__brand">
        <img :src="isDark ? '/logo_dark.png' : '/logo.png'" class="brand-bar__logo" alt="XinText" />
        <span class="brand-bar__name">{{ appName }}</span>
        <span class="brand-bar__sub">{{ appSubTitle }}</span>
      </div>
      <!-- 文件菜单 -->
      <div class="menu" ref="fileMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'file' }"
          @click="toggleMenu('file')"
        >
          {{ i18n('menu.file') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'file'" class="menu__dropdown" role="menu">
          <li role="menuitem" @click="runAndClose('new')">
            <span class="menu__label">{{ i18n('menu.new') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+N') }}</span>
          </li>
          <li role="menuitem" @click="runAndClose('open')">
            <span class="menu__label">{{ i18n('menu.open') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Shift+N') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile }" @click="runAndClose('save')">
            <span class="menu__label">{{ i18n('menu.save') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+S') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile }" @click="runAndClose('save-as')">
            <span class="menu__label">{{ i18n('menu.saveAs') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Shift+S') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile }" @click="runAndClose('print')">
            <span class="menu__label">{{ i18n('menu.print') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+P') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile || !pandocAvailable }"
            @click="runAndClose('export-html')"
          >
            <span class="menu__label">{{ i18n('menu.exportHtml') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile }"
            @click="runAndClose('export-pdf')"
          >
            <span class="menu__label">{{ i18n('menu.exportPdf') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile || !pandocAvailable }"
            @click="runAndClose('export-docx')"
          >
            <span class="menu__label">{{ i18n('menu.exportDocx') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile }"
            @click="runAndClose('export-txt')"
          >
            <span class="menu__label">{{ i18n('menu.exportTxt') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile }"
            :title="i18n('menu.exportMdTip')"
            @click="runAndClose('export-md')"
          >
            <span class="menu__label">{{ i18n('menu.exportMd') }}</span>
          </li>
        </ul>
      </div>
      <!-- 编辑菜单：仅在编辑模式可用，阅读模式置灰 -->
      <div class="menu" ref="editMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'edit' }"
          @click="toggleMenu('edit')"
        >
          {{ i18n('menu.edit') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'edit'" class="menu__dropdown" role="menu">
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' || !props.canUndo }" @click="runAndClose('undo')">
            <span class="menu__label">{{ i18n('menu.undo') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Z') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' || !props.canRedo }" @click="runAndClose('redo')">
            <span class="menu__label">{{ i18n('menu.redo') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Y') }}</span>
          </li>
        </ul>
      </div>
      <!-- 段落菜单：仅在编辑模式可用，阅读模式置灰 -->
      <div class="menu" ref="paragraphMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'paragraph' }"
          @click="toggleMenu('paragraph')"
        >
          {{ i18n('menu.paragraph') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'paragraph'" class="menu__dropdown" role="menu">
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile || mode === 'read' }"
            @click="runAndClose('paragraph-0')"
          >
            <span class="menu__label">{{ i18n('menu.paragraphBody') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+·') }}</span>
          </li>
          <li
            v-for="lvl in [1, 2, 3, 4, 5, 6]"
            :key="lvl"
            role="menuitem"
            :class="{ 'is-disabled': !hasFile || mode === 'read' }"
            @click="runAndClose(`paragraph-${lvl}` as MenuAction)"
          >
            <span class="menu__label">{{ i18n(`menu.heading${lvl}` as keyof typeof i18n) }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+' + lvl) }}</span>
          </li>
        </ul>
      </div>
      <!-- 插入菜单：仅在编辑模式可用，阅读模式置灰 -->
      <div class="menu" ref="insertMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'insert' }"
          @click="toggleMenu('insert')"
        >
          {{ i18n('menu.insert') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'insert'" class="menu__dropdown" role="menu">
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' || mode === 'source' }" @click="runAndClose('insert-paragraph-before' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertParagraphBefore') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+1') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' || mode === 'source' }" @click="runAndClose('insert-paragraph-after' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertParagraphAfter') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+2') }}</span>
          </li>
          <li class="menu__sep"></li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-image' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertImage') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+3') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-link' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertLink') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+4') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-quote' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertQuote') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+5') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-footnote' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertFootnote') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+6') }}</span>
          </li>
          <li
            role="menuitem"
            class="menu__table-item"
            :class="{ 'is-disabled': !hasFile || mode === 'read' }"
            @mouseenter="onTableItemEnter"
            @mouseleave="onTableItemLeave"
          >
            <span class="menu__label" @click="onTableLabelClick">{{ i18n('menu.insertTable') }}</span>
            <span class="menu__table-tail">
              <span class="menu__shortcut">{{ sc('Alt+7') }}</span>
              <span class="menu__table-chev">▸</span>
            </span>
            <div
              v-if="tableGridVisible"
              class="menu__table-grid"
              @mousemove="onGridMouseMove"
              @mouseenter="onGridMouseEnter"
              @mouseleave="onGridMouseLeave"
              @click="onGridClick"
            >
              <div class="menu__table-cells">
                <div
                  v-for="idx in 400"
                  :key="idx"
                  class="menu__table-cell"
                  :class="{ 'is-active': isCellActive(idx) }"
                ></div>
              </div>
              <div class="menu__table-info">{{ tableHoverRow }} × {{ tableHoverCol }}</div>
            </div>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-line' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertLine') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+8') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-toc-file' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertTocFile') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+9') }}</span>
          </li>
          <li class="menu__sep"></li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-inline-code' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertInlineCode') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+Q') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-code' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertCode') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+W') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('insert-formula' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.insertFormula') }}</span>
            <span class="menu__shortcut">{{ sc('Alt+E') }}</span>
          </li>
        </ul>
      </div>
      <!-- 格式菜单：仅在编辑模式可用，阅读模式置灰 -->
      <div class="menu" ref="formatMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'format' }"
          @click="toggleMenu('format')"
        >
          {{ i18n('menu.format') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'format'" class="menu__dropdown" role="menu">
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-bold' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatBold') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+B') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-italic' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatItalic') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+I') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-strike' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatStrike') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+D') }}</span>
          </li>
          <li class="menu__sep"></li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-list' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatList') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+L') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-ordered-list' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatOrderedList') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+O') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-check' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatCheck') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+J') }}</span>
          </li>
          <li class="menu__sep"></li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-outdent' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatOutdent') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Shift+I') }}</span>
          </li>
          <li role="menuitem" :class="{ 'is-disabled': !hasFile || mode === 'read' }" @click="runAndClose('format-indent' as MenuAction)">
            <span class="menu__label">{{ i18n('menu.formatIndent') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+Shift+O') }}</span>
          </li>
        </ul>
      </div>
      <!-- 查找菜单 -->
      <div class="menu" ref="findMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'find' }"
          @click="toggleMenu('find')"
        >
          {{ i18n('menu.find') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'find'" class="menu__dropdown" role="menu">
          <li role="menuitem" :class="{ 'is-disabled': !hasFile }" @click="runAndClose('find')">
            <span class="menu__label">{{ i18n('menu.find') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+F') }}</span>
          </li>
          <li
            role="menuitem"
            :class="{ 'is-disabled': !hasFile || mode === 'read' }"
            :title="hasFile && mode === 'read' ? i18n('menu.replaceReadonlyTip') : ''"
            @click="runAndClose('replace')"
          >
            <span class="menu__label">{{ i18n('menu.replace') }}</span>
            <span class="menu__shortcut">{{ sc('Ctrl+H') }}</span>
          </li>
        </ul>
      </div>
      <!-- 帮助菜单 -->
      <div class="menu" ref="helpMenuRef">
        <button
          class="menu__trigger"
          :class="{ 'is-open': activeMenu === 'help' }"
          @click="toggleMenu('help')"
        >
          {{ i18n('menu.help') }}<span class="menu__chevron">▾</span>
        </button>
        <ul v-if="activeMenu === 'help'" class="menu__dropdown" role="menu">
          <li role="menuitem" @click="runAndClose('feedback')">
            <span class="menu__label">{{ i18n('menu.feedback') }}</span>
          </li>
          <li role="menuitem" @click="runAndClose('log')">
            <span class="menu__label">{{ i18n('menu.log') }}</span>
          </li>
          <li role="menuitem" @click="runAndClose('about')">
            <span class="menu__label">{{ i18n('menu.about') }}</span>
          </li>
        </ul>
      </div>
    </div>
    <!-- 右侧：主题切换 + 设置入口 -->
    <div class="brand-bar__right">
      <button
        class="brand-bar__icon-btn"
        :title="isDark ? i18n('menu.switchToLight') : i18n('menu.switchToDark')"
        @click="emit('toggle-theme')"
      >
        <!-- 暗色时显示太阳（点击切到亮色），亮色时显示月亮（点击切到暗色） -->
        <svg v-if="isDark" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
        </svg>
        <svg v-else width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z" />
        </svg>
      </button>
      <button class="brand-bar__icon-btn" :title="i18n('menu.settings')" @click="emit('settings')">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { Browser } from '@wailsio/runtime'
import appConfig from '@app-config'
import { i18n } from '../i18n'
import { shortcutLabel as sc } from '../composables/platform'

const props = defineProps<{
  appName: string
  appSubTitle: string
  /** 当前是否有打开的文件：无文件时保存/另存为/导出置灰 */
  hasFile: boolean
  /** pandoc 可用性：不可用时 HTML/DOCX/TXT 导出菜单项置灰（PDF 走 msedge，不依赖 pandoc） */
  pandocAvailable: boolean
  /** 当前编辑模式：阅读模式下「替换」置灰（只读） */
  mode?: 'read' | 'wysiwyg' | 'source'
  /** 当前是否暗色主题 */
  isDark: boolean
  /** undo/redo 可用状态（从 Vditor 工具栏按钮同步） */
  canUndo: boolean
  canRedo: boolean
}>()

const emit = defineEmits<{
  (e: 'new'): void
  (e: 'open'): void
  (e: 'save'): void
  (e: 'save-as'): void
  (e: 'print'): void
  (e: 'export-html'): void
  (e: 'export-pdf'): void
  (e: 'export-docx'): void
  (e: 'export-txt'): void
  (e: 'export-md'): void
  (e: 'find'): void
  (e: 'replace'): void
  (e: 'paragraph', level: 0 | 1 | 2 | 3 | 4 | 5 | 6): void
  (e: 'insert', kind: 'insert-before' | 'insert-after' | 'image' | 'link' | 'quote' | 'table' | 'line' | 'inline-code' | 'code' | 'footnote' | 'formula'): void
  (e: 'insert-toc-file'): void
  (e: 'insert-table', rows: number, cols: number): void
  (e: 'format', type: 'bold' | 'italic' | 'strike' | 'list' | 'ordered-list' | 'check' | 'outdent' | 'indent'): void
  (e: 'undo'): void
  (e: 'redo'): void
  (e: 'about'): void
  (e: 'log'): void
  (e: 'settings'): void
  (e: 'toggle-theme'): void
}>()

const activeMenu = ref<'file' | 'edit' | 'paragraph' | 'insert' | 'format' | 'find' | 'help' | null>(null)
const fileMenuRef = ref<HTMLElement | null>(null)
const editMenuRef = ref<HTMLElement | null>(null)
const paragraphMenuRef = ref<HTMLElement | null>(null)
const insertMenuRef = ref<HTMLElement | null>(null)
const formatMenuRef = ref<HTMLElement | null>(null)
const findMenuRef = ref<HTMLElement | null>(null)
const helpMenuRef = ref<HTMLElement | null>(null)

// 表格网格浮层：hover 表格菜单项时显示 20x20 网格，mousemove 实时
// 更新高亮区域，click 插入选定 n*n 表格。隐藏时机：仅当鼠标已进入
// 网格后再离开时立即隐藏；从 li 滑入网格穿过 gap 期间用 timer 延迟
// 隐藏，由 grid 的 mouseenter 取消，避免提前消失。
const tableGridVisible = ref(false)
const tableHoverRow = ref(2)
const tableHoverCol = ref(2)
let tableHideTimer: number | null = null

function onTableItemEnter() {
  if (!props.hasFile || props.mode === 'read') return
  if (tableHideTimer) { clearTimeout(tableHideTimer); tableHideTimer = null }
  tableGridVisible.value = true
  // 默认 2x2，用户滑入网格后通过 mousemove 更新
  tableHoverRow.value = 2
  tableHoverCol.value = 2
}

function onTableItemLeave() {
  // 延迟隐藏，给用户时间从 li 穿过 gap 滑入 grid；
  // 进入 grid 后 onGridMouseEnter 会取消该计时
  if (tableHideTimer) clearTimeout(tableHideTimer)
  tableHideTimer = window.setTimeout(() => {
    tableGridVisible.value = false
  }, 200)
}

function onGridMouseEnter() {
  // 已进入网格，取消 li leave 安排的隐藏
  if (tableHideTimer) { clearTimeout(tableHideTimer); tableHideTimer = null }
}

function onGridMouseLeave() {
  // 离开网格，立即隐藏
  if (tableHideTimer) { clearTimeout(tableHideTimer); tableHideTimer = null }
  tableGridVisible.value = false
}

function onGridMouseMove(e: MouseEvent) {
  const target = e.currentTarget as HTMLElement
  const cells = target.querySelector('.menu__table-cells') as HTMLElement | null
  if (!cells) return
  const rect = cells.getBoundingClientRect()
  // 鼠标在 cells 区域外（如停在 info 标签上）不更新
  if (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom) return
  const cellW = rect.width / 20
  const cellH = rect.height / 20
  const col = Math.min(20, Math.max(1, Math.ceil((e.clientX - rect.left) / cellW)))
  const row = Math.min(20, Math.max(1, Math.ceil((e.clientY - rect.top) / cellH)))
  tableHoverRow.value = row
  tableHoverCol.value = col
}

function isCellActive(idx: number): boolean {
  const row = Math.ceil(idx / 20)
  const col = ((idx - 1) % 20) + 1
  return row <= tableHoverRow.value && col <= tableHoverCol.value
}

function onGridClick() {
  if (!props.hasFile || props.mode === 'read') return
  const r = tableHoverRow.value
  const c = tableHoverCol.value
  if (tableHideTimer) { clearTimeout(tableHideTimer); tableHideTimer = null }
  activeMenu.value = null
  tableGridVisible.value = false
  emit('insert-table', r, c)
}

/** 直接点击表格项的标签（未滑入网格）：插入默认 2x2 表格 */
function onTableLabelClick() {
  if (!props.hasFile || props.mode === 'read') return
  if (tableHideTimer) { clearTimeout(tableHideTimer); tableHideTimer = null }
  activeMenu.value = null
  tableGridVisible.value = false
  emit('insert-table', 2, 2)
}

function toggleMenu(menu: 'file' | 'edit' | 'paragraph' | 'insert' | 'format' | 'find' | 'help') {
  activeMenu.value = activeMenu.value === menu ? null : menu
}

type MenuAction =
  | 'new' | 'open' | 'save' | 'save-as' | 'print'
  | 'export-html' | 'export-pdf' | 'export-docx' | 'export-txt' | 'export-md'
  | 'find' | 'replace' | 'about' | 'log' | 'feedback'
  | 'paragraph-0' | 'paragraph-1' | 'paragraph-2' | 'paragraph-3' | 'paragraph-4' | 'paragraph-5' | 'paragraph-6'
  | 'insert-paragraph-before' | 'insert-paragraph-after' | 'insert-image' | 'insert-link' | 'insert-quote' | 'insert-table' | 'insert-line' | 'insert-toc-file' | 'insert-inline-code' | 'insert-code' | 'insert-footnote' | 'insert-formula'
  | 'format-bold' | 'format-italic' | 'format-strike' | 'format-list' | 'format-ordered-list' | 'format-check' | 'format-outdent' | 'format-indent'
  | 'undo' | 'redo'

/** 在系统默认浏览器打开问题反馈页面（链接配置在 app.config.json） */
function openFeedback() {
  const url = appConfig.feedback?.url
  if (!url) return
  Browser.OpenURL(url).catch(() => window.open(url, '_blank'))
}

function runAndClose(action: MenuAction) {
  // 无打开文件时保存/另存为/导出/查找/替换置灰，不触发
  if (action !== 'new' && action !== 'open' && action !== 'about' && action !== 'log' && action !== 'feedback' && !props.hasFile) return
  // pandoc 不可用时 HTML/DOCX 导出置灰（pandoc 仅用于 HTML/DOCX；
  // TXT 导出由前端实现，PDF 用 msedge headless 渲染 Vditor HTML，均不依赖 pandoc）
  if ((action === 'export-html' || action === 'export-docx') && !props.pandocAvailable) return
  // 阅读模式只读，替换/段落/插入操作置灰
  if (action === 'replace' && props.mode === 'read') return
  if (typeof action === 'string' && action.startsWith('paragraph-') && props.mode === 'read') return
  if (typeof action === 'string' && action.startsWith('insert-') && props.mode === 'read') return
  // 段落(上方/下方) 在源码编辑模式下也不可用（依赖 Vditor IR/WYSIWYG toolbar）
  if ((action === 'insert-paragraph-before' || action === 'insert-paragraph-after') && props.mode === 'source') return
  if (typeof action === 'string' && action.startsWith('format-') && props.mode === 'read') return
  if ((action === 'undo' || action === 'redo') && props.mode === 'read') return
  if (action === 'undo' && !props.canUndo) return
  if (action === 'redo' && !props.canRedo) return
  activeMenu.value = null
  switch (action) {
    case 'new': emit('new'); break
    case 'open': emit('open'); break
    case 'save': emit('save'); break
    case 'save-as': emit('save-as'); break
    case 'print': emit('print'); break
    case 'export-html': emit('export-html'); break
    case 'export-pdf': emit('export-pdf'); break
    case 'export-docx': emit('export-docx'); break
    case 'export-txt': emit('export-txt'); break
    case 'export-md': emit('export-md'); break
    case 'find': emit('find'); break
    case 'replace': emit('replace'); break
    case 'about': emit('about'); break
    case 'log': emit('log'); break
    case 'feedback': openFeedback(); break
    case 'paragraph-0': emit('paragraph', 0); break
    case 'paragraph-1': emit('paragraph', 1); break
    case 'paragraph-2': emit('paragraph', 2); break
    case 'paragraph-3': emit('paragraph', 3); break
    case 'paragraph-4': emit('paragraph', 4); break
    case 'paragraph-5': emit('paragraph', 5); break
    case 'paragraph-6': emit('paragraph', 6); break
    case 'insert-paragraph-before': emit('insert', 'insert-before'); break
    case 'insert-paragraph-after': emit('insert', 'insert-after'); break
    case 'insert-image': emit('insert', 'image'); break
    case 'insert-link': emit('insert', 'link'); break
    case 'insert-quote': emit('insert', 'quote'); break
    case 'insert-table': emit('insert', 'table'); break
    case 'insert-line': emit('insert', 'line'); break
    case 'insert-toc-file': emit('insert-toc-file'); break
    case 'insert-inline-code': emit('insert', 'inline-code'); break
    case 'insert-code': emit('insert', 'code'); break
    case 'insert-footnote': emit('insert', 'footnote'); break
    case 'insert-formula': emit('insert', 'formula'); break
    case 'format-bold': emit('format', 'bold'); break
    case 'format-italic': emit('format', 'italic'); break
    case 'format-strike': emit('format', 'strike'); break
    case 'format-list': emit('format', 'list'); break
    case 'format-ordered-list': emit('format', 'ordered-list'); break
    case 'format-check': emit('format', 'check'); break
    case 'format-outdent': emit('format', 'outdent'); break
    case 'format-indent': emit('format', 'indent'); break
    case 'undo': emit('undo'); break
    case 'redo': emit('redo'); break
  }
}

function onDocPointerDown(e: MouseEvent) {
  if (!activeMenu.value) return
  const refs = [fileMenuRef.value, editMenuRef.value, paragraphMenuRef.value, insertMenuRef.value, formatMenuRef.value, findMenuRef.value, helpMenuRef.value].filter(Boolean)
  if (refs.every((r) => !r!.contains(e.target as Node))) {
    activeMenu.value = null
  }
}

function onDocKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') activeMenu.value = null
}

onMounted(() => {
  document.addEventListener('mousedown', onDocPointerDown)
  window.addEventListener('keydown', onDocKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocPointerDown)
  window.removeEventListener('keydown', onDocKeydown)
})
</script>

<style scoped>
.brand-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 40px;
  padding: 0 14px;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-bar-bg);
  flex-shrink: 0;
}

.brand-bar__left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.brand-bar__brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-right: 12px;
  margin-right: 4px;
  border-right: 1px solid var(--app-border);
}

.brand-bar__logo {
  width: 20px;
  height: 20px;
  object-fit: contain;
}

.brand-bar__name {
  font-size: 14px;
  font-weight: 700;
  color: var(--app-text);
  white-space: nowrap;
}

.brand-bar__sub {
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
}

.menu {
  position: relative;
}

.menu__trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--app-text);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.menu__trigger:hover,
.menu__trigger.is-open {
  background: var(--app-hover);
  border-color: var(--app-border);
}

.menu__chevron {
  font-size: 10px;
  opacity: 0.7;
}

.menu__dropdown {
  position: absolute;
  top: calc(100% + 2px);
  left: 0;
  z-index: 1000;
  min-width: 200px;
  margin: 0;
  padding: 4px 0;
  list-style: none;
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
}

.menu__dropdown li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 6px 14px;
  font-size: 13px;
  color: var(--app-text);
  cursor: pointer;
  white-space: nowrap;
}

.menu__dropdown li:hover {
  background: var(--app-hover);
  color: var(--app-active-text);
}

.menu__shortcut {
  font-size: 11px;
  color: var(--app-text-muted);
}

.menu__dropdown li:hover .menu__shortcut {
  color: var(--app-active-text);
  opacity: 0.8;
}

/* pandoc 不可用时导出菜单项置灰 */
.menu__dropdown li.is-disabled {
  color: var(--app-text-muted);
  opacity: 0.55;
  cursor: not-allowed;
}

.menu__dropdown li.is-disabled:hover {
  background: transparent;
  color: var(--app-text-muted);
}

/* 分组分隔线：1px 灰色细线，覆盖默认 li 的 flex/内边距/光标/hover */
.menu__dropdown li.menu__sep {
  display: block;
  height: 0;
  margin: 4px 0;
  padding: 0;
  border-top: 1px solid var(--app-border);
  cursor: default;
}

.menu__dropdown li.menu__sep:hover {
  background: transparent;
  color: inherit;
}

/* 表格菜单项：右侧延伸出 10x10 网格浮层 */
.menu__dropdown li.menu__table-item {
  position: relative;
}

.menu__table-tail {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.menu__table-chev {
  color: var(--app-text-muted);
  font-size: 10px;
  line-height: 1;
}

/* 网格浮层：紧贴表格项右侧，超出 dropdown 不裁剪 */
.menu__table-grid {
  position: absolute;
  top: 0;
  left: 100%;
  margin-left: 4px;
  z-index: 1001;
  padding: 6px;
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  cursor: pointer;
}

.menu__table-cells {
  display: grid;
  grid-template-columns: repeat(20, 12px);
  grid-template-rows: repeat(20, 12px);
  gap: 1px;
  background: var(--app-border);
}

.menu__table-cell {
  background: var(--app-bg);
  border: none;
}

.menu__table-cell.is-active {
  background: var(--app-active-text);
}

.menu__table-info {
  margin-top: 6px;
  text-align: center;
  font-size: 11px;
  color: var(--app-text-muted);
  line-height: 1;
}

.brand-bar__right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.brand-bar__icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--app-text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.brand-bar__icon-btn:hover {
  background: var(--app-hover);
  border-color: var(--app-border);
  color: var(--app-text);
}

</style>
