<template>
  <Teleport to="body">
    <div v-if="visible" class="settings-overlay" @click.self="close">
      <div class="settings-dialog">
        <div class="settings-header">
          <span class="settings-title">{{ i18n('settings.title') }}</span>
          <button class="settings-close" :title="i18n('common.close')" @click="close">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="settings-body">
          <!-- 左侧分类列表 -->
          <aside class="settings-nav">
            <button
              v-for="cat in categories"
              :key="cat.key"
              class="settings-nav__item"
              :class="{ 'is-active': activeCategory === cat.key }"
              @click="activeCategory = cat.key"
            >
              <span class="settings-nav__icon" v-html="cat.icon"></span>
              <span>{{ i18n(cat.labelKey) }}</span>
            </button>
          </aside>

          <!-- 右侧内容区 -->
          <section class="settings-content">
            <!-- 通用：界面语言 -->
            <div v-if="activeCategory === 'general'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.generalLanguageTitle') }}</h3>
              <div class="theme-options">
                <button
                  v-for="loc in SUPPORTED_LOCALES"
                  :key="loc.value"
                  class="theme-option"
                  :class="{ 'is-active': currentLanguage === loc.value }"
                  @click="selectLanguage(loc.value)"
                >
                  <span class="theme-option__preview lang-option__preview">{{ loc.value === 'zh' ? '文' : 'A' }}</span>
                  <span class="theme-option__label">{{ loc.label }}</span>
                  <svg v-if="currentLanguage === loc.value" class="theme-option__check" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                </button>
              </div>
              <p class="settings-pane__hint">{{ i18n('settings.generalLanguageHint') }}</p>

              <!-- 自动保存开关 -->
              <div class="settings-pane__divider"></div>
              <div class="toggle-row">
                <div class="toggle-row__text">
                  <div class="toggle-row__title">{{ i18n('settings.autoSaveTitle') }}</div>
                  <div class="toggle-row__desc">{{ i18n('settings.autoSaveDesc') }}</div>
                </div>
                <button
                  class="toggle-switch"
                  :class="{ 'is-on': autoSaveEnabled }"
                  role="switch"
                  :aria-checked="autoSaveEnabled"
                  :title="i18n('settings.autoSaveTitle')"
                  @click="toggleAutoSave"
                >
                  <span class="toggle-switch__knob"></span>
                </button>
              </div>
            </div>

            <!-- 外观 -->
            <div v-else-if="activeCategory === 'appearance'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.themeTitle') }}</h3>
              <div class="theme-options">
                <button
                  class="theme-option"
                  :class="{ 'is-active': currentTheme === 'light' }"
                  @click="selectTheme('light')"
                >
                  <span class="theme-option__preview theme-option__preview--light"></span>
                  <span class="theme-option__label">{{ i18n('settings.themeLight') }}</span>
                  <svg v-if="currentTheme === 'light'" class="theme-option__check" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                </button>
                <button
                  class="theme-option"
                  :class="{ 'is-active': currentTheme === 'dark' }"
                  @click="selectTheme('dark')"
                >
                  <span class="theme-option__preview theme-option__preview--dark"></span>
                  <span class="theme-option__label">{{ i18n('settings.themeDark') }}</span>
                  <svg v-if="currentTheme === 'dark'" class="theme-option__check" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                </button>
              </div>
              <p class="settings-pane__hint">{{ i18n('settings.themeHint') }}</p>
            </div>

            <!-- 导出 -->
            <div v-else-if="activeCategory === 'export'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.pandocTitle') }}</h3>
              <p class="settings-pane__desc">{{ i18n('settings.pandocDesc') }}</p>
              <p class="settings-pane__download">
                {{ i18n('settings.notInstalled') }}
                <a href="javascript:void(0)" class="download-link" @click="openExternal('https://github.com/jgm/pandoc/releases')">{{ i18n('settings.downloadPandoc') }}</a>
              </p>
              <div class="pandoc-row">
                <input
                  v-model="pandocPathInput"
                  class="pandoc-input"
                  type="text"
                  :placeholder="i18n('settings.pandocPlaceholder')"
                  spellcheck="false"
                />
                <button class="pandoc-btn" @click="browsePandoc">{{ i18n('common.browse') }}</button>
              </div>
              <div class="pandoc-status">
                <span class="pandoc-status__dot" :class="pandocAvailable ? 'is-ok' : 'is-bad'"></span>
                <span v-if="checking">{{ i18n('settings.checkingPandoc') }}</span>
                <span v-else-if="pandocAvailable">{{ i18n('settings.pandocOk', { detail: pandocVersion ? i18n('settings.pandocVersionDetail', { version: pandocVersion }) : '' }) }}</span>
                <span v-else>{{ i18n('settings.pandocMissing') }}</span>
              </div>
              <div class="pandoc-actions">
                <button class="pandoc-save" :disabled="saving" @click="saveExportPaths">
                  {{ saving ? i18n('common.saving') : i18n('common.save') }}
                </button>
              </div>
            </div>
            <!-- 图片 -->
            <div v-else-if="activeCategory === 'image'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.imageTitle') }}</h3>
              <p class="settings-pane__desc">
                {{ i18n('settings.imageDesc') }}
              </p>
              <div class="pandoc-row">
                <input
                  v-model="imageDirInput"
                  class="pandoc-input"
                  type="text"
                  :placeholder="i18n('settings.imagePlaceholder')"
                  spellcheck="false"
                />
                <button class="pandoc-btn" @click="browseImageDir">{{ i18n('common.browse') }}</button>
                <button class="pandoc-btn" :title="i18n('settings.resetPathTip')" @click="imageDirInput = ''">{{ i18n('common.restoreDefault') }}</button>
              </div>
              <div class="pandoc-status">
                <span class="pandoc-status__dot" :class="imageDirInput.trim() ? 'is-ok' : 'is-ok'"></span>
                <span v-if="imageDirInput.trim()">{{ i18n('settings.imageStatusCustom', { path: imageDirInput.trim() }) }}</span>
                <span v-else>{{ i18n('settings.imageStatusDefault') }}</span>
              </div>
              <div class="pandoc-actions">
                <button class="pandoc-save" :disabled="saving" @click="saveImageDir">
                  {{ saving ? i18n('common.saving') : i18n('common.save') }}
                </button>
              </div>
            </div>
            <!-- 日志 -->
            <div v-else-if="activeCategory === 'log'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.logTitle') }}</h3>
              <p class="settings-pane__desc">
                {{ i18n('settings.logDesc') }}
              </p>
              <div class="pandoc-row">
                <input
                  v-model="logDirInput"
                  class="pandoc-input"
                  type="text"
                  :placeholder="i18n('settings.logPlaceholder')"
                  spellcheck="false"
                />
                <button class="pandoc-btn" @click="browseLogDir">{{ i18n('common.browse') }}</button>
                <button class="pandoc-btn" :title="i18n('settings.resetPathTip')" @click="logDirInput = ''">{{ i18n('common.restoreDefault') }}</button>
              </div>
              <div class="pandoc-status">
                <span class="pandoc-status__dot is-ok"></span>
                <span v-if="currentLogDir">{{ i18n('settings.logStatus', { path: currentLogDir }) }}</span>
                <span v-else>{{ i18n('settings.logStatusEmpty') }}</span>
              </div>
              <div class="pandoc-actions">
                <button class="pandoc-save" :disabled="saving" @click="saveLogDir">
                  {{ saving ? i18n('common.saving') : i18n('common.save') }}
                </button>
              </div>
            </div>
            <!-- 回收站 -->
            <div v-else-if="activeCategory === 'recycle'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.recycleTitle') }}</h3>
              <p class="settings-pane__desc">
                {{ i18n('settings.recycleDesc') }}
              </p>
              <div class="pandoc-row">
                <input
                  v-model="recycleDirInput"
                  class="pandoc-input"
                  type="text"
                  :placeholder="i18n('settings.recyclePlaceholder')"
                  spellcheck="false"
                />
                <button class="pandoc-btn" @click="browseRecycleDir">{{ i18n('common.browse') }}</button>
                <button class="pandoc-btn" :title="i18n('settings.resetPathTip')" @click="recycleDirInput = ''">{{ i18n('common.restoreDefault') }}</button>
              </div>
              <div class="pandoc-status">
                <span class="pandoc-status__dot is-ok"></span>
                <span v-if="currentRecycleDir">{{ i18n('settings.recycleStatusCustom', { path: currentRecycleDir }) }}</span>
                <span v-else>{{ i18n('settings.recycleStatusDefault') }}</span>
              </div>
              <div class="pandoc-actions">
                <button class="pandoc-save" :disabled="saving" @click="saveRecycleDir">
                  {{ saving ? i18n('common.saving') : i18n('common.save') }}
                </button>
              </div>
            </div>
            <!-- 浏览器 -->
            <div v-else-if="activeCategory === 'browser'" class="settings-pane">
              <h3 class="settings-pane__title">{{ i18n('settings.browserTitle') }}</h3>
              <p class="settings-pane__desc">
                {{ i18n('settings.browserDesc') }}
              </p>
              <div class="pandoc-row">
                <input
                  v-model="browserHomeURLInput"
                  class="pandoc-input"
                  type="text"
                  :placeholder="i18n('settings.browserPlaceholder')"
                  spellcheck="false"
                />
                <button class="pandoc-btn" :title="i18n('settings.resetPathTip')" @click="browserHomeURLInput = ''">{{ i18n('common.restoreDefault') }}</button>
              </div>
              <div class="pandoc-status">
                <span class="pandoc-status__dot is-ok"></span>
                <span v-if="browserHomeURLInput.trim()">{{ i18n('settings.browserStatusCustom', { url: browserHomeURLInput.trim() }) }}</span>
                <span v-else>{{ i18n('settings.browserStatusDefault') }}</span>
              </div>
              <div class="pandoc-actions">
                <button class="pandoc-save" :disabled="saving" @click="saveBrowserHomeURL">
                  {{ saving ? i18n('common.saving') : i18n('common.save') }}
                </button>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Browser } from '@wailsio/runtime'
