<template>
  <div class="read-view">
    <!-- 左侧：文章内容（Vditor 静态渲染） -->
    <div ref="articleRef" class="read-view__article">
      <div ref="previewRef" class="read-view__preview"></div>
      <p v-if="!markdown?.trim()" class="read-view__empty">{{ i18n('readView.empty') }}</p>
    </div>

    <!-- 页内查找浮层（Ctrl+F）：定位在右上角，避开右侧目录栏 -->
    <div
      v-if="findVisible"
      class="read-view__findbar"
      :style="{ right: tocVisible ? '242px' : '12px' }"
    >
      <input
        ref="findInputRef"
        v-model="findKeyword"
        class="read-view__find-input"
        :class="{ 'is-no-result': isNoResult }"
        type="text"
        :placeholder="i18n('editor.find')"
        @keydown="onFindKeydown"
      >
      <span class="read-view__find-count" :class="{ 'is-none': isNoResult }">
        {{ matchTotal ? `${matchIndex + 1}/${matchTotal}` : '0/0' }}
      </span>
      <button
        class="read-view__find-btn"
        :title="i18n('editor.prev')"
        :disabled="!matchTotal"
        @click="gotoPrev"
      >
        <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
          <path
            d="M4 6.5l4-4 4 4M4 11.5l4-4 4 4"
            stroke="currentColor"
            stroke-width="1.3"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
        </svg>
      </button>
      <button
        class="read-view__find-btn"
        :title="i18n('editor.next')"
        :disabled="!matchTotal"
        @click="gotoNext"
      >
        <svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
          <path
            d="M4 4.5l4 4 4-4M4 9.5l4 4 4-4"
            stroke="currentColor"
            stroke-width="1.3"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
        </svg>
      </button>
      <button class="read-view__find-btn" :title="i18n('editor.close')" @click="closeFind">
        <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
          <path
            d="M4 4l8 8M12 4l-8 8"
            stroke="currentColor"
            stroke-width="1.3"
            stroke-linecap="round"
            fill="none"
          />
        </svg>
      </button>
    </div>

    <!-- 目录收起后的悬浮展开按钮（查找栏打开时下移避让） -->
    <button
      v-if="!tocVisible"
      class="read-view__expand"
      :style="{ top: findVisible ? '50px' : '10px' }"
      :title="i18n('readView.expandToc')"
      @click="tocVisible = true"
    >
      <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
        <path
          d="M2 3.5h1.5M6 3.5h8M2 8h1.5M6 8h8M2 12.5h1.5M6 12.5h8"
          stroke="currentColor"
          stroke-width="1.3"
          stroke-linecap="round"
          fill="none"
        />
      </svg>
      {{ i18n('readView.toc') }}
    </button>

    <!-- 右侧：目录 -->
    <aside v-show="tocVisible" ref="tocRef" class="read-view__toc">
      <div class="read-view__toc-head">
        <span class="read-view__toc-title">
          <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
            <path
              d="M2 3.5h1.5M6 3.5h8M2 8h1.5M6 8h8M2 12.5h1.5M6 12.5h8"
              stroke="currentColor"
              stroke-width="1.3"
              stroke-linecap="round"
              fill="none"
            />
          </svg>
          {{ i18n('readView.toc') }}
        </span>
        <button class="read-view__toc-collapse" :title="i18n('readView.collapseToc')" @click="tocVisible = false">
          <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
            <path
              d="M5 3l5 5-5 5M10 3l5 5-5 5"
              stroke="currentColor"
              stroke-width="1.3"
              stroke-linecap="round"
              stroke-linejoin="round"
              fill="none"
            />
          </svg>
        </button>
      </div>
      <nav class="read-view__toc-list">
        <a
          v-for="h in tocList"
          :key="h.id"
          :class="[
            'read-view__toc-item',
            'read-view__toc-item--lv' + h.level,
            { 'is-active': h.id === activeTocId },
          ]"
          :title="h.text"
          @click.prevent="scrollToHeading(h, $event)"
        >{{ h.text }}</a>
        <p v-if="!tocList.length" class="read-view__toc-empty">{{ i18n('readView.tocEmpty') }}</p>
      </nav>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, onBeforeUnmount } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { resolveImagesIn } from '../composables/resolveImages'
import { mountLinkInterception } from '../composables/useLinkInterception'
import { i18n } from '../i18n'

