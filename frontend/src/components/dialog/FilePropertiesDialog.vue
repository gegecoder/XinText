<template>
  <Teleport to="body">
    <div v-if="visible" class="fp-overlay" @click.self="close">
      <div class="fp-dialog">
        <div class="fp-header">
          <span class="fp-title">{{ i18n('fileProps.title') }}</span>
          <button class="fp-close" :title="i18n('common.closeEsc')" @click="close">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="fp-body">
          <dl class="fp-list">
            <div class="fp-row">
              <dt>{{ i18n('fileProps.fileName') }}</dt>
              <dd class="fp-value">{{ info?.name ?? '—' }}</dd>
            </div>
            <div class="fp-row">
              <dt>{{ i18n('fileProps.path') }}</dt>
              <dd class="fp-value fp-value--path">{{ displayPath }}</dd>
            </div>
            <div class="fp-row">
              <dt>{{ i18n('fileProps.size') }}</dt>
              <dd class="fp-value">{{ sizeText }}</dd>
            </div>
            <div v-if="wordsText !== null" class="fp-row">
              <dt>{{ i18n('fileProps.words') }}</dt>
              <dd class="fp-value">{{ wordsText }}</dd>
            </div>
            <div v-if="charsText !== null" class="fp-row">
              <dt>{{ i18n('fileProps.chars') }}</dt>
              <dd class="fp-value">{{ charsText }}</dd>
            </div>
            <div class="fp-row">
              <dt>{{ i18n('fileProps.type') }}</dt>
              <dd class="fp-value">{{ info?.isDir ? i18n('fileProps.folder') : i18n('fileProps.file') }}</dd>
            </div>
            <div class="fp-row">
              <dt>{{ i18n('fileProps.modTime') }}</dt>
              <dd class="fp-value">{{ modTimeText }}</dd>
            </div>
            <div v-if="createTimeText !== null" class="fp-row">
              <dt>{{ i18n('fileProps.createTime') }}</dt>
              <dd class="fp-value">{{ createTimeText }}</dd>
            </div>
            <div class="fp-row">
              <dt>{{ i18n('fileProps.readOnly') }}</dt>
              <dd class="fp-value">{{ info?.readOnly ? i18n('fileProps.yes') : i18n('fileProps.no') }}</dd>
            </div>
          </dl>
        </div>

        <div class="fp-actions">
          <button class="fp-ok" @click="close">{{ i18n('common.ok') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { FileService } from '../../../bindings/XinText/internal/service'
import type { FileInfo } from '../../../bindings/XinText/internal/service/models'
import { i18n, dateLocaleCode } from '../../i18n'

const props = defineProps<{ visible: boolean; info: FileInfo | null }>()
const emit = defineEmits<{ 'update:visible': [v: boolean] }>()

// 统一显示为正斜杠，与 tab.path 风格一致
const displayPath = computed(() => props.info?.path ? props.info.path.replace(/\\/g, '/') : '—')

const sizeText = computed(() => {
  const size = props.info?.size ?? 0
  if (size < 1048576) return i18n('fileProps.sizeKb', { kb: (size / 1024).toFixed(1), bytes: String(size) })
  return i18n('fileProps.sizeMb', { mb: (size / 1048576).toFixed(2), bytes: String(size) })
})

// 字数 / 字符数：弹窗打开时异步读取文件内容统计（仅文本文件）
const wordCount = ref<number | null>(null)
const charCount = ref<number | null>(null)
const wordsText = computed(() =>
  wordCount.value === null ? null : wordCount.value.toLocaleString(dateLocaleCode())
)
const charsText = computed(() =>
  charCount.value === null ? null : charCount.value.toLocaleString(dateLocaleCode())
)

/** 中文按字、英文连续字母数字按词合并计数 */
function countWords(s: string): number {
  const m = s.match(/[\u4e00-\u9fa5]|[a-zA-Z0-9]+/g)
  return m ? m.length : 0
}

watch(
  () => props.info,
  async (info) => {
    wordCount.value = null
    charCount.value = null
    if (!info || info.isDir) return
    // 超过 10MB 的文件跳过统计，避免阻塞 UI
    if (info.size > 10 * 1024 * 1024) return
    try {
      const content = await FileService.ReadFile(info.path)
      charCount.value = content.length
      wordCount.value = countWords(content)
    } catch {
      // 二进制 / 不可读文件：静默忽略，不显示统计行
    }
  },
  { immediate: true }
)

const modTimeText = computed(() => {
  const modTime = props.info?.modTime
  if (!modTime) return '—'
  const time = new Date(modTime * 1000).toLocaleString(dateLocaleCode())
  const rel = formatRelativeTime(modTime)
  return rel ? i18n('fileProps.modTimeRel', { time, rel }) : time
})

const createTimeText = computed(() => {
  const ct = props.info?.createTime
  if (!ct) return null
  const time = new Date(ct * 1000).toLocaleString(dateLocaleCode())
  const rel = formatRelativeTime(ct)
  return rel ? i18n('fileProps.modTimeRel', { time, rel }) : time
})

/**
 * 把修改时间转为相对时间文案，如「1分钟前」「1周前」「1年2个月3天前」。
 * 单一级别用 Intl.RelativeTimeFormat（自动中英单复数），复合年月日按日历差拼接。
 */
function formatRelativeTime(modTime: number): string {
  const diffMs = Date.now() - modTime * 1000
  if (diffMs < 0) return ''
  const totalSec = Math.floor(diffMs / 1000)
  if (totalSec < 60) return i18n('fileProps.relJustNow')

  const locale = dateLocaleCode()
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'always' })
  const isZh = locale.startsWith('zh')

  const min = Math.floor(totalSec / 60)
  if (min < 60) return tidy(rtf.format(-min, 'minute'), isZh)
  const hr = Math.floor(totalSec / 3600)
  if (hr < 24) return tidy(rtf.format(-hr, 'hour'), isZh)
  const day = Math.floor(totalSec / 86400)
  if (day < 7) return tidy(rtf.format(-day, 'day'), isZh)
  if (day < 30) return tidy(rtf.format(-Math.floor(day / 7), 'week'), isZh)
  if (day < 365) return tidy(rtf.format(-Math.floor(day / 30), 'month'), isZh)

  // 复合：按日历算年/月/日差，省略为 0 的部分
  const from = new Date(modTime * 1000)
  const to = new Date()
  let years = to.getFullYear() - from.getFullYear()
  let months = to.getMonth() - from.getMonth()
  let days = to.getDate() - from.getDate()
  if (days < 0) {
    months--
    days += new Date(to.getFullYear(), to.getMonth(), 0).getDate()
  }
  if (months < 0) {
    years--
    months += 12
  }
  const joinSep = isZh ? '' : ' '
  const parts: string[] = []
  if (years > 0) parts.push(stripAgo(rtf.format(-years, 'year'), isZh))
  if (months > 0) parts.push(stripAgo(rtf.format(-months, 'month'), isZh))
  if (days > 0) parts.push(stripAgo(rtf.format(-days, 'day'), isZh))
  if (parts.length === 0) return i18n('fileProps.relJustNow')
  return parts.join(joinSep) + joinSep + i18n('fileProps.relAgo')
}

