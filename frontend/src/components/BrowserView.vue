<template>
  <div class="browser-view">
    <!-- Chrome 风格工具栏：后退 / 前进 / 重载 / 地址栏 / 主页 -->
    <div class="browser-view__toolbar">
      <div class="browser-view__nav">
        <button
          class="browser-view__btn"
          :class="{ 'is-disabled': !canBack }"
          :title="i18n('browser.back')"
          :disabled="!canBack"
          @click="goBack"
        >
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path d="M20 12H4M10 6l-6 6 6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
        <button
          class="browser-view__btn"
          :class="{ 'is-disabled': !canForward }"
          :title="i18n('browser.forward')"
          :disabled="!canForward"
          @click="goForward"
        >
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path d="M4 12h16M14 6l6 6-6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
        <button
          class="browser-view__btn"
          :title="i18n('browser.reload')"
          @click="reload"
        >
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true" :class="{ 'is-loading': loading }">
            <path d="M4 4v6h6M20 20v-6h-6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
            <path d="M20 9a8 8 0 0 0-14.5-2M4 15a8 8 0 0 0 14.5 2" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
        <button
          class="browser-view__btn"
          :title="i18n('browser.goHome')"
          @click="goHome"
        >
          <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
            <path d="M3 11l9-8 9 8M5 9.5V20h5v-6h4v6h5V9.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
      </div>

      <div class="browser-view__address-wrap">
        <span class="browser-view__lock" :title="isHttps ? 'Secure' : ''">
          <svg v-if="isHttps" viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <path d="M6 10V8a6 6 0 0 1 12 0v2" stroke="currentColor" stroke-width="2" stroke-linecap="round" fill="none" />
            <rect x="5" y="10" width="14" height="10" rx="2" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
          <svg v-else viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
            <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2" fill="none" />
            <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        </span>
        <input
          ref="addressRef"
          v-model="addressInput"
          class="browser-view__address"
          type="text"
          :placeholder="i18n('browser.addressPlaceholder')"
          spellcheck="false"
          @focus="onAddressFocus"
          @blur="onAddressBlur"
          @keydown.enter.prevent="navigateFromAddress"
          @keydown.esc="onAddressEsc"
        >
        <button
          v-if="!addressFocused"
          class="browser-view__btn browser-view__external"
          :title="i18n('browser.openExternal')"
          @click="openInSystemBrowser"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
            <path d="M14 5h5v5M19 5l-9 9M18 14v5H5V6h5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
      </div>
    </div>

    <!-- iframe 容器：失败时显示遮罩引导用户在外部浏览器打开 -->
    <div class="browser-view__content">
      <iframe
        v-if="iframeSrc && !isNotFound"
        ref="iframeRef"
        :src="iframeSrc"
        class="browser-view__iframe"
        frameborder="0"
        allow="autoplay; encrypted-media; fullscreen; picture-in-picture; clipboard-read; clipboard-write"
        :sandbox="sandbox"
        referrerpolicy="no-referrer-when-downgrade"
        allowfullscreen
        @load="onIframeLoad"
        @error="onIframeError"
      ></iframe>

      <!-- 404 兜底界面：链接既非本地 .md 也非标准 web 链接 -->
      <div v-if="isNotFound" class="browser-view__notfound">
        <div class="browser-view__notfound-code">404</div>
        <p class="browser-view__notfound-title">{{ i18n('browser.notFoundTitle') }}</p>
        <p class="browser-view__notfound-desc">{{ i18n('browser.notFoundDesc') }}</p>
        <p class="browser-view__notfound-href" :title="notFoundHref">{{ notFoundHref }}</p>
        <div class="browser-view__notfound-actions">
          <button class="browser-view__notfound-btn" @click="goHome">{{ i18n('browser.goHome') }}</button>
          <button class="browser-view__notfound-btn browser-view__notfound-btn--ghost" @click="closeTab">{{ i18n('browser.closeTab') }}</button>
        </div>
      </div>

      <div v-if="loadError" class="browser-view__error">
        <svg viewBox="0 0 24 24" width="48" height="48" aria-hidden="true">
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none" />
          <path d="M12 8v5M12 16h.01" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        </svg>
        <p class="browser-view__error-text">{{ i18n('browser.loadError') }}</p>
        <p class="browser-view__error-url">{{ currentUrl }}</p>
        <button class="browser-view__error-btn" @click="openInSystemBrowser">
          {{ i18n('browser.openExternalAction') }}
        </button>
      </div>

      <div v-if="loading && !loadError" class="browser-view__loading-bar"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Browser } from '@wailsio/runtime'