const props = defineProps<{
  markdown: string
  /** 'classic' (light) or 'dark'. */
  theme?: 'classic' | 'dark'
  /** Absolute path of the document, for resolving relative-path images. */
  docPath?: string | null
}>()

const emit = defineEmits<{
  /** 命中本地 .md 文件链接 */
  (e: 'open-md', path: string): void
  /** 命中 web 链接 */
  (e: 'open-browser', url: string): void
}>()

const articleRef = ref<HTMLDivElement | null>(null)
const previewRef = ref<HTMLDivElement | null>(null)
const docPathRef = computed(() => props.docPath ?? null)
// markdown 源码：用于在链接拦截时从原文恢复 lute 渲染丢失的反斜杠
const markdownRef = computed(() => props.markdown ?? '')
let detachLinkInterception: (() => void) | null = null
const tocRef = ref<HTMLElement | null>(null)

const tocVisible = ref(true)
const activeTocId = ref('')

interface TocItem {
  id: string
  level: number
  text: string
  el: HTMLElement
}
const tocList = ref<TocItem[]>([])

// 渲染序号：防止快速连续的内容/主题变化导致旧渲染回调覆盖新结果
let renderSeq = 0

/* ---------------- 页内查找（Ctrl+F） ---------------- */

const FIND_MARK_CLASS = 'ot-find-mark'

const findVisible = ref(false)
const findKeyword = ref('')
const findInputRef = ref<HTMLInputElement | null>(null)

// 当前匹配到的 <mark> 元素列表（非响应式，避免大量元素被代理）
let matchEls: HTMLElement[] = []
const matchTotal = ref(0)
const matchIndex = ref(-1) // 0-based；-1 表示无当前项
const isNoResult = computed(() => findKeyword.value.length > 0 && matchTotal.value === 0)

function openFind() {
  // 已打开时仅聚焦并全选（App 统一分发 Ctrl+F）
  if (findVisible.value) {
    nextTick(() => {
      findInputRef.value?.focus()
      findInputRef.value?.select()
    })
    return
  }
  findVisible.value = true
  // Chrome 行为：打开时若页面有选中文本则预填
  if (!findKeyword.value) {
    const sel = window.getSelection()?.toString().trim()
    if (sel && sel.length <= 100 && !sel.includes('\n')) {
      findKeyword.value = sel
    }
  }
  nextTick(() => {
    findInputRef.value?.focus()
    findInputRef.value?.select()
    if (findKeyword.value) runFind()
  })
}

function closeFind() {
  findVisible.value = false
  if (findTimer) clearTimeout(findTimer)
  clearMarks()
  findKeyword.value = ''
  matchEls = []
  matchTotal.value = 0
  matchIndex.value = -1
}

// 遍历渲染容器内所有文本节点，把关键词包裹成 <mark>。
// autoReveal=false 时仅建立高亮，不滚动（供重新渲染后恢复状态使用）。
function runFind(autoReveal = true) {
  clearMarks()
  matchEls = []
  matchTotal.value = 0
  matchIndex.value = -1

  const root = previewRef.value
  const kw = findKeyword.value
  if (!root || !kw) return

  const kwLower = kw.toLowerCase()
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = node.parentElement
      if (!parent || !node.nodeValue) return NodeFilter.FILTER_REJECT
      const tag = parent.tagName
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'NOSCRIPT') {
        return NodeFilter.FILTER_REJECT
      }
      return node.nodeValue.toLowerCase().includes(kwLower)
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_REJECT
    },
  })

  const targets: Text[] = []
  let current: Node | null
  while ((current = walker.nextNode())) {
    targets.push(current as Text)
  }

  for (const textNode of targets) {
    const text = textNode.nodeValue as string
    const lower = text.toLowerCase()
    const fragment = document.createDocumentFragment()
    let cursor = 0
    for (;;) {
      const idx = lower.indexOf(kwLower, cursor)
      if (idx === -1) {
        if (cursor < text.length) {
          fragment.appendChild(document.createTextNode(text.slice(cursor)))
        }
        break
      }
      if (idx > cursor) {
        fragment.appendChild(document.createTextNode(text.slice(cursor, idx)))
      }
      const mark = document.createElement('mark')
      mark.className = FIND_MARK_CLASS
      mark.textContent = text.slice(idx, idx + kw.length)
      fragment.appendChild(mark)
      matchEls.push(mark)
      cursor = idx + kw.length
    }
    textNode.parentNode?.replaceChild(fragment, textNode)
  }

  // 合并包裹后产生的相邻文本节点，保持 DOM 整洁
  root.normalize()

  matchTotal.value = matchEls.length
  if (matchEls.length) {
    matchIndex.value = 0
    if (autoReveal) revealMatch(false)
  }
}

