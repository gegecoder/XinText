<template>
  <Teleport to="body">
    <div v-if="visible" class="search-overlay" @click.self="close">
      <div class="search-dialog">
        <header class="search-dialog__header">
          <span class="search-dialog__title">{{ i18n('search.title') }}</span>
          <button class="search-dialog__close" :title="i18n('common.closeEsc')" @click="close">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </header>

        <div class="search-dialog__bar">
          <svg class="search-dialog__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
          </svg>
          <input
            ref="inputRef"
            v-model="keyword"
            class="search-dialog__input"
            type="text"
            :placeholder="i18n('search.placeholder')"
            spellcheck="false"
            @keydown.enter.prevent="doSearch"
            @keydown.esc.prevent="close"
          />
          <div class="search-dialog__ext">
            <button
              v-for="opt in extOptions"
              :key="opt.value"
              class="ext-btn"
              :class="{ 'is-active': extFilter === opt.value }"
              @click="extFilter = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>

        <div class="search-dialog__body">
          <div v-if="searching" class="search-dialog__state">{{ i18n('search.searching') }}</div>
          <div v-else-if="searched && hits.length === 0" class="search-dialog__state">
            {{ i18n('search.noResult', { keyword: lastKeyword }) }}
          </div>
          <div v-else-if="!searched" class="search-dialog__state">
            {{ rootPath ? i18n('search.startHint') : i18n('search.selectFolderHint') }}
          </div>
          <ul v-else class="search-results">
            <li
              v-for="(hit, i) in hits"
              :key="hit.path + ':' + hit.lineNumber + ':' + i"
              class="search-result"
              :title="hit.path.replace(/\\/g, '/') + i18n('search.lineTitle', { n: hit.lineNumber })"
              @click="openHit(hit)"
            >
              <div class="search-result__main">
                <div class="search-result__title">
                  <span class="search-result__name">{{ hit.name }}</span>
                  <span class="search-result__line">{{ i18n('search.line', { n: hit.lineNumber }) }}</span>
                </div>
                <div class="search-result__snippet" v-html="highlight(hit)"></div>
              </div>
              <button class="search-result__open" :title="i18n('search.openFile')" @click.stop="openHit(hit)">
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M14 3h7v7" /><path d="M21 3l-9 9" /><path d="M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5" />
                </svg>
              </button>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, nextTick, watch } from 'vue'
import { SearchService } from '../../../bindings/XinText/internal/service'
import { i18n } from '../../i18n'

interface TextMatch {
  start: number
  end: number
}

interface SearchHit {
  path: string
  name: string
  lineNumber: number
  snippet: string
  matches?: TextMatch[] | null
}

const props = defineProps<{
  visible: boolean
  rootPath: string | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'open-file', path: string): void
}>()

// 用 computed 包裹，切换界面语言后「全部」标签即时刷新；md/txt 为扩展名不翻译
const extOptions = computed(() => [
  { value: 'all', label: i18n('search.all') },
  { value: 'md', label: 'md' },
  { value: 'txt', label: 'txt' },
] as const)

const inputRef = ref<HTMLInputElement | null>(null)
const keyword = ref('')
const extFilter = ref<'all' | 'md' | 'txt'>('all')
const searching = ref(false)
const searched = ref(false)
const lastKeyword = ref('')
const hits = ref<SearchHit[]>([])

watch(
  () => props.visible,
  async (v) => {
    if (v) {
      await nextTick()
      inputRef.value?.focus()
    } else {
      // 关闭时重置检索状态，下次打开是干净的
      searching.value = false
      searched.value = false
      hits.value = []
      keyword.value = ''
    }
  }
)

function close() {
  emit('update:visible', false)
}

