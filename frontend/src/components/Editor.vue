<template>
  <div ref="editorRef" class="XinText-editor" />
  <!-- 查找/替换浮层（Ctrl+F / Ctrl+H）：独立根节点，避免 Vditor 挂载时清空 -->
  <div v-if="findVisible" class="ot-fr" :class="{ 'ot-fr--replace': replaceMode }">
    <div class="ot-fr__row">
      <input
        ref="findInputRef"
        v-model="findKeyword"
        class="ot-fr__input"
        :class="{ 'is-no-result': isNoResult }"
        type="text"
        :placeholder="i18n('editor.find')"
        @compositionstart="findComposing = true"
        @compositionend="findComposing = false"
        @keydown="onFindKeydown"
      >
      <span class="ot-fr__count" :class="{ 'is-none': isNoResult }">
        {{ matchTotal ? `${matchIndex + 1}/${matchTotal}` : '0/0' }}
      </span>
      <button class="ot-fr__btn" :title="i18n('editor.prev')" :disabled="!matchTotal" @click="gotoPrev">
        <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
          <path d="M4 6.5l4-4 4 4M4 11.5l4-4 4 4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" fill="none" />
        </svg>
      </button>
      <button class="ot-fr__btn" :title="i18n('editor.next')" :disabled="!matchTotal" @click="gotoNext">
        <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
          <path d="M4 4.5l4 4 4-4M4 9.5l4 4 4-4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" fill="none" />
        </svg>
      </button>
      <button
        class="ot-fr__btn ot-fr__btn--toggle"
        :class="{ 'is-on': replaceMode }"
        :title="replaceMode ? i18n('editor.collapseReplace') : sc(i18n('editor.replaceToggle'))"
        @click="toggleReplaceMode"
      >
        <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
          <path d="M3 4h7.5a2.5 2.5 0 0 1 0 5H7m0 0l2-2m-2 2l2 2M13 12H5.5a2.5 2.5 0 0 1 0-5H9" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
        </svg>
      </button>
      <button class="ot-fr__btn" :title="i18n('editor.close')" @click="closeFindBar">
        <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
          <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" fill="none" />
        </svg>
      </button>
    </div>
    <div v-if="replaceMode" class="ot-fr__row">
      <input
        ref="replaceInputRef"
        v-model="replaceKeyword"
        class="ot-fr__input"
        type="text"
        :placeholder="i18n('editor.replacePlaceholder')"
        @keydown="onReplaceKeydown"
      >
      <button class="ot-fr__action" :disabled="!matchTotal" :title="i18n('editor.replaceOneTip')" @click="replaceOne">
        {{ i18n('editor.replace') }}
      </button>
      <button class="ot-fr__action" :disabled="!matchTotal" :title="i18n('editor.replaceAllTip')" @click="replaceAll">
        {{ i18n('editor.replaceAll') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { ImageService } from '../../bindings/XinText/internal/service'
import { resolveImagesIn } from '../composables/resolveImages'
import { mountLinkInterception } from '../composables/useLinkInterception'
import { EMOJI_MAP } from '../data/emoji-map'
import { i18n, editorLangCode } from '../i18n'
import { shortcutLabel as sc } from '../composables/platform'

const props = defineProps<{
  modelValue: string
  /** 'wysiwyg' -> Vditor ir (即时渲染); 'source' -> Vditor sv (分屏预览) */
  mode?: 'wysiwyg' | 'source'
  /** Directory of the currently open file; pasted images land in its assets/ folder. */
  docPath?: string | null
  /** 'classic' (light) or 'dark'. */
  theme?: 'classic' | 'dark'
  /** 全屏编辑模式：临时移除部分 Vditor 内置控件（DOM 级，退出时原样插回） */
  zen?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  /** 命中本地 .md 文件链接 */
  (e: 'open-md', path: string): void
  /** 命中 web 链接 */
  (e: 'open-browser', url: string): void
  /** undo/redo 可用状态变化（Vditor 工具栏按钮 disabled 同步） */
  (e: 'undo-redo-state', canUndo: boolean, canRedo: boolean): void
}>()

const editorRef = ref<HTMLDivElement | null>(null)
const docPathRef = computed(() => props.docPath ?? null)
// markdown 源码：用于在链接拦截时从原文恢复 lute 渲染丢失的反斜杠
const markdownRef = computed(() => props.modelValue ?? '')
let detachLinkInterception: (() => void) | null = null

let vditor: Vditor | null = null
let ready = false
// Observer that rewrites relative-path images to data URLs for rendering
let imgObserver: MutationObserver | null = null
// Guard against echo: when we setValue from an external modelValue change,
// Vditor's input must not re-emit the same content.
let suppressInput = false

const MODE_MAP: Record<'wysiwyg' | 'source', 'ir' | 'sv'> = {
  wysiwyg: 'ir',
  source: 'sv',
}

function currentVditorMode(): 'ir' | 'sv' {
  return MODE_MAP[props.mode ?? 'wysiwyg']
}

function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

// Vditor 4.x routes every pasted/dropped image through upload.handler. We read
// each file as a data URL, persist it to disk via the Go service, and insert the
// resulting relative path as a Markdown image at the caret.
async function handleImageUpload(files: File[]): Promise<null> {
  for (const file of files) {
    try {
      const dataURL = await fileToDataURL(file)
      // 保存到设置中配置的图片目录（默认 %AppData%/XinText/images），返回绝对路径引用
      const abs = await ImageService.SaveImage(dataURL)
      vditor?.insertMD(`\n![](${abs})\n`)
    } catch (e) {
      console.error('save image failed', e)
    }
  }
  return null
}

// 通过编辑器内部 Tip 显示操作反馈（show(text, time) 自动定时隐藏）
function showEditorTip(text: string) {
  ;(vditor?.vditor as any)?.tip?.show?.(text, 1500)
}

// 复制预览内容：富文本 HTML + 纯文本 Markdown 双格式写入剪贴板。
// 粘贴到公众号/邮件等保留格式；粘贴到纯文本编辑器得到 Markdown 源文。
async function copyPreviewContent() {
  const md = props.modelValue ?? ''
  const html = vditor?.getHTML() ?? ''
  try {
    if (typeof ClipboardItem !== 'undefined' && navigator.clipboard?.write) {
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/html': new Blob([html], { type: 'text/html' }),
          'text/plain': new Blob([md], { type: 'text/plain' }),
        }),
      ])
    } else {
      await navigator.clipboard.writeText(md)
    }
    showEditorTip(i18n('editor.copied'))
  } catch (e) {
    console.error('copy failed', e)
    try {
      await navigator.clipboard.writeText(md)
      showEditorTip(i18n('editor.copied'))
    } catch {
      /* 剪贴板不可用时忽略 */
    }
  }
}