// 解除所有 <mark> 包裹，恢复原文
function clearMarks() {
  const root = previewRef.value
  if (!root) return
  root.querySelectorAll(`mark.${FIND_MARK_CLASS}`).forEach((mark) => {
    const parent = mark.parentNode
    if (!parent) return
    while (mark.firstChild) parent.insertBefore(mark.firstChild, mark)
    parent.removeChild(mark)
  })
}

// 高亮当前匹配并滚动到文章容器中部
function revealMatch(smooth = true) {
  const el = matchEls[matchIndex.value]
  const container = articleRef.value
  if (!el || !container) return
  matchEls.forEach((m) => m.classList.remove('is-current'))
  el.classList.add('is-current')
  const cRect = container.getBoundingClientRect()
  const eRect = el.getBoundingClientRect()
  const target =
    eRect.top - cRect.top + container.scrollTop - container.clientHeight / 2 + eRect.height / 2
  container.scrollTo({ top: Math.max(0, target), behavior: smooth ? 'smooth' : 'auto' })
}

function gotoNext() {
  if (!matchEls.length) return
  matchIndex.value = (matchIndex.value + 1) % matchEls.length
  revealMatch()
}

function gotoPrev() {
  if (!matchEls.length) return
  matchIndex.value = (matchIndex.value - 1 + matchEls.length) % matchEls.length
  revealMatch()
}

function onFindKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) gotoPrev()
    else gotoNext()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    closeFind()
  }
}

// 输入即时检索（轻微防抖，避免大文档逐键全量遍历）
let findTimer: ReturnType<typeof setTimeout> | null = null
watch(findKeyword, () => {
  if (!findVisible.value) return
  if (findTimer) clearTimeout(findTimer)
  findTimer = setTimeout(() => {
    if (findVisible.value) runFind()
  }, 120)
})

async function renderPreview() {
  const el = previewRef.value
  if (!el) return
  const seq = ++renderSeq
  const dark = props.theme === 'dark'
  try {
    await Vditor.preview(el, props.markdown ?? '', {
      cdn: '/vditor',
      lang: 'zh_CN',
      mode: dark ? 'dark' : 'light',
      theme: { current: props.theme ?? 'classic' },
      hljs: { enable: true, style: dark ? 'github-dark' : 'github' },
    })
  } catch (e) {
    console.error('render preview failed', e)
    return
  }
  if (seq !== renderSeq) return
  // 渲染 Promise resolve 后内部 DOM 已就绪；再等双 rAF 确保布局稳定
  await new Promise((r) => requestAnimationFrame(r))
  await new Promise((r) => requestAnimationFrame(r))
  if (seq !== renderSeq) return
  // 相对路径图片（assets/xxx.png）→ data URL，WebView 才能显示
  resolveImagesIn(previewRef.value, props.docPath)
  buildToc()
  bindScroll()
  // 重新渲染会重建 DOM，查找框若开着需重新高亮，保持原索引（钳制到有效范围）
  if (findVisible.value && findKeyword.value) {
    const prevIndex = Math.max(matchIndex.value, 0)
    runFind(false)
    if (matchEls.length) {
      matchIndex.value = Math.min(prevIndex, matchEls.length - 1)
      revealMatch(false)
    }
  }
}

// 从渲染后的 DOM 中抽取 h1-h6 生成目录。
// 直接持有 DOM 引用并用 data 属性标记，避免重名 id 冲突。
function buildToc() {
  const root = previewRef.value
  if (!root) {
    tocList.value = []
    return
  }
  const headings = Array.from(root.querySelectorAll<HTMLElement>('h1, h2, h3, h4, h5, h6'))
  tocList.value = headings.map((el, i) => {
    const text = (el.textContent || '').trim()
    const id = `toc-h-${i}`
    el.setAttribute('data-toc-id', id)
    return { id, level: Number(el.tagName[1]), text, el }
  })
  activeTocId.value = tocList.value.length ? tocList.value[0].id : ''
}

// 点击目录项 → 滚动到对应标题。
// 加滚动锁，避免 smooth 滚动未结束时 scroll 事件回写高亮。
let scrollLock = false
let scrollLockTimer: ReturnType<typeof setTimeout> | null = null