import { FileService, ExportService, LogService, RecycleService } from '../../../bindings/XinText/internal/service'
import { usePreferencesStore } from '../../store/preferences'
import { useNotice } from '../../composables/useNotice'
import { i18n, resolveLocale, SUPPORTED_LOCALES, type Locale } from '../../i18n'

const props = defineProps<{ visible: boolean }>()

const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'theme-change', theme: string): void
  (e: 'language-change', language: Locale): void
  (e: 'saved'): void
}>()

const prefsStore = usePreferencesStore()

type CategoryKey = 'general' | 'appearance' | 'export' | 'image' | 'log' | 'recycle' | 'browser'
type CategoryItem = { key: CategoryKey; labelKey: string; icon: string }

const categories = computed<CategoryItem[]>(() => [
  {
    key: 'general',
    labelKey: 'settings.navGeneral',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a15 15 0 0 1 0 18 15 15 0 0 1 0-18z"/></svg>'
  },
  {
    key: 'appearance',
    labelKey: 'settings.navAppearance',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 0 0 18c.83 0 1.5-.67 1.5-1.5 0-.39-.15-.74-.39-1.01a1.5 1.5 0 0 1 1.12-2.49H16a5 5 0 0 0 5-5c0-4.42-4.03-8-9-8z"/><circle cx="7.5" cy="10.5" r="1"/><circle cx="12" cy="7.5" r="1"/><circle cx="16.5" cy="10.5" r="1"/></svg>'
  },
  {
    key: 'export',
    labelKey: 'settings.navExport',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/></svg>'
  },
  {
    key: 'image',
    labelKey: 'settings.navImage',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/></svg>'
  },
  {
    key: 'log',
    labelKey: 'settings.navLog',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="M9 13h6M9 17h6"/></svg>'
  },
  {
    key: 'recycle',
    labelKey: 'settings.navRecycle',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>'
  },
  {
    key: 'browser',
    labelKey: 'settings.navBrowser',
    icon: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a15 15 0 0 1 0 18 15 15 0 0 1 0-18z"/></svg>'
  }
])