// Vditor 4.x does not expose a public setMode(); switch modes by clicking the
// edit-mode toolbar buttons (data-mode="ir" | "sv" | "wysiwyg").
function switchMode(mode: 'ir' | 'sv') {
  if (!vditor) return
  const editModeEl = (vditor.vditor as any)?.toolbar?.elements?.['edit-mode'] as HTMLElement | undefined
  if (!editModeEl) return
  const btn = editModeEl.querySelector(`button[data-mode="${mode}"]`) as HTMLElement | null
  btn?.click()
}

/**
 * 源码模式：重置光标到开头并把滚动条拉回顶部。
 * Chromium 在 textarea.value 赋值后 selectionStart 会跳到末尾，
 * 后续 focus() 会让浏览器把滚动条拉到底部以显示末尾光标。
 */
function resetSvScrollTop() {
  if (currentVditorMode() !== 'sv') return
  const ta = editorRef.value?.querySelector('textarea.vditor-sv') as HTMLTextAreaElement | null
  if (ta) {
    ta.setSelectionRange(0, 0)
    ta.scrollTop = 0
  }
}

function setEditorValue(value: string) {
  if (!vditor || !ready) return
  if (vditor.getValue() === value) return
  suppressInput = true
  vditor.setValue(value)
  // Vditor's setValue resolves synchronously; release the guard on next tick.
  setTimeout(() => {
    suppressInput = false
    updateSvGutterLines()
    // 初始加载 / 切换文件后，源码模式光标与滚动条重置到顶部
    resetSvScrollTop()
  }, 0)
}

/**
 * Vditor 4 的源码模式是原生 <textarea class="vditor-sv">（已无 CodeMirror），
 * 不支持行号。这里在其左侧同步一个"镜像行号槽"：
 * - 保留 textarea 原生 pre-wrap 软折行（不能改 pre，否则会出现水平滚动条
 *   且破坏 Vditor 源码↔预览的按比例滚动联动）
 * - 行号槽内每行放"行号 + 透明文本"，排版参数（字体/行高/折行宽度）与
 *   textarea 完全一致，使软折行撑出的高度也能逐行对齐
 * - textarea 滚动时行号槽整体 translateY 同步
 */
function refreshSvGutter() {
  const root = editorRef.value
  if (!root) return
  const ta = root.querySelector('textarea.vditor-sv') as (HTMLTextAreaElement & { _svBound?: boolean; _svResizeObs?: ResizeObserver }) | null
  let gutter = root.querySelector('.XinText-sv-gutter') as HTMLElement | null

  // ir 模式下 textarea 不存在或被隐藏 → 行号槽与分隔条隐藏
  const visible = !!ta && ta.parentElement !== null && ta.getBoundingClientRect().height > 0
  if (!visible) {
    if (gutter) gutter.style.display = 'none'
    const resizer = root.querySelector('.ot-sv-resizer') as HTMLElement | null
    if (resizer) resizer.style.display = 'none'
    return
  }

  if (!gutter) {
    gutter = document.createElement('div')
    gutter.className = 'XinText-sv-gutter'
    const inner = document.createElement('div')
    inner.className = 'XinText-sv-gutter__inner'
    gutter.appendChild(inner)
  }
  if (gutter.parentElement !== ta!.parentElement || gutter.nextElementSibling !== ta) {
    ta!.parentElement!.insertBefore(gutter, ta)
  }
  gutter.style.display = ''

  if (!ta!._svBound) {
    ta!._svBound = true
    ta!.addEventListener('input', updateSvGutterLines)
    ta!.addEventListener('input', () => {
      // 查找栏打开时，编辑内容变化后刷新匹配
      if (findVisible.value) scheduleRunFind(160)
    })
    ta!.addEventListener('scroll', syncSvGutterScroll)
    // 宽度变化（窗口缩放/侧栏调整）后镜像宽度需重新计算
    ta!._svResizeObs = new ResizeObserver(() => updateSvGutterLines())
    ta!._svResizeObs.observe(ta!)
  }
  updateSvGutterLines()
  syncSvGutterScroll()

  // 源码区与预览区之间的可拖拽分隔条
  ensureSvResizer(ta!)
}