function scrollToHeading(item: TocItem, e?: MouseEvent) {
  const el = item?.el
  const container = articleRef.value
  if (!el || !container) return
  scrollLock = true
  activeTocId.value = item.id
  // 用 rect 差值 + scrollTop 计算滚动目标，不依赖 offsetParent 链
  const cRect = container.getBoundingClientRect()
  const eRect = el.getBoundingClientRect()
  const offset = 20
  const targetTop = eRect.top - cRect.top + container.scrollTop - offset
  container.scrollTo({ top: Math.max(0, targetTop), behavior: 'smooth' })
  if (scrollLockTimer) clearTimeout(scrollLockTimer)
  const unlock = () => {
    scrollLock = false
    container.removeEventListener('scrollend', unlock)
  }
  container.addEventListener('scrollend', unlock, { once: true })
  // 兜底：3s 后强制解锁（部分内核不支持 scrollend）
  scrollLockTimer = setTimeout(unlock, 3000)
  // 目录项自身在目录区内滚动到可见
  const target = e?.currentTarget as HTMLElement | null
  target?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}

// 监听文章滚动，自动高亮当前章节（已滚出顶部的最后一个标题）
function handleArticleScroll() {
  if (scrollLock) return
  const container = articleRef.value
  if (!container || !tocList.value.length) return
  const cRect = container.getBoundingClientRect()
  const offset = 20
  let current = tocList.value[0].id
  for (const h of tocList.value) {
    if (!h.el) continue
    const top = h.el.getBoundingClientRect().top - cRect.top
    if (top <= offset) {
      current = h.id
    } else {
      break
    }
  }
  if (activeTocId.value !== current) {
    activeTocId.value = current
  }
}

function bindScroll() {
  const container = articleRef.value
  if (!container) return
  container.removeEventListener('scroll', handleArticleScroll)
  container.addEventListener('scroll', handleArticleScroll, { passive: true })
}

// 高亮变化时，让 active 目录项在目录区内滚动到可见（不干扰文章滚动）
watch(activeTocId, () => {
  const sidebar = tocRef.value
  if (!sidebar) return
  const active = sidebar.querySelector('.read-view__toc-item.is-active')
  active?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
})

watch(
  () => [props.markdown, props.theme] as const,
  () => {
    renderPreview()
  }
)

onMounted(() => {
  renderPreview()
  // 链接点击拦截：previewRef 是稳定的容器元素，capture 阶段监听
  // 不会因 Vditor.preview 重建内部 DOM 而失效（事件冒泡基于父节点）。
  // markdownRef 用于从源码恢复 lute 渲染丢失的反斜杠（Windows 绝对路径写法）
  detachLinkInterception = mountLinkInterception(
    previewRef,
    docPathRef,
    {
      onOpenMd: (path) => emit('open-md', path),
      onOpenBrowser: (url) => emit('open-browser', url),
    },
    markdownRef
  )
})

onBeforeUnmount(() => {
  articleRef.value?.removeEventListener('scroll', handleArticleScroll)
  if (findTimer) clearTimeout(findTimer)
  if (scrollLockTimer) clearTimeout(scrollLockTimer)
  if (detachLinkInterception) {
    detachLinkInterception()
    detachLinkInterception = null
  }
})

// 由 App 统一分发 Ctrl+F（菜单/快捷键）
defineExpose({ openFind })
</script>

<style scoped>
.read-view {
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: var(--app-bg);
}

.read-view__article {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  /* 全屏时为左上角悬浮缩放控件预留顶部空间（该容器不被 zoom 影响） */
  padding-top: var(--zen-top-pad, 0px);
}

.read-view__preview {
  /* 全屏阅读宽度规则（z=--zen-zoom，g=--zen-grow=max(z-1, 0)）：
     max-width(border-box) = 860/z·(1-g) + (80% + 80px)·g
     · 100%（z=1,g=0）：860px，与常规阅读一致
     · 200%（z=2,g=1）：80%+80px——zoom:z 元素的百分比参考宽 = 父宽/z，
       80% 经 zoom 后视觉为阅读区 80%；+80px 补偿 border-box 两侧 padding
       （CSS 40px×2），使「文字内容区」视觉宽度恰好为 80%
     · 中间档位线性过渡；低于 100%（g=0）视觉行宽保持 860px */
  max-width: calc(
    860px / var(--zen-zoom, 1) * (1 - var(--zen-grow, 0))
    + (80% + 80px) * var(--zen-grow, 0)
  );
  margin: 0 auto;
  padding: 24px 40px 64px;
  /* 全屏阅读缩放：由 App.vue 通过 --zen-zoom 注入（50%~200% 离散档位）；
     非全屏时变量缺省为 1，不影响常规视图。用 zoom 而非 transform：
     文字排版随宽度重排，避免 transform 缩放产生的横向留白与滚动条 */
  zoom: var(--zen-zoom, 1);
}