const activeCategory = ref<CategoryKey>('general')
const currentTheme = ref('light')
const currentLanguage = ref<Locale>('zh')
const autoSaveEnabled = ref(false)
const pandocPathInput = ref('')
const imageDirInput = ref('')
const logDirInput = ref('')
const currentLogDir = ref('')
const recycleDirInput = ref('')
const currentRecycleDir = ref('')
const browserHomeURLInput = ref('')
const checking = ref(false)
const saving = ref(false)
const pandocAvailable = ref(false)
const pandocVersion = ref('')

// 全局提示（NoticeDialog 由 App.vue 单例挂载）
const { showNotice } = useNotice()

// 通过系统默认浏览器打开外部链接（WebView 内无法直接访问外网）
function openExternal(url: string) {
  Browser.OpenURL(url).catch(() => window.open(url, '_blank'))
}

// 每次打开时同步最新配置并重新检测 pandoc
watch(
  () => props.visible,
  (v) => {
    if (!v) return
    activeCategory.value = 'general'
    currentTheme.value = prefsStore.config.theme || 'light'
    currentLanguage.value = resolveLocale(prefsStore.config.language)
    autoSaveEnabled.value = prefsStore.config.autoSave ?? false
    pandocPathInput.value = prefsStore.config.pandocPath || ''
    imageDirInput.value = prefsStore.config.imageDir || ''
    logDirInput.value = prefsStore.config.logDir || ''
    refreshLogDir()
    recycleDirInput.value = prefsStore.config.recycleDir || ''
    refreshRecycleDir()
    browserHomeURLInput.value = prefsStore.config.browserHomeURL || ''
    checkPandoc()
  }
)