/** 中文去空格贴合示例格式（「1分钟前」而非「1 分钟前」） */
function tidy(s: string, isZh: boolean): string {
  return isZh ? s.replace(/\s+/g, '') : s
}

/** 复合拼接时去掉单级别末尾的「前 / ago」 */
function stripAgo(s: string, isZh: boolean): string {
  let r = s.replace(/\s*前$/, '').replace(/\s*ago$/i, '').trim()
  if (isZh) r = r.replace(/\s+/g, '')
  return r
}

function close() {
  emit('update:visible', false)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) close()
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.fp-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: fp-fade 0.18s ease-out;
}

@keyframes fp-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.fp-dialog {
  width: 480px;
  max-width: 90vw;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: fp-pop 0.2s ease-out;
  overflow: hidden;
}

@keyframes fp-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.fp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 12px;
  border-bottom: 1px solid var(--app-border, #e4e7ed);
}

.fp-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--app-text, #303133);
}

.fp-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  color: var(--app-text-muted, #909399);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.fp-close:hover {
  background: var(--app-hover, #f5f7fa);
  color: var(--app-text, #303133);
}

.fp-body {
  padding: 8px 20px 12px;
}

.fp-list {
  width: 100%;
  margin: 0;
  padding: 0;
  text-align: left;
}

.fp-row {
  display: flex;
  gap: 16px;
  padding: 8px 0;
  border-bottom: 1px solid var(--app-border, #f0f0f0);
  font-size: 13px;
  line-height: 1.6;
}

.fp-row:last-child {
  border-bottom: none;
}

.fp-row dt {
  flex-shrink: 0;
  width: 64px;
  margin: 0;
  color: var(--app-text-muted, #909399);
  font-weight: 600;
}

.fp-value {
  margin: 0;
  flex: 1;
  min-width: 0;
  color: var(--app-text, #303133);
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
  word-break: break-all;
}

.fp-value--path {
  line-height: 1.5;
}

.fp-actions {
  padding: 0 20px 16px;
  display: flex;
  justify-content: flex-end;
}

.fp-ok {
  min-width: 80px;
  height: 34px;
  font-size: 13px;
  color: #fff;
  background: var(--app-active-text, #409eff);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.fp-ok:hover {
  opacity: 0.9;
}
</style>