.read-view__empty {
  margin: 0;
  padding: 40px 0;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted);
}

.read-view__expand {
  position: absolute;
  top: 10px;
  right: 12px;
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-bar-bg);
  border: 1px solid var(--app-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.read-view__expand:hover {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.read-view__toc {
  display: flex;
  flex-direction: column;
  width: 230px;
  flex-shrink: 0;
  overflow: hidden;
  border-left: 1px solid var(--app-border);
  background: var(--app-bar-bg);
}

.read-view__toc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.read-view__toc-title {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
}

.read-view__toc-collapse {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  color: var(--app-text-muted);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.read-view__toc-collapse:hover {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.read-view__toc-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  /* flex 子项默认 min-height:auto，目录过多时会被内容撑开而不滚动，
     必须显式置 0，overflow-y:auto 才能在固定高度内生效 */
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 8px;
}

.read-view__toc-item {
  /* flex 纵向滚动列表：条目必须禁止压缩，保持自然高度，
     否则目录过多时各项会被 flex-shrink 压扁（视觉上"挤一块"），
     压缩后总高不超过容器又导致滚动条不出现 */
  flex-shrink: 0;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--app-text);
  border-radius: 4px;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}

.read-view__toc-item:hover {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.read-view__toc-item.is-active {
  color: var(--app-active-text);
  background: var(--app-hover);
  font-weight: 600;
}

.read-view__toc-item--lv1 {
  font-size: 13px;
  font-weight: 600;
}

.read-view__toc-item--lv2 {
  padding-left: 20px;
}

.read-view__toc-item--lv3 {
  padding-left: 32px;
  font-size: 11px;
}

.read-view__toc-item--lv4 {
  padding-left: 44px;
  font-size: 11px;
  color: var(--app-text-muted);
}

.read-view__toc-item--lv5 {
  padding-left: 56px;
  font-size: 11px;
  color: var(--app-text-muted);
}

.read-view__toc-item--lv6 {
  padding-left: 68px;
  font-size: 11px;
  color: var(--app-text-muted);
}

.read-view__toc-empty {
  margin: 0;
  padding: 16px 0;
  font-size: 12px;
  color: var(--app-text-muted);
  text-align: center;
}

/* ---------- 页内查找浮层（Ctrl+F） ---------- */

.read-view__findbar {
  position: absolute;
  top: 10px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 6px;
  background: var(--app-bar-bg);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  transition: right 0.15s ease;
}

.read-view__find-input {
  width: 150px;
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

.read-view__find-input:focus {
  border-color: var(--app-active-text);
}

.read-view__find-input.is-no-result {
  border-color: #e5484d;
}

.read-view__find-count {
  min-width: 42px;
  padding: 0 4px;
  font-size: 11px;
  text-align: center;
  color: var(--app-text-muted);
  user-select: none;
}

.read-view__find-count.is-none {
  color: #e5484d;
}

.read-view__find-btn {
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

.read-view__find-btn:hover:not(:disabled) {
  color: var(--app-active-text);
  background: var(--app-hover);
}

.read-view__find-btn:disabled {
  opacity: 0.35;
  cursor: default;
}

/* 关键词高亮（<mark> 为动态插入的 DOM，需用 :deep 命中）。
   统一深色文字，保证在亮/暗主题的黄/橙底色上都清晰。 */
.read-view__preview :deep(mark.ot-find-mark) {
  padding: 0 1px;
  color: #1f2328;
  background-color: #ffe066;
  border-radius: 2px;
}

.read-view__preview :deep(mark.ot-find-mark.is-current) {
  background-color: #ff9800;
}

/* 统一阅读模式链接风格（与编辑器 IR/WYSIWYG/SV 预览一致） */
.read-view__preview :deep(a) {
  color: var(--app-active-text);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.15s ease, color 0.15s ease;
}

.read-view__preview :deep(a:hover) {
  border-bottom-color: var(--app-active-text);
}
</style>