function close() {
  emit('update:visible', false)
}

function selectTheme(theme: string) {
  currentTheme.value = theme
  emit('theme-change', theme)
}

/** 语言选择：立即生效并持久化（与主题一致的交互） */
function selectLanguage(language: Locale) {
  currentLanguage.value = language
  emit('language-change', language)
}

/** 自动保存开关：立即生效并持久化（App.vue 读 prefsStore.config.autoSave 响应式生效） */
async function toggleAutoSave() {
  const next = !autoSaveEnabled.value
  autoSaveEnabled.value = next
  await prefsStore.setAutoSave(next)
  emit('saved')
}

async function browsePandoc() {
  // 默认过滤 exe 类型
  const path = await FileService.PickPandocFile()
  if (!path) return
  pandocPathInput.value = path
  // 选中后自动校验所选文件是否为可用 pandoc
  checking.value = true
  try {
    const info = await ExportService.CheckPandocPath(path)
    if (info.available) {
      pandocAvailable.value = true
      pandocVersion.value = info.version || ''
    } else {
      pandocAvailable.value = false
      pandocVersion.value = ''
      showNotice(i18n('settings.invalidPandoc'))
    }
  } catch (e) {
    console.error('check pandoc path failed', e)
    pandocAvailable.value = false
    pandocVersion.value = ''
    showNotice(i18n('settings.checkFailed'))
  } finally {
    checking.value = false
  }
}

async function checkPandoc() {
  checking.value = true
  try {
    const info = await ExportService.CheckPandoc()
    pandocAvailable.value = !!info.available
    pandocVersion.value = info.version || ''
  } catch (e) {
    console.error('check pandoc failed', e)
    pandocAvailable.value = false
    pandocVersion.value = ''
  } finally {
    checking.value = false
  }
}

async function saveExportPaths() {
  saving.value = true
  try {
    await prefsStore.setPandocPath(pandocPathInput.value.trim())
    await checkPandoc()
    emit('saved')
  } finally {
    saving.value = false
  }
}