import { i18n } from '../i18n'
import { useEditorStore } from '../store/editor'
import { usePreferencesStore } from '../store/preferences'

const props = defineProps<{
  tabId: string
  url: string
}>()

const editorStore = useEditorStore()
const prefsStore = usePreferencesStore()

// 浏览器主页 URL：用户在设置中配置，空则回退到内置默认
const homeURL = computed(() => prefsStore.config.browserHomeURL || i18n('browser.homeUrl'))

const addressRef = ref<HTMLInputElement | null>(null)
const iframeRef = ref<HTMLIFrameElement | null>(null)

const addressInput = ref(props.url)
const addressFocused = ref(false)
const loading = ref(false)
const loadError = ref(false)
const iframeSrc = ref(props.url)

// 404 兜底：XinText:notfound?href=<原 href> 协议，不加载 iframe，显示自定义友好界面
const NOT_FOUND_RE = /^XinText:notfound\?href=(.+)$/i
const isNotFound = computed(() => NOT_FOUND_RE.test(props.url))
const notFoundHref = computed(() => {
  const m = props.url.match(NOT_FOUND_RE)
  if (!m) return ''
  try { return decodeURIComponent(m[1]) } catch { return m[1] }
})

// 历史指针来自 store 中的 tab.history / tab.historyIndex
const tab = computed(() => editorStore.tabs.find((t) => t.id === props.tabId))
const history = computed<string[]>(() => tab.value?.history ?? [props.url])
const historyIndex = computed<number>(() => tab.value?.historyIndex ?? 0)
const canBack = computed(() => historyIndex.value > 0)
const canForward = computed(() => historyIndex.value < history.value.length - 1)
const currentUrl = computed(() => history.value[historyIndex.value] ?? props.url)
const isHttps = computed(() => /^https:\/\//i.test(currentUrl.value))

// 允许 iframe 内部脚本与同源，否则多数现代站点无法正常渲染。
// 注意：allow-same-origin + allow-scripts 同开在沙箱规范中是高危组合，
// 但桌面端浏览器视图本就以"信任用户主动访问的页面"为前提。
const sandbox = 'allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-modals allow-presentation allow-downloads'

let loadTimer: number | null = null
let unwatchUrl: (() => void) | null = null

function clearLoadTimer() {
  if (loadTimer !== null) {
    clearTimeout(loadTimer)
    loadTimer = null
  }
}

function startLoad() {
  loading.value = true
  loadError.value = false
  clearLoadTimer()
  // iframe 跨域加载失败时不会触发 error 事件，超时则提示用户。
  loadTimer = window.setTimeout(() => {
    if (loading.value) {
      loading.value = false
      loadError.value = true
    }
  }, 12000)
}

function onIframeLoad() {
  clearLoadTimer()
  loading.value = false
  // 加载成功后尝试读取 iframe 当前 URL（跨域时会抛错，忽略）
  try {
    const u = iframeRef.value?.contentWindow?.location?.href
    if (u && u !== 'about:blank' && u !== currentUrl.value) {
      // iframe 内部跳转：同步到 store 历史
      editorStore.updateBrowserUrl(props.tabId, u, true)
      addressInput.value = u
    }
  } catch {
    // 跨域：保持原地址栏，不报错
  }
}

function onIframeError() {
  clearLoadTimer()
  loading.value = false
  loadError.value = true
}

function goBack() {
  if (!canBack.value) return
  const target = editorStore.browserGoBack(props.tabId)
  if (target) loadIframe(target)
}

function goForward() {
  if (!canForward.value) return
  const target = editorStore.browserGoForward(props.tabId)
  if (target) loadIframe(target)
}

function reload() {
  if (!iframeRef.value) return
  // 通过重新赋值 src 强制重载
  const src = iframeSrc.value
  iframeSrc.value = ''
  nextTick(() => {
    iframeSrc.value = src
    startLoad()
  })
}

function goHome() {
  navigateTo(homeURL.value)
}

function navigateTo(url: string, pushHistory = true) {
  loadError.value = false
  if (pushHistory) {
    editorStore.updateBrowserUrl(props.tabId, url, true)
  }
  loadIframe(url)
}

function loadIframe(url: string) {
  // 404 兜底：不加载 iframe，地址栏显示原始 href（更友好），不触发 loading
  if (NOT_FOUND_RE.test(url)) {
    iframeSrc.value = 'about:blank'
    addressInput.value = notFoundHref.value || url
    loading.value = false
    loadError.value = false
    return
  }
  iframeSrc.value = url
  addressInput.value = url
  startLoad()
}

function navigateFromAddress() {
  let v = addressInput.value.trim()
  if (!v) return
  // 自动补协议
  if (!/^[a-z]+:\/\//i.test(v) && !/^about:/i.test(v)) {
    // 看起来像域名则补 https://，否则当搜索
    if (/^[\w-]+(\.[\w-]+)+/.test(v)) {
      v = 'https://' + v
    } else {
      v = homeURL.value.replace(/\/$/, '') + '/search?q=' + encodeURIComponent(v)
    }
  }
  addressInput.value = v
  navigateTo(v, true)
  addressRef.value?.blur()
}

function onAddressFocus() {
  addressFocused.value = true
  nextTick(() => addressRef.value?.select())
}

function onAddressBlur() {
  addressFocused.value = false
  // 失焦时把地址栏恢复为当前 URL，避免显示半截输入
  addressInput.value = currentUrl.value
}

function onAddressEsc() {
  addressInput.value = currentUrl.value
  addressRef.value?.blur()
}

async function openInSystemBrowser() {
  try {
    await Browser.OpenURL(currentUrl.value)
  } catch {
    window.open(currentUrl.value, '_blank')
  }
}

/** 关闭当前浏览器标签页 */
function closeTab() {
  editorStore.closeTab(props.tabId)
}

// 外部 URL 变更（如切换到该 tab、Tabs 点击同一 url）时同步
unwatchUrl = watch(
  () => props.url,
  (val) => {
    if (!val) return
    // 统一走 loadIframe，让 404 兜底 URL 也能被识别
    if (val !== iframeSrc.value || NOT_FOUND_RE.test(val)) {
      loadIframe(val)
    } else {
      addressInput.value = val
    }
  }
)

onMounted(() => {
  // 初始 URL 可能是 404 兜底，统一走 loadIframe 处理
  loadIframe(props.url)
})

onUnmounted(() => {
  clearLoadTimer()
  if (unwatchUrl) unwatchUrl()
})
</script>

<style scoped>
.browser-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  background: var(--app-bg);
  overflow: hidden;
}