function escapeLineText(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function updateSvGutterLines() {
  const root = editorRef.value
  if (!root) return
  const ta = root.querySelector('textarea.vditor-sv') as HTMLTextAreaElement | null
  const inner = root.querySelector('.XinText-sv-gutter__inner') as HTMLElement | null
  if (!ta || !inner) return
  const lines = ta.value.split('\n')
  let html = ''
  for (let i = 0; i < lines.length; i++) {
    html +=
      '<div class="sv-line"><span class="sv-line__num">' +
      (i + 1) +
      '</span><span class="sv-line__text">' +
      escapeLineText(lines[i]) +
      '</span></div>'
  }
  inner.innerHTML = html
  // 镜像文本的可用宽度必须等于 textarea 内容盒宽度，折行点才一致：
  // 46px 行号区 + (clientWidth - 左右 padding 10/9)
  inner.style.width = 46 + (ta.clientWidth - 19) + 'px'
}

function syncSvGutterScroll() {
  const root = editorRef.value
  if (!root) return
  const ta = root.querySelector('textarea.vditor-sv') as HTMLTextAreaElement | null
  const inner = root.querySelector('.XinText-sv-gutter__inner') as HTMLElement | null
  if (ta && inner) inner.style.transform = `translateY(${-ta.scrollTop}px)`
}

// —— 源码区 / 预览区宽度拖拽 ——
const SV_WIDTH_KEY = 'XinText.sv.width'
const SV_MIN = 500       // 源码区最小宽度
const SV_PREVIEW_MIN = 200 // 预览区最小宽度

function applySvWidth(ta: HTMLTextAreaElement, w: number) {
  const content = ta.parentElement as HTMLElement
  const max = content.clientWidth - SV_PREVIEW_MIN
  const next = Math.min(max, Math.max(SV_MIN, w))
  ta.style.flex = '0 0 auto'
  ta.style.width = next + 'px'
  ta.style.maxWidth = '70%'
  ta.style.minWidth = '30%'
  return next
}

function ensureSvResizer(ta: HTMLTextAreaElement) {
  const content = ta.parentElement as HTMLElement
  let resizer = content.querySelector('.ot-sv-resizer') as HTMLElement | null
  if (!resizer) {
    resizer = document.createElement('div')
    resizer.className = 'ot-sv-resizer'
    // 插入到 textarea 之后、preview 之前
    content.insertBefore(resizer, ta.nextSibling)
    let startX = 0
    let startWidth = 0
    const onMove = (e: MouseEvent) => {
      const w = applySvWidth(ta, startWidth + (e.clientX - startX))
      localStorage.setItem(SV_WIDTH_KEY, String(w))
    }
    const onUp = () => {
      document.removeEventListener('mousemove', onMove)
      document.removeEventListener('mouseup', onUp)
      document.body.style.userSelect = ''
      document.body.style.cursor = ''
      resizer!.classList.remove('is-dragging')
    }
    resizer.addEventListener('mousedown', (e) => {
      if (e.button !== 0) return
      startX = e.clientX
      startWidth = ta.getBoundingClientRect().width
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
      document.body.style.userSelect = 'none'
      document.body.style.cursor = 'col-resize'
      resizer!.classList.add('is-dragging')
      e.preventDefault()
    })
  }
  resizer.style.display = ''
  // 应用持久化宽度（首次用 localStorage，否则保持当前）
  const saved = Number(localStorage.getItem(SV_WIDTH_KEY))
  if (saved >= SV_MIN) {
    applySvWidth(ta, saved)
  } else if (!ta.style.width) {
    // 默认 50%
    applySvWidth(ta, content.clientWidth / 2)
  }
}

/**
 * 查找 / 替换（源码模式操作 textarea；预览编辑操作 ir contenteditable）
 * - sv：setSelectionRange 选中 + execCommand/setRangeText 替换（保留撤销栈）
 * - ir：TreeWalker 收集文本节点匹配，Range 选中定位、Range 改写后派发 input
 *   让 Vditor 从 DOM 重新同步 Markdown
 */
const findVisible = ref(false)
const replaceMode = ref(false)
const findKeyword = ref('')
const replaceKeyword = ref('')
const findInputRef = ref<HTMLInputElement | null>(null)
const replaceInputRef = ref<HTMLInputElement | null>(null)
const matchTotal = ref(0)
const matchIndex = ref(-1)
const findComposing = ref(false) // 中文输入法组字中不做定位，避免焦点跳动打断拼音
const isNoResult = computed(() => findKeyword.value.length > 0 && matchTotal.value === 0)

interface IrMatch {
  node: Text
  start: number
  end: number
}
let svOffsets: number[] = []
let irMatches: IrMatch[] = []
let findDebounce: ReturnType<typeof setTimeout> | null = null
let irInputBound = false

function getEditable():
  | { kind: 'sv'; el: HTMLTextAreaElement }
  | { kind: 'ir'; el: HTMLElement }
  | null {
  const root = editorRef.value
  if (!root) return null
  // sv 模式：textarea 可见（有高度）
  const ta = root.querySelector('textarea.vditor-sv') as HTMLTextAreaElement | null
  if (ta && ta.getBoundingClientRect().height > 0) {
    return { kind: 'sv', el: ta }
  }
  // ir/wysiwyg 模式：遍历所有 contenteditable 元素，找可见的那个
  // （DOM 中可能同时存在 wysiwyg 和 ir 两个 contenteditable，只有一个可见）
  const elements = Array.from(root.querySelectorAll('[contenteditable="true"]'))
  for (const el of elements) {
    if ((el as HTMLElement).getBoundingClientRect().height > 0) {
      return { kind: 'ir', el: el as HTMLElement }
    }
  }
  return null
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** 收集 ir 内容区所有文本节点中的匹配（跳过非编辑控件/脚本样式） */
function collectIrMatches(root: HTMLElement, lower: string): IrMatch[] {
  const out: IrMatch[] = []
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const p = (node as Text).parentElement
      if (!p) return NodeFilter.FILTER_REJECT
      if (p.closest('[contenteditable="false"], .vditor-ir__preview, .vditor-toolbar, script, style')) {
        return NodeFilter.FILTER_REJECT
      }
      return node.nodeValue && node.nodeValue.toLowerCase().includes(lower)
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_REJECT
    },
  })
  let n: Node | null
  while ((n = walker.nextNode())) {
    const text = (n.textContent || '').toLowerCase()
    let i = 0
    while ((i = text.indexOf(lower, i)) !== -1) {
      out.push({ node: n as Text, start: i, end: i + lower.length })
      i += Math.max(lower.length, 1)
    }
  }
  return out
}

function runFind() {
  const kw = findKeyword.value
  const target = getEditable()
  if (!target || !kw) {
    matchTotal.value = 0
    matchIndex.value = -1
    svOffsets = []
    irMatches = []
    return
  }
  const lower = kw.toLowerCase()
  if (target.kind === 'sv') {
    const text = target.el.value.toLowerCase()
    svOffsets = []
    let i = 0
    while ((i = text.indexOf(lower, i)) !== -1) {
      svOffsets.push(i)
      i += Math.max(lower.length, 1)
    }
    irMatches = []
  } else {
    irMatches = collectIrMatches(target.el, lower)
    svOffsets = []
  }
  const total = target.kind === 'sv' ? svOffsets.length : irMatches.length
  matchTotal.value = total
  if (!total) {
    matchIndex.value = -1
    return
  }
  if (matchIndex.value < 0 || matchIndex.value >= total) matchIndex.value = 0
  revealCurrent()
}

function scheduleRunFind(delay = 120) {
  if (findDebounce) clearTimeout(findDebounce)
  findDebounce = setTimeout(runFind, delay)
}

/** 定位并高亮当前匹配 */
function revealCurrent() {
  const target = getEditable()
  if (!target || matchIndex.value < 0) return
  const kw = findKeyword.value
  if (target.kind === 'sv') {
    const ta = target.el
    const off = svOffsets[matchIndex.value]
    if (off === undefined) return
    ta.focus()
    ta.setSelectionRange(off, off + kw.length)
    // 手动滚动使选中行可见（WebView2 的 setSelectionRange 不保证自动滚动）
    const maxScroll = ta.scrollHeight - ta.clientHeight
    const lineNum = (ta.value.substring(0, off).match(/\n/g) || []).length
    const lineEstimate = lineNum * 22 - Math.floor(ta.clientHeight / 3)
    const ratio = off / Math.max(ta.value.length, 1)
    // 折行时 lineNum*22 低估底部位置；匹配在末尾 20% 时改用比例×scrollHeight 估算
    const scrollTarget = ratio > 0.8
      ? ratio * maxScroll
      : lineEstimate
    ta.scrollTop = Math.max(0, Math.min(scrollTarget, maxScroll))
    // 不归还焦点——WebView2 中 textarea 失焦后选中高亮会消失
    // 全局 keydown 监听处理 Enter/Esc 导航
  } else {
    const m = irMatches[matchIndex.value]
    if (!m || !m.node.isConnected) return
    const range = document.createRange()
    range.setStart(m.node, m.start)
    range.setEnd(m.node, m.end)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(range)
    m.node.parentElement?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }
}

function gotoNext() {
  if (!matchTotal.value) return
  matchIndex.value = (matchIndex.value + 1) % matchTotal.value
  revealCurrent()
}

function gotoPrev() {
  if (!matchTotal.value) return
  matchIndex.value = (matchIndex.value - 1 + matchTotal.value) % matchTotal.value
  revealCurrent()
}

function onFindKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) gotoPrev()
    else gotoNext()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    closeFindBar()
  }
}

function onReplaceKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    replaceOne()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    closeFindBar()
  }
}

/** 替换当前匹配；替换后重建匹配并定位到下一处。
 *  matchIndex 保持不变即指向下一个（被替换的匹配已从列表移除），
 *  若 matchIndex 超出新 total，runFind 自动回绕到 0。 */
function replaceOne() {
  const kw = findKeyword.value
  const repl = replaceKeyword.value
  const target = getEditable()
  if (!target || !kw || !matchTotal.value || matchIndex.value < 0) return

  if (target.kind === 'sv') {
    const ta = target.el
    const off = svOffsets[matchIndex.value]
    if (off === undefined) return
    ta.focus()
    ta.setSelectionRange(off, off + kw.length)
    let ok = false
    try {
      ok = document.execCommand('insertText', false, repl)
    } catch {
      ok = false
    }
    if (!ok) {
      ta.setRangeText(repl, off, off + kw.length, 'select')
      ta.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: repl }))
    }
    // sv 模式：textarea 值同步更新，立即重建匹配
    runFind()
  } else {
    const m = irMatches[matchIndex.value]
    if (!m || !m.node.isConnected) {
      // 引用已失效（Vditor 重渲染替换了节点），重建匹配后退出
      runFind()
      setTimeout(runFind, 110)
      return
    }
    // 验证偏移量仍有效：前几次替换或 Vditor 重渲染可能已改变文本节点内容/长度
    const text = m.node.nodeValue || ''
    if (m.end > text.length || text.substring(m.start, m.end).toLowerCase() !== kw.toLowerCase()) {
      // 偏移失效，重建匹配后退出
      runFind()
      setTimeout(runFind, 110)
      return
    }
    target.el.focus()
    const range = document.createRange()
    range.setStart(m.node, m.start)
    range.setEnd(m.node, m.end)
    range.deleteContents()
    range.insertNode(document.createTextNode(repl))
    m.node.parentElement?.normalize()
    target.el.dispatchEvent(
      new InputEvent('input', { bubbles: true, inputType: 'insertText', data: repl })
    )
    // ir 模式：Vditor 异步重渲染会替换文本节点，延迟重建匹配并定位到下一个
  }
  // 兜底：延迟重建匹配并定位到下一个（ir 等待 Vditor 重渲染完成）
  setTimeout(runFind, 110)
}

/** 全部替换（大小写不敏感） */
function replaceAll() {
  const kw = findKeyword.value
  const repl = replaceKeyword.value
  const target = getEditable()
  if (!target || !kw) return

  if (target.kind === 'sv') {
    const ta = target.el
    const next = ta.value.replace(new RegExp(escapeRegExp(kw), 'gi'), repl)
    if (next === ta.value) return
    ta.focus()
    ta.select()
    let ok = false
    try {
      ok = document.execCommand('insertText', false, next)
    } catch {
      ok = false
    }
    if (!ok) {
      ta.value = next
      ta.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: repl }))
    }
  } else {
    const all = collectIrMatches(target.el, kw.toLowerCase())
    if (!all.length) return
    target.el.focus()
    // 逆序替换保证偏移量不失效
    for (let i = all.length - 1; i >= 0; i--) {
      const m = all[i]
      if (!m.node.isConnected) continue
      const range = document.createRange()
      range.setStart(m.node, m.start)
      range.setEnd(m.node, m.end)
      range.deleteContents()
      range.insertNode(document.createTextNode(repl))
    }
    target.el.normalize()
    target.el.dispatchEvent(
      new InputEvent('input', { bubbles: true, inputType: 'insertReplacementText', data: repl })
    )
  }
  showEditorTip(i18n('editor.replacedAll'))
  // 重建匹配列表
  runFind()
  setTimeout(runFind, 200)
}

function toggleReplaceMode() {
  replaceMode.value = !replaceMode.value
  if (replaceMode.value) {
    nextTick(() => replaceInputRef.value?.focus())
  } else {
    nextTick(() => findInputRef.value?.focus())
  }
}

function getSelectionTextInEditor(): string {
  const target = getEditable()
  if (!target) return ''
  if (target.kind === 'sv') {
    return target.el.value.substring(target.el.selectionStart, target.el.selectionEnd)
  }
  return window.getSelection()?.toString() ?? ''
}

function openFind(withReplace = false) {
  findVisible.value = true
  replaceMode.value = withReplace
  if (!findKeyword.value) {
    const sel = getSelectionTextInEditor().trim()
    if (sel && sel.length <= 100 && !sel.includes('\n')) findKeyword.value = sel
  }
  nextTick(() => {
    const input = withReplace ? replaceInputRef.value : findInputRef.value
    input?.focus()
    if (!withReplace) findInputRef.value?.select()
    if (findKeyword.value) runFind()
  })
}

function openReplace() {
  openFind(true)
}

function closeFindBar() {
  findVisible.value = false
  replaceMode.value = false
  if (findDebounce) clearTimeout(findDebounce)
  // 清除残留选区高亮
  const target = getEditable()
  if (target?.kind === 'ir') {
    const sel = window.getSelection()
    if (sel && target.el.contains(sel.anchorNode)) sel.removeAllRanges()
  }
  svOffsets = []
  irMatches = []
  matchTotal.value = 0
  matchIndex.value = -1
}

watch(findKeyword, () => {
  if (!findVisible.value) return
  scheduleRunFind(findComposing.value ? 200 : 120)
})