async function doSearch() {
  const kw = keyword.value.trim()
  if (!kw || !props.rootPath || searching.value) return
  searching.value = true
  searched.value = false
  lastKeyword.value = kw
  try {
    hits.value = (await SearchService.SearchInDirectory(props.rootPath, kw, extFilter.value)) ?? []
  } catch (e) {
    console.error('fulltext search failed', e)
    hits.value = []
  } finally {
    searching.value = false
    searched.value = true
  }
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

/**
 * 转义片段并高亮命中位置：优先使用后端返回的精确区间（rune/code point
 * 坐标，已覆盖一行内多关键词的全部出现位置并合并重叠）；无区间数据时
 * 回退到按输入关键词自行定位。用码元数组遍历，保证中文与 emoji 不错位。
 */
function highlight(hit: SearchHit): string {
  const s = hit.snippet
  // 按 code point 展开，与后端 rune 下标一一对应（中文/emoji 均占 1 位）
  const chars = Array.from(s)
  const marks = new Array<boolean>(chars.length).fill(false)

  if (hit.matches && hit.matches.length > 0) {
    for (const m of hit.matches) {
      const end = Math.min(m.end, chars.length)
      for (let k = Math.max(m.start, 0); k < end; k++) marks[k] = true
    }
  } else {
    const words = lastKeyword.value.split(/\s+/).filter(Boolean)
    if (words.length === 0) return escapeHtml(s)
    const lower = chars.map((c) => c.toLowerCase())
    for (const w of words) {
      const wChars = Array.from(w.toLowerCase())
      outer: for (let i = 0; i + wChars.length <= chars.length; i++) {
        for (let j = 0; j < wChars.length; j++) {
          if (lower[i + j] !== wChars[j]) continue outer
        }
        for (let k = i; k < i + wChars.length; k++) marks[k] = true
      }
    }
  }

  let out = ''
  let inMark = false
  for (let i = 0; i < chars.length; i++) {
    if (marks[i] && !inMark) {
      out += '<mark>'
      inMark = true
    } else if (!marks[i] && inMark) {
      out += '</mark>'
      inMark = false
    }
    out += escapeHtml(chars[i])
  }
  if (inMark) out += '</mark>'
  return out
}

function openHit(hit: SearchHit) {
  emit('open-file', hit.path)
  close()
}
</script>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: search-fade 0.18s ease-out;
}

@keyframes search-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.search-dialog {
  display: flex;
  flex-direction: column;
  width: 640px;
  max-width: 90vw;
  height: 650px;
  max-height: 85vh;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: search-pop 0.2s ease-out;
  overflow: hidden;
}

@keyframes search-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.search-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 42px;
  padding: 0 14px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.search-dialog__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.search-dialog__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: none;
  border: none;
  border-radius: 4px;
  color: var(--app-text-muted);
  cursor: pointer;
}

.search-dialog__close:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

.search-dialog__bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.search-dialog__icon {
  color: var(--app-text-muted);
  flex-shrink: 0;
}

.search-dialog__input {
  flex: 1;
  min-width: 0;
  height: 30px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
  outline: none;
}

.search-dialog__input:focus {
  border-color: var(--app-active-text);
}

.search-dialog__ext {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.ext-btn {
  height: 30px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
  cursor: pointer;
}

.ext-btn.is-active {
  color: #fff;
  background: var(--app-active-text);
  border-color: var(--app-active-text);
}

.search-dialog__body {
  flex: 1;
  min-height: 160px;
  overflow-y: auto;
}

.search-dialog__state {
  padding: 40px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted);
}

.search-results {
  margin: 0;
  padding: 6px;
  list-style: none;
}

.search-result {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 52px;
  padding: 0 10px;
  border-radius: 8px;
  cursor: pointer;
}

.search-result:hover {
  background: var(--app-hover);
}

.search-result__main {
  flex: 1;
  min-width: 0;
}

.search-result__title {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 20px;
}

.search-result__name {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--app-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.search-result__line {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--app-text-muted);
}

.search-result__snippet {
  height: 20px;
  line-height: 20px;
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.search-result__snippet :deep(mark) {
  padding: 0 1px;
  color: inherit;
  background: rgba(255, 213, 79, 0.55);
  border-radius: 2px;
}

.search-result__open {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  color: var(--app-text-muted);
  background: none;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.12s, background 0.12s, color 0.12s;
}

.search-result:hover .search-result__open {
  opacity: 1;
}

.search-result__open:hover {
  color: var(--app-active-text);
  background: var(--app-hover);
}
</style>