.browser-view__toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 36px;
  padding: 0 8px;
  background: var(--app-bar-bg);
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.browser-view__nav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.browser-view__btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--app-text);
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.12s ease;
  flex-shrink: 0;
}

.browser-view__btn:hover:not(.is-disabled):not(:disabled) {
  background: var(--app-hover);
}

.browser-view__btn.is-disabled,
.browser-view__btn:disabled {
  color: var(--app-text-muted);
  cursor: not-allowed;
  opacity: 0.6;
}

.browser-view__btn .is-loading {
  animation: browser-spin 0.9s linear infinite;
}

@keyframes browser-spin {
  to { transform: rotate(360deg); }
}

.browser-view__address-wrap {
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  min-width: 0;
  height: 30px;
  background: var(--app-bg);
  border: 1px solid var(--app-border);
  border-radius: 16px;
  padding: 0 8px;
  margin: 0 6px;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.browser-view__address-wrap:focus-within {
  border-color: var(--app-active-text);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-active-text) 20%, transparent);
}

.browser-view__lock {
  display: inline-flex;
  align-items: center;
  color: var(--app-text-muted);
  flex-shrink: 0;
  margin-right: 6px;
}

.browser-view__address {
  flex: 1 1 auto;
  border: none;
  outline: none;
  background: transparent;
  color: var(--app-text);
  font-size: 13px;
  min-width: 0;
  height: 100%;
  padding: 0;
}