// 查找浮层打开时，全局 keydown 监听 Enter/Esc
// （焦点在 textarea 上时，查找输入框的 @keydown 不触发）
function onFindGlobalKeydown(e: KeyboardEvent) {
  if (!findVisible.value) return
  const t = e.target as HTMLElement
  // 焦点在查找/替换输入框上时由它们自己的 @keydown 处理
  if (t === findInputRef.value || t === replaceInputRef.value) return
  if (e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) {
      gotoPrev()
    } else {
      gotoNext()
    }
  } else if (e.key === 'Escape') {
    e.preventDefault()
    closeFindBar()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onFindGlobalKeydown, true)
  if (!editorRef.value) return
  const initialMode = currentVditorMode()
  const dark = (props.theme ?? 'classic') === 'dark'

  // DOM 变化时：解析相对路径图片 + 同步源码模式行号槽
  let domTimer: ReturnType<typeof setTimeout> | null = null
  imgObserver = new MutationObserver(() => {
    if (domTimer) clearTimeout(domTimer)
    domTimer = setTimeout(() => {
      resolveImagesIn(editorRef.value, props.docPath)
      refreshSvGutter()
    }, 60)
  })
  imgObserver.observe(editorRef.value, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['src'],
  })

  vditor = new Vditor(editorRef.value, {
    cdn: '/vditor',
    mode: initialMode,
    theme: props.theme ?? 'classic',
    // Vditor 内置工具栏提示语言（初始化时确定，切换界面语言后需重新挂载才会变化）
    lang: editorLangCode() as 'zh_CN' | 'en_US',
    cache: { enable: false },
    height: '100%',
    placeholder: i18n('editor.placeholder'),
    toolbar: [
      'emoji', 'headings', 'bold', 'italic', 'strike', '|',
      'link', 'list', 'ordered-list', 'check', 'outdent', 'indent', '|',
      'quote', 'line', 'code', 'inline-code', 'insert-before', 'insert-after', '|',
      'table', 'upload', '|',
      'undo', 'redo', '|',
      // edit-mode 保留在配置中（应用级模式切换依赖其内部元素），通过 CSS 隐藏
      'edit-mode', 'outline', 'preview', 'fullscreen',
    ],
    upload: {
      accept: 'image/*',
      multiple: true,
      handler: handleImageUpload,
    },
    // 预览区工具条：仅保留一个「复制」按钮
    // （默认的 desktop / tablet / mobile / 微信 / 知乎 切换按钮已移除）
    preview: {
      actions: [
        {
          key: 'XinText-copy',
          text: i18n('editor.copy'),
          // __e：tooltip 显示在按钮右侧（按钮位于预览面板左边缘，
          // 气泡向右展开落在面板内，不会被 overflow:auto 裁切）
          className: 'vditor-tooltipped vditor-tooltipped__e',
          tooltip: i18n('editor.copyTip'),
          click: () => copyPreviewContent(),
        },
      ],
      // 正文渲染区主题（ir 正文 / sv 预览），暗色必须显式指定，
      // 否则内容区始终是 light 白底样式
      theme: {
        current: dark ? 'dark' : 'light',
        path: '/vditor/dist/css/content-theme',
      },
      hljs: {
        style: dark ? 'github-dark' : 'github',
      },
    },
    // 表情提示：内置 Unicode emoji 映射。
    // - value 不含 `.`：按文本（unicode 字符）渲染，直接显示 emoji
    // - 用户输入 `:` 后空匹配走此 map；输入 `:xxx` 走 lute 内置 emoji map
    hint: {
      emoji: EMOJI_MAP,
    },
    input(value: string) {
      if (suppressInput) return
      emit('update:modelValue', value)
    },
    after() {
      ready = true
      setEditorValue(props.modelValue)
      setTimeout(refreshSvGutter, 0)
      // 就绪时若已处于全屏（如编辑器在全屏态重建），下一帧移除目标控件
      if (props.zen) nextTick(detachZenElements)
      // ir 内容区编辑时，若查找栏开着则刷新匹配（只绑一次，ir 元素常驻）
      const irEl = (vditor?.vditor as any)?.ir?.element as HTMLElement | undefined
      if (irEl && !irInputBound) {
        irInputBound = true
        irEl.addEventListener('input', () => {
          if (findVisible.value) scheduleRunFind(200)
        })
      }
      // 链接点击拦截：命中本地 .md 用 ReadView 打开，命中 web 链接用 BrowserView 打开
      // editorRef 是 Vditor 的根 DOM，监听它在 capture 阶段拦截 ir/wysiwyg/sv 内的 <a>
      // markdownRef 用于从源码恢复 lute 渲染丢失的反斜杠（Windows 绝对路径写法）
      if (detachLinkInterception) detachLinkInterception()
      detachLinkInterception = mountLinkInterception(
        editorRef,
        docPathRef,
        {
          onOpenMd: (path) => emit('open-md', path),
          onOpenBrowser: (url) => emit('open-browser', url),
        },
        markdownRef
      )
      // 监听 Vditor 工具栏 undo/redo 按钮的 disabled 状态，
      // 同步到编辑菜单（按钮 disabled 时菜单项置灰）
      setupUndoRedoObserver()
    },
  })
})

// External content reload (e.g. opened a different file).
watch(
  () => props.modelValue,
  (value) => {
    setEditorValue(value)
  }
)

// Mode switch driven by the app-level "预览编辑 / 源码编辑" buttons.
watch(
  () => props.mode,
  () => {
    // 全屏中切换模式前先插回被移除的控件，让 Vditor 在完整 DOM 上重建，
    // 切换完成后再按新模式移除（下方 nextTick）
    if (props.zen) restoreZenElements()
    switchMode(currentVditorMode())
    // setEditMode 切到 sv 时会 .focus()，Chromium 因 selectionStart 在末尾而把
    // 滚动条拉到底部；立即重置光标到开头并滚回顶部
    resetSvScrollTop()
    // 模式切换后 textarea 挂载/延迟，稍后确保行号槽就位/隐藏
    setTimeout(refreshSvGutter, 80)
    // 查找栏开着时，切换 ir/sv 后在新内容区重建匹配
    if (findVisible.value) setTimeout(runFind, 120)
    if (props.zen) nextTick(detachZenElements)
  }
)

// —— 全屏编辑：临时移除 Vditor 内置控件 ——
// 用真实 DOM 移除/插回而非 display:none：Vditor 工具栏 flex 布局会对
// display:none 的项做重排，导致样式错乱；元素真实不存在时布局才正确。
interface DetachedNode {
  el: HTMLElement
  parent: HTMLElement
  nextSibling: ChildNode | null
}
let detachedZenNodes: DetachedNode[] = []

function detachZenElements() {
  const root = editorRef.value
  if (!root || detachedZenNodes.length > 0) return
  const pick = (selector: string, useItemWrapper: boolean) => {
    root.querySelectorAll<HTMLElement>(selector).forEach((node) => {
      const el = useItemWrapper
        ? (node.closest('.vditor-toolbar__item') as HTMLElement | null)
        : node
      if (!el || !el.parentElement) return
      detachedZenNodes.push({
        el,
        parent: el.parentElement,
        nextSibling: el.nextSibling,
      })
      el.remove()
    })
  }
  // 工具栏「全屏切换 Ctrl+'」按钮：两种模式都移除
  pick("[data-type='fullscreen']", true)
  if (props.mode === 'source') {
    // 源码模式：右侧预览面板顶部 action 操作条（自定义「复制」按钮）
    pick('.vditor-preview__action', false)
  } else {
    // 预览编辑模式：工具栏「预览」按钮
    pick("[data-type='preview']", true)
  }
}

function restoreZenElements() {
  const nodes = detachedZenNodes
  detachedZenNodes = []
  for (const { el, parent, nextSibling } of nodes) {
    // 锚点仍在原父节点下则原位插回，否则追加到末尾（父节点本身已失效则放弃）
    if (!parent.isConnected) continue
    if (nextSibling && nextSibling.parentNode === parent) {
      parent.insertBefore(el, nextSibling)
    } else {
      parent.appendChild(el)
    }
  }
}

watch(
  () => props.zen,
  (zen) => {
    if (!ready) return
    if (zen) nextTick(detachZenElements)
    else restoreZenElements()
  }
)