/** 浏览选择图片保存目录 */
async function browseImageDir() {
  const dir = await FileService.PickOpenDirectory()
  if (!dir) return
  imageDirInput.value = dir
}

/** 保存图片目录设置 */
async function saveImageDir() {
  saving.value = true
  try {
    await prefsStore.setImageDir(imageDirInput.value.trim())
    showNotice(i18n('settings.imageUpdated'))
    emit('saved')
  } finally {
    saving.value = false
  }
}

/** 浏览选择日志存储目录 */
async function browseLogDir() {
  const dir = await FileService.PickOpenDirectory()
  if (!dir) return
  logDirInput.value = dir
}

/** 拉取当前实际日志目录（后端解析后） */
async function refreshLogDir() {
  try {
    currentLogDir.value = await LogService.GetLogDir()
  } catch {
    currentLogDir.value = ''
  }
}

/** 保存日志目录设置：持久化 + 后端立即切换 */
async function saveLogDir() {
  saving.value = true
  try {
    const trimmed = logDirInput.value.trim()
    await prefsStore.setLogDir(trimmed)
    // 后端切换目录并返回新绝对路径
    const newDir = await LogService.SetLogDir(trimmed)
    currentLogDir.value = newDir
    logDirInput.value = prefsStore.config.logDir || ''
    showNotice(i18n('settings.logUpdated'))
    emit('saved')
  } catch (e: any) {
    showNotice(i18n('settings.saveFailed', { reason: e?.message || String(e) }))
  } finally {
    saving.value = false
  }
}

/** 浏览选择回收站目录 */
async function browseRecycleDir() {
  const dir = await FileService.PickOpenDirectory()
  if (!dir) return
  recycleDirInput.value = dir
}

/** 拉取后端当前生效的回收站绝对目录 */
async function refreshRecycleDir() {
  try {
    currentRecycleDir.value = await RecycleService.GetRecycleDir()
  } catch {
    currentRecycleDir.value = ''
  }
}

/** 保存回收站目录：后端校验/建目录并持久化，前端同步本地配置 */
async function saveRecycleDir() {
  saving.value = true
  try {
    const newDir = await RecycleService.SetRecycleDir(recycleDirInput.value.trim())
    await prefsStore.load()
    currentRecycleDir.value = newDir
    recycleDirInput.value = prefsStore.config.recycleDir || ''
    showNotice(i18n('settings.recycleUpdated'))
    emit('saved')
  } catch (e: any) {
    showNotice(i18n('settings.saveFailed', { reason: e?.message || String(e) }))
  } finally {
    saving.value = false
  }
}

/** 保存浏览器主页 URL：空值时由后端用内置默认兜底 */
async function saveBrowserHomeURL() {
  saving.value = true
  try {
    let url = browserHomeURLInput.value.trim()
    // 用户输入若无协议前缀，自动补 https://
    if (url && !/^[a-z]+:\/\//i.test(url)) {
      url = 'https://' + url
      browserHomeURLInput.value = url
    }
    await prefsStore.setBrowserHomeURL(url)
    showNotice(i18n('settings.browserUpdated'))
    emit('saved')
  } catch (e: any) {
    showNotice(i18n('settings.saveFailed', { reason: e?.message || String(e) }))
  } finally {
    saving.value = false
  }
}

function onEscKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) close()
}

onMounted(() => window.addEventListener('keydown', onEscKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onEscKeydown))
</script>

<style scoped>
.settings-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: settings-fade 0.18s ease-out;
}

@keyframes settings-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.settings-dialog {
  display: flex;
  flex-direction: column;
  width: 680px;
  max-width: 90vw;
  height: 440px;
  max-height: 85vh;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: settings-pop 0.2s ease-out;
  overflow: hidden;
}

@keyframes settings-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.settings-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 16px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.settings-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.settings-close {
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
  transition: background 0.12s, color 0.12s;
}

.settings-close:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

.settings-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