.browser-view__address::placeholder {
  color: var(--app-text-muted);
}

.browser-view__external {
  flex-shrink: 0;
  margin-left: 4px;
}

.browser-view__content {
  flex: 1 1 auto;
  position: relative;
  background: #fff;
  overflow: hidden;
}

.browser-view__iframe {
  width: 100%;
  height: 100%;
  border: none;
  display: block;
  background: #fff;
}

.browser-view__loading-bar {
  position: absolute;
  top: 0;
  left: 0;
  height: 3px;
  width: 40%;
  background: var(--app-active-text);
  border-radius: 2px;
  animation: browser-loading 1.1s ease-in-out infinite;
  box-shadow: 0 0 4px var(--app-active-text);
}

@keyframes browser-loading {
  0% { left: -40%; }
  100% { left: 100%; }
}

.browser-view__error {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--app-bg);
  color: var(--app-text-muted);
  text-align: center;
  padding: 24px;
}

.browser-view__error-text {
  font-size: 15px;
  color: var(--app-text);
  margin: 0;
}

.browser-view__error-url {
  font-size: 12px;
  color: var(--app-text-muted);
  margin: 0 0 8px;
  word-break: break-all;
}

.browser-view__error-btn {
  padding: 6px 14px;
  border: 1px solid var(--app-active-text);
  background: transparent;
  color: var(--app-active-text);
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
}

.browser-view__error-btn:hover {
  background: var(--app-active-text);
  color: #fff;
}

/* 404 兜底界面 */
.browser-view__notfound {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: var(--app-bg);
  color: var(--app-text-muted);
  text-align: center;
  padding: 24px;
}

.browser-view__notfound-code {
  font-size: 84px;
  font-weight: 600;
  line-height: 1;
  color: var(--app-active-text);
  letter-spacing: -2px;
  margin-bottom: 4px;
}

.browser-view__notfound-title {
  font-size: 16px;
  color: var(--app-text);
  margin: 0;
}

.browser-view__notfound-desc {
  font-size: 13px;
  margin: 0;
}

.browser-view__notfound-href {
  font-size: 12px;
  color: var(--app-text-muted);
  margin: 4px 0 12px;
  word-break: break-all;
  max-width: 80%;
  font-family: var(--app-mono, ui-monospace, SFMono-Regular, monospace);
  padding: 4px 10px;
  background: var(--app-hover);
  border-radius: 4px;
}

.browser-view__notfound-actions {
  display: flex;
  gap: 10px;
}

.browser-view__notfound-btn {
  padding: 6px 16px;
  border: 1px solid var(--app-active-text);
  background: var(--app-active-text);
  color: #fff;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.12s ease;
}

.browser-view__notfound-btn:hover {
  opacity: 0.85;
}

.browser-view__notfound-btn--ghost {
  background: transparent;
  color: var(--app-text);
  border-color: var(--app-border);
}

.browser-view__notfound-btn--ghost:hover {
  background: var(--app-hover);
  opacity: 1;
}

/* 暗色模式下 iframe 区域保持白色背景是浏览器惯例，
   错误页背景跟随主题，因此错误页 bg 在 dark 下仍用 app-bg */
:global(.dark) .browser-view__content {
  background: #fff;
}
</style>