// Theme switch: 编辑器 chrome、正文渲染区、代码高亮三主题必须同步切换，
// 否则暗色下内容区仍是白底（或浅色文字落在浅色背景上）。
watch(
  () => props.theme,
  (theme) => {
    if (!vditor) return
    const t = theme ?? 'classic'
    const dark = t === 'dark'
    vditor.setTheme(t, dark ? 'dark' : 'light', dark ? 'github-dark' : 'github')
  }
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onFindGlobalKeydown, true)
  imgObserver?.disconnect()
  imgObserver = null
  undoRedoObserver?.disconnect()
  undoRedoObserver = null
  if (findDebounce) clearTimeout(findDebounce)
  if (detachLinkInterception) {
    detachLinkInterception()
    detachLinkInterception = null
  }
  editorRef.value?.querySelector('textarea.vditor-sv')?.['_svResizeObs']?.disconnect?.()
  vditor?.destroy()
  vditor = null
  ready = false
})

// 由 App 统一分发 Ctrl+F / Ctrl+H（菜单/快捷键）
// 段落/标题：通过模拟点击 Vditor toolbar 内置的 headings 子菜单按钮触发，
// 让 Vditor 内部统一处理 IR/WYSIWYG/SV 三种模式的 setHeading 逻辑。
// level=0 表示段落正文，需要临时改写 h1 按钮的 data-tag/data-value 触发段落处理
function setHeading(level: 0 | 1 | 2 | 3 | 4 | 5 | 6) {
  if (!vditor) return
  const element = (vditor.vditor as any)?.element as HTMLElement | undefined
  if (!element) return
  // 定位 headings 项的子菜单面板（vditor 初始化时已渲染 6 个 h1~h6 子按钮）
  const headingsBtn = element.querySelector('[data-type="headings"]') as HTMLElement | null
  const headingsItem = headingsBtn?.closest('.vditor-toolbar__item') as HTMLElement | null
  const panel = headingsItem?.querySelector('.vditor-hint') as HTMLElement | null
  if (!panel) return
  if (level === 0) {
    // 段落正文：临时改写 h1 按钮的 data-tag/data-value，click，再恢复
    // 触发 vditor 内部的 It(e, "") / W(e, "") / ge(e, "p")，对应清除标题前缀
    const h1Btn = panel.querySelector('button[data-tag="h1"]') as HTMLButtonElement | null
    if (!h1Btn) return
    const origTag = h1Btn.getAttribute('data-tag')
    const origValue = h1Btn.getAttribute('data-value')
    h1Btn.setAttribute('data-tag', 'p')
    h1Btn.setAttribute('data-value', '')
    h1Btn.click()
    if (origTag !== null) h1Btn.setAttribute('data-tag', origTag)
    if (origValue !== null) h1Btn.setAttribute('data-value', origValue)
  } else {
    const subBtn = panel.querySelector(`button[data-tag="h${level}"]`) as HTMLButtonElement | null
    subBtn?.click()
  }
}

/**
 * 插入菜单：通过 vditor.insertValue 直接插入对应 markdown 标记。
 *
 * 不再用模拟点击 toolbar 按钮的方式：Vditor 源码中 toolbar 项的事件
 * 绑定条件是 t.prefix（只有 bold/italic/strike 等带前缀的项才会绑定
 * click 监听），link/quote/code/table/line/inline-code 等无 prefix 的
 * 项事件不绑定在 button 自身，.click() 无效。
 *
 * 走 insertValue 兜底，让 Vditor 内部统一处理 IR/WYSIWYG/SV 三模式
 * 渲染。code/quote 等块级元素的 prefix 由 Vditor 内部处理。
 */
type InsertKind =
  | 'insert-before' | 'insert-after' | 'image' | 'link' | 'quote' | 'table'
  | 'line' | 'inline-code' | 'code' | 'footnote' | 'formula'

function insertBlock(kind: InsertKind) {
  if (!vditor) return
  // 从菜单按钮触发时，浏览器选区落在按钮上（不在编辑器内），
  // vditor.ir.range 也可能指向已脱离 DOM 的节点，导致 getEditorRange
  // 返回无效选区，insertMD 与 toolbar 按钮点击都插入失败。这里在
  // IR/WYSIWYG 模式下主动把选区设置到编辑器内，绕过浏览器 focus 后
  // 异步恢复选区的不确定行为。SV 模式走 textarea.selectionStart，
  // 不依赖 Range，无需处理。
  ensureEditorSelection()
  // 优先用 Vditor 实例的 toolbar 模拟点击触发内置逻辑（如带 prefix
  // 的 line/insert-before/insert-after），失败再走 insertMD 兜底
  const tried = tryTriggerMenu(kind)
  if (tried) return
  // 各插入项的 markdown 模板
  // 块级元素（图片/引用/表格/分割线/代码块/公式/脚注）前后加 \n 确保独占行；
  // 行内元素（链接/行内代码）不加换行，直接插在光标处。
  const templates: Record<InsertKind, string> = {
    'insert-before': '\n',
    'insert-after': '\n',
    'image': '\n![图片](https://)\n',
    'link': '[链接](https://)',
    'quote': '\n> 引用内容\n',
    'table': '\n| 列1 | 列2 |\n| --- | --- |\n|  |  |\n',
    'line': '\n\n---\n\n',
    'inline-code': '`code`',
    'code': '\n```\ncode\n```\n',
    'footnote': '\n[^1]: 脚注内容\n',
    'formula': '\n$$\nE = mc^2\n$$\n',
  }
  const md = templates[kind]
  // 用 insertMD 而非 insertValue：前者用 lute 把 markdown 转成
  // Vditor IR/WYSIWYG DOM 后插入，是正确的 markdown 插入方式；
  // 后者把字符串当 HTML 解析，无法识别 ```/[]() 等语法
  vditor.insertMD(md)
  // 插入后滚动到选区位置，确保用户能看到插入的内容
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0) {
    const r = sel.getRangeAt(0)
    let node: Node | null = r.startContainer
    while (node && !(node instanceof HTMLElement)) node = node.parentNode
    if (node instanceof HTMLElement) {
      try {
        node.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
      } catch {
        /* 忽略 */
      }
    }
  }
}

/**
 * 在 IR/WYSIWYG 模式下主动建立编辑器内选区。
 * 优先沿用 vditor[mode].range（用户上次在编辑器中的选区），
 * 仅当其仍落在当前 element 内时使用；否则折叠到 element 内容末尾。
 * 同步更新 vditor[mode].range 与浏览器选区，保证 getEditorRange 兜底
 * 逻辑一致，避免 insertHTML/insertMD 静默失败。
 */