/* 左侧分类 */
.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 150px;
  padding: 10px 8px;
  border-right: 1px solid var(--app-border);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  flex-shrink: 0;
}

.settings-nav__item {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--app-text);
  background: none;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s, color 0.12s;
}

.settings-nav__item:hover {
  background: var(--app-hover);
}

.settings-nav__item.is-active {
  background: var(--app-active-bg);
  color: var(--app-active-text);
  font-weight: 500;
}

.settings-nav__icon {
  display: flex;
  align-items: center;
  color: inherit;
}

/* 右侧内容区 */
.settings-content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 18px 20px;
}

.settings-pane__title {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.settings-pane__desc {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.6;
}

/* 下载链接行 */
.settings-pane__download {
  margin: 0 0 10px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.download-link {
  color: var(--app-active-text);
  text-decoration: none;
}

.download-link:hover {
  text-decoration: underline;
}

.settings-pane__hint {
  margin: 14px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
  opacity: 0.8;
}

/* 分类内分隔线 */
.settings-pane__divider {
  height: 1px;
  margin: 18px 0;
  background: var(--app-border);
}

/* 开关行：左侧标题+描述，右侧 toggle */
.toggle-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.toggle-row__text {
  flex: 1;
  min-width: 0;
}

.toggle-row__title {
  font-size: 13px;
  font-weight: 500;
  color: var(--app-text);
}

.toggle-row__desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.6;
}

/* iOS 风格 toggle 开关 */
.toggle-switch {
  position: relative;
  flex-shrink: 0;
  width: 38px;
  height: 22px;
  margin-top: 2px;
  padding: 0;
  background: var(--app-border);
  border: none;
  border-radius: 11px;
  cursor: pointer;
  transition: background 0.18s;
}

.toggle-switch.is-on {
  background: var(--app-active-text);
}

.toggle-switch__knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  background: #fff;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
  transition: transform 0.18s;
}

.toggle-switch.is-on .toggle-switch__knob {
  transform: translateX(16px);
}

/* 主题选项卡片 */
.theme-options {
  display: flex;
  gap: 14px;
}

.theme-option {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 10px;
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 2px solid var(--app-border);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.theme-option:hover {
  border-color: var(--app-active-text);
}

.theme-option.is-active {
  border-color: var(--app-active-text);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-active-text) 25%, transparent);
}

.theme-option__preview {
  width: 88px;
  height: 56px;
  border-radius: 6px;
  border: 1px solid var(--app-border);
}

.theme-option__preview--light {
  background: linear-gradient(180deg, #ffffff 0%, #f5f7fa 100%);
}

.theme-option__preview--dark {
  background: linear-gradient(180deg, #24292e 0%, #1d2125 100%);
}

/* 语言卡片预览：用语言代表字符（文 / A）示意 */
.lang-option__preview {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 600;
  color: var(--app-text-muted);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
}

.theme-option__label {
  font-size: 12px;
  color: var(--app-text);
}

.theme-option__check {
  position: absolute;
  top: 6px;
  right: 6px;
  color: var(--app-active-text);
}

/* pandoc 设置 */
.pandoc-row {
  display: flex;
  gap: 8px;
}

.pandoc-input {
  flex: 1;
  height: 30px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
  outline: none;
  transition: border-color 0.15s;
}

.pandoc-input:focus {
  border-color: var(--app-active-text);
}

.pandoc-btn {
  height: 30px;
  padding: 0 14px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.pandoc-btn:hover {
  background: var(--app-hover);
  color: var(--app-active-text);
}

.pandoc-status {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.pandoc-status__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.pandoc-status__dot.is-ok {
  background: #67c23a;
}

.pandoc-status__dot.is-bad {
  background: #f56c6c;
}

.pandoc-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.pandoc-save {
  height: 30px;
  padding: 0 22px;
  font-size: 13px;
  color: #fff;
  background: var(--app-active-text);
  border: 1px solid var(--app-active-text);
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.pandoc-save:hover {
  opacity: 0.9;
}

.pandoc-save:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