function ensureEditorSelection() {
  if (!vditor) return
  const v = vditor.vditor as any
  const mode = v.currentMode
  if (mode !== 'ir' && mode !== 'wysiwyg') return
  const element = v[mode]?.element as HTMLElement | undefined
  if (!element) return
  const sel = window.getSelection()
  // 1. 优先沿用当前浏览器选区：用户刚在编辑器内点击/输入，选区就在光标处。
  //    此时不能覆写，否则会把光标从用户位置挪走（之前的 bug：键盘快捷键
  //    触发时 vditor.ir.range 已失效，旧代码用末尾选区覆盖了用户当前选区，
  //    导致内容插到文档末尾而非光标处）。
  if (sel && sel.rangeCount > 0) {
    const cur = sel.getRangeAt(0)
    if (element === cur.startContainer || element.contains(cur.startContainer)) {
      v[mode].range = cur
      element.focus()
      return
    }
  }
  // 2. 浏览器选区不在编辑器内（焦点在菜单按钮等），用 lastRange 兜底
  const lastRange = v[mode]?.range as Range | undefined
  let range: Range
  if (lastRange && element.contains(lastRange.startContainer)) {
    range = lastRange
  } else {
    // 3. lastRange 也失效，折叠到内容末尾
    range = document.createRange()
    range.selectNodeContents(element)
    range.collapse(false)
  }
  if (!sel) return
  sel.removeAllRanges()
  sel.addRange(range)
  v[mode].range = range
  element.focus()
}

/**
 * 尝试调用 Vditor 内部的 toolbar 触发逻辑。
 * Vditor 实例没有公开的 triggerMenu API，但可以通过模拟点击带 prefix
 * 的 toolbar 项触发：line(insert-before/insert-after) 有 prefix，IR 模式
 * 下 click 会进入 process_processToolbar 插入对应 markdown 前缀。
 * 无 prefix 的项（如 image/link）未绑定 click 监听，需走 insertMD 兜底。
 */
function tryTriggerMenu(kind: InsertKind): boolean {
  if (!vditor) return false
  // 仅 line/insert-before/insert-after 有 prefix，可走模拟点击
  if (kind !== 'insert-before' && kind !== 'insert-after' && kind !== 'line') return false
  const element = (vditor.vditor as any)?.element as HTMLElement | undefined
  if (!element) return false
  const btn = element.querySelector(`.vditor-toolbar [data-type="${kind}"]`) as HTMLElement | null
  if (!btn) return false
  btn.click()
  return true
}

/**
 * 插入 n*n 表格（来自表格菜单的 20x20 网格浮层）。
 * 按行列数动态构建 markdown 表格源文，复用 insertBlock 的选区
 * 修复逻辑（ensureEditorSelection）保证 IR/SV 模式都能正确插入。
 */
function insertTable(rows: number, cols: number) {
  if (!vditor) return
  const r = Math.max(1, Math.min(20, Math.floor(rows)))
  const c = Math.max(1, Math.min(20, Math.floor(cols)))
  // 表头行：| 列1 | 列2 | ... |
  const header = '| ' + Array.from({ length: c }, (_, i) => `列${i + 1}`).join(' | ') + ' |'
  // 分隔行：| --- | --- | ... |
  const sep = '| ' + Array.from({ length: c }, () => '---').join(' | ') + ' |'
  // 数据行：空单元格，r 行（表格至少 1 行数据，r 包含表头则数据行 r-1 行）
  const dataRows = Math.max(1, r - 1)
  const dataRow = '| ' + Array.from({ length: c }, () => '  ').join(' | ') + ' |'
  const md = '\n' + header + '\n' + sep + '\n' + Array.from({ length: dataRows }, () => dataRow).join('\n') + '\n'
  ensureEditorSelection()
  vditor.insertMD(md)
  // 滚动到插入点
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0) {
    const range = sel.getRangeAt(0)
    let node: Node | null = range.startContainer
    while (node && !(node instanceof HTMLElement)) node = node.parentNode
    if (node instanceof HTMLElement) {
      try { node.scrollIntoView({ block: 'nearest', behavior: 'smooth' }) } catch { /* 忽略 */ }
    }
  }
}

/**
 * 格式化：加粗/斜体/删除线/下划线/列表/缩进。
 * 大部分项通过模拟点击 Vditor toolbar 按钮触发内置逻辑
 * （bold/italic/strike/list/ordered-list/check/outdent/indent 都有
 * click 监听）。underline 不在 Vditor 内置 toolbar 中，用 <u> 标签
 * 通过 insertValue 插入 HTML 实现下划线效果。
 */
type FormatType = 'bold' | 'italic' | 'strike' | 'list' | 'ordered-list' | 'check' | 'outdent' | 'indent'

function format(type: FormatType) {
  if (!vditor) return
  ensureEditorSelection()
  // 模拟点击 Vditor toolbar 按钮
  const element = (vditor.vditor as any)?.element as HTMLElement | undefined
  if (!element) return
  const btn = element.querySelector(`.vditor-toolbar [data-type="${type}"]`) as HTMLElement | null
  if (btn) btn.click()
}

/** 撤销/重做：调用 Vditor 内部 undo 管理器 */
function undo() {
  if (!vditor) return
  const v = vditor.vditor as any
  v.undo?.undo?.(v)
}
function redo() {
  if (!vditor) return
  const v = vditor.vditor as any
  v.undo?.redo?.(v)
}

// 监听 Vditor 工具栏 undo/redo 按钮的 disabled class 变化，
// 同步状态到编辑菜单
let undoRedoObserver: MutationObserver | null = null
function setupUndoRedoObserver() {
  undoRedoObserver?.disconnect()
  undoRedoObserver = null
  const element = (vditor?.vditor as any)?.element as HTMLElement | undefined
  if (!element) return
  const undoBtn = element.querySelector('.vditor-toolbar [data-type="undo"]') as HTMLElement | null
  const redoBtn = element.querySelector('.vditor-toolbar [data-type="redo"]') as HTMLElement | null
  if (!undoBtn || !redoBtn) return
  const emitState = () => {
    emit('undo-redo-state',
      !undoBtn.classList.contains('vditor-menu--disabled'),
      !redoBtn.classList.contains('vditor-menu--disabled'))
  }
  emitState()
  undoRedoObserver = new MutationObserver(emitState)
  undoRedoObserver.observe(undoBtn, { attributes: true, attributeFilter: ['class'] })
  undoRedoObserver.observe(redoBtn, { attributes: true, attributeFilter: ['class'] })
}

/** 在光标处直接插入一段 Markdown 文本（由 App.vue 在目录文件等场景调用） */
function insertMarkdown(md: string) {
  if (!vditor) return
  ensureEditorSelection()
  vditor.insertMD(md)
}

defineExpose({ openFind, openReplace, setHeading, insertBlock, insertTable, insertMarkdown, format, undo, redo })
</script>

<style scoped>
.XinText-editor {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

/* 隐藏 Vditor 自带的模式切换按钮（模式切换由应用工具栏统一控制）。
   不能从 toolbar 配置中移除——应用级 switchMode 依赖其内部 DOM 元素。 */
:deep(.vditor-toolbar__item:has(> [data-type='edit-mode'])) {
  display: none;
}

/* 去除 vditor 根容器默认边框（.vditor 是组件根元素，不能用 :deep，
   :deep 会编译为后代选择器，匹配不到自身） */
.vditor {
  border: none;
}

/* 表情候选面板最大高度限制为 300px，超出滚动，避免长列表撑高面板 */
:deep(.vditor-emojis) {
  max-height: 300px;
}
:deep(.vditor-emojis button) {
  height: 40px;
  width: 40px;
}

/* 统一各模式（IR / WYSIWYG / SV 预览）的链接显示风格
   去掉默认下划线，仅用颜色区分；悬停时渐显下划线，交互更柔和 */
:deep(.vditor-ir a),
:deep(.vditor-wysiwyg a),
:deep(.vditor-preview a) {
  color: var(--app-active-text);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.15s ease, color 0.15s ease;
}

:deep(.vditor-ir a:hover),
:deep(.vditor-wysiwyg a:hover),
:deep(.vditor-preview a:hover) {
  border-bottom-color: var(--app-active-text);
}

/* 源码模式右侧预览顶部的操作区：按钮左对齐 + 与正文 1px 分隔线 */
:deep(.vditor-preview__action) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  padding: 6px 10px;
  text-align: left;
  background-color: var(--toolbar-background-color);
  border-bottom: 1px solid var(--border-color);
}

:deep(.vditor-preview__action button) {
  margin: 0;
  padding: 2px 12px;
  font-size: 12px;
  line-height: 20px;
  color: var(--toolbar-icon-color);
  background-color: var(--textarea-background-color);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  cursor: pointer;
  transition:
    color 0.15s ease,
    border-color 0.15s ease,
    background-color 0.15s ease;
}

:deep(.vditor-preview__action button:hover) {
  color: var(--toolbar-icon-hover-color);
  background-color: var(--textarea-background-color);
  border-color: var(--toolbar-icon-hover-color);
}

/* 复制按钮 tooltip（显示在按钮右侧）：在 Vditor 默认深色气泡基础上
   加大内边距/字号/圆角并加投影，箭头颜色随气泡底色保持一致 */
:deep(.vditor-preview__action .vditor-tooltipped::after) {
  padding: 6px 10px;
  font-size: 12px;
  line-height: 18px;
  border-radius: 4px;
  background: #3b3e43;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.18);
}

:deep(.vditor-preview__action .vditor-tooltipped__e::before) {
  border-right-color: #3b3e43;
}

/* 源码区 / 预览区可拖拽分隔条：1px 可见线 + 6px 透明命中区 */
:deep(.ot-sv-resizer) {
  position: relative;
  flex: 0 0 2px;
  width: 6px;
  cursor: col-resize;
  background: transparent;
  user-select: none;
}

:deep(.ot-sv-resizer::before) {
  content: '';
  position: absolute;
  top: 0;
  left: 1px;
  width: 1px;
  height: 100%;
  background: var(--border-color);
  transition: background 0.15s;
}

:deep(.ot-sv-resizer:hover::before),
:deep(.ot-sv-resizer.is-dragging::before) {
  background: #4285f4;
}

/* 分隔条已充当分隔线，去掉预览区自带的左边框，避免双线 */
:deep(.ot-sv-resizer + .vditor-preview) {
  border-left: none;
  margin-left: 0;
}

/* 源码模式行号槽（与 textarea.vditor-sv 并排，内含透明镜像文本对齐软折行） */
:deep(.XinText-sv-gutter) {
  position: relative;
  flex: 0 0 46px;
  width: 46px;
  align-self: stretch;
  overflow: hidden;
  background-color: var(--textarea-background-color);
  border-right: 1px solid var(--border-color);
  user-select: none;
}

:deep(.XinText-sv-gutter__inner) {
  position: absolute;
  top: 0;
  left: 0;
  box-sizing: content-box;
  padding: 10px 0;
  will-change: transform;
}

/* 每行：透明文本决定折行高度，行号绝对定位在该行首行 */
:deep(.sv-line) {
  position: relative;
  margin: 0;
  /* 空行时透明文本不生成行盒、父元素高度会塌陷为 0，
     导致绝对定位的行号与上下行重叠；min-height 保底一行高度 */
  min-height: 22px;
  white-space: pre-wrap;
  /* 与 textarea.vditor-sv 保持一致，长英文串/URL 的断行点才相同 */
  word-break: break-word;
  overflow-wrap: break-word;
  font-size: 16px;
  line-height: 22px;
  font-variant-ligatures: no-common-ligatures;
  font-family: "Helvetica Neue", "Luxi Sans", "DejaVu Sans", "Hiragino Sans GB",
    "Microsoft Yahei", sans-serif, "Apple Color Emoji", "Segoe UI Emoji",
    "Noto Color Emoji", "Segoe UI Symbol", "Android Emoji", "EmojiSymbols";
}

:deep(.sv-line__num) {
  position: absolute;
  top: 0;
  left: 0;
  width: 36px;
  text-align: right;
  color: var(--second-color);
}

:deep(.sv-line__text) {
  display: block;
  margin-left: 46px;
  color: transparent;
}

/* ===== 查找/替换浮层（定位上下文为 App 的 .XinText-app__editor） ===== */
.ot-fr {
  position: absolute;
  top: 10px;
  right: 12px;
  z-index: 30;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 6px;
  background: var(--app-bar-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.ot-fr__row {
  display: flex;
  align-items: center;
  gap: 2px;
}

.ot-fr__input {
  width: 170px;
  height: 24px;
  padding: 0 8px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 3px;
  outline: none;
  transition: border-color 0.15s ease;
}

.ot-fr__input:focus {
  border-color: var(--app-active-text);
}

.ot-fr__input.is-no-result {
  border-color: #e5484d;
}

.ot-fr__count {
  min-width: 42px;
  padding: 0 4px;
  font-size: 11px;
  text-align: center;
  color: var(--app-text-muted);
  user-select: none;
}

.ot-fr__count.is-none {
  color: #e5484d;
}

.ot-fr__btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  color: var(--app-text-muted);
  background: transparent;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition:
    color 0.15s ease,
    background-color 0.15s ease;
}

.ot-fr__btn:hover:not(:disabled) {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.ot-fr__btn:disabled {
  opacity: 0.35;
  cursor: default;
}

.ot-fr__btn--toggle.is-on {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.ot-fr__action {
  height: 24px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 3px;
  cursor: pointer;
  white-space: nowrap;
  transition:
    color 0.15s ease,
    border-color 0.15s ease,
    background-color 0.15s ease;
}

.ot-fr__action:hover:not(:disabled) {
  color: var(--app-active-text);
  border-color: var(--app-active-text);
}

.ot-fr__action:disabled {
  opacity: 0.4;
  cursor: default;
}
</style>
