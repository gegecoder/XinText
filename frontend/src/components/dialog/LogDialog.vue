<template>
  <Teleport to="body">
    <div v-if="visible" class="log-overlay" @click.self="close">
      <div class="log-dialog">
        <header class="log-header">
          <div class="log-header__left">
            <span class="log-title">{{ i18n('log.title') }}</span>
            <div class="log-tabs">
              <button
                class="log-tab"
                :class="{ 'is-active': activeTab === 'go' }"
                @click="activeTab = 'go'"
              >{{ i18n('log.goTab') }}</button>
              <button
                class="log-tab"
                :class="{ 'is-active': activeTab === 'frontend' }"
                @click="activeTab = 'frontend'"
              >{{ i18n('log.frontendTab') }}</button>
            </div>
          </div>
          <button class="log-close" :title="i18n('common.close')" @click="close">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </header>

        <div class="log-body">
          <pre v-if="content" class="log-content">{{ content }}</pre>
          <div v-else-if="loading" class="log-empty">{{ i18n('log.loading') }}</div>
          <div v-else class="log-empty">{{ i18n('log.empty') }}</div>
        </div>

        <footer class="log-footer">
          <span class="log-path" :title="logDir">{{ logDir || '—' }}</span>
          <div class="log-footer__actions">
            <button class="log-btn" :disabled="loading" @click="loadCurrent">
              {{ loading ? i18n('log.loading') : i18n('log.refresh') }}
            </button>
            <button class="log-btn log-btn--danger" @click="clearCurrent">{{ i18n('log.clearCurrent') }}</button>
            <button class="log-btn" @click="revealInExplorer">{{ i18n('log.reveal') }}</button>
          </div>
        </footer>
      </div>

      <!-- 清理日志二次确认（WebView2 抑制 window.confirm，必须用自定义弹窗） -->
      <ConfirmDialog
        v-model:visible="confirmClearVisible"
        :message="activeTab === 'go' ? i18n('log.confirmClearGo') : i18n('log.confirmClearFrontend')"
        @confirm="doClear"
      />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { LogService, FileService } from '../../../bindings/XinText/internal/service'
import { useNotice } from '../../composables/useNotice'
import ConfirmDialog from './ConfirmDialog.vue'
import { i18n } from '../../i18n'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [v: boolean] }>()

const { showNotice } = useNotice()

const activeTab = ref<'go' | 'frontend'>('go')
const content = ref('')
const loading = ref(false)
const logDir = ref('')
const confirmClearVisible = ref(false)

function close() {
  emit('update:visible', false)
}

async function loadCurrent() {
  loading.value = true
  try {
    content.value = activeTab.value === 'go'
      ? await LogService.ReadGoLog()
      : await LogService.ReadFrontendLog()
  } catch (e: any) {
    showNotice(i18n('log.readFailed', { reason: e?.message || String(e) }))
    content.value = ''
  } finally {
    loading.value = false
  }
}

async function refreshLogDir() {
  try {
    logDir.value = await LogService.GetLogDir()
  } catch {
    logDir.value = ''
  }
}

function clearCurrent() {
  confirmClearVisible.value = true
}

async function doClear() {
  try {
    if (activeTab.value === 'go') {
      await LogService.ClearGoLog()
    } else {
      await LogService.ClearFrontendLog()
    }
    await loadCurrent()
    showNotice(i18n('log.cleared'))
  } catch (e: any) {
    showNotice(i18n('log.clearFailed', { reason: e?.message || String(e) }))
  }
}

async function revealInExplorer() {
  if (!logDir.value) {
    showNotice(i18n('log.dirEmpty'))
    return
  }
  try {
    await FileService.RevealInExplorer(logDir.value)
  } catch {
    showNotice(i18n('log.revealFailed'))
  }
}

watch(
  () => props.visible,
  (v) => {
    if (!v) return
    activeTab.value = 'go'
    refreshLogDir()
    loadCurrent()
  }
)

watch(activeTab, () => {
  if (props.visible) loadCurrent()
})

function onEscKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.visible) close()
}

onMounted(() => window.addEventListener('keydown', onEscKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onEscKeydown))
</script>

<style scoped>
.log-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: log-fade 0.18s ease-out;
}

@keyframes log-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.log-dialog {
  display: flex;
  flex-direction: column;
  width: 760px;
  max-width: 92vw;
  height: 500px;
  max-height: 86vh;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: log-pop 0.2s ease-out;
  overflow: hidden;
}

@keyframes log-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 16px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.log-header__left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.log-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
}

.log-tabs {
  display: inline-flex;
  padding: 2px;
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
}

.log-tab {
  height: 24px;
  padding: 0 12px;
  font-size: 12px;
  color: var(--app-text-muted);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.log-tab:hover {
  color: var(--app-text);
}

.log-tab.is-active {
  background: var(--app-bg);
  color: var(--app-active-text);
  font-weight: 600;
}

.log-close {
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

.log-close:hover {
  background: var(--app-hover);
  color: var(--app-text);
}

.log-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
}

.log-content {
  flex: 1;
  margin: 0;
  padding: 12px 16px;
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
  font-size: 12px;
  line-height: 1.55;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
}

.log-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  font-size: 13px;
  color: var(--app-text-muted);
}

.log-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 44px;
  padding: 0 16px;
  border-top: 1px solid var(--app-border);
  flex-shrink: 0;
}

.log-path {
  flex: 1;
  min-width: 0;
  font-size: 11px;
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
  color: var(--app-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-footer__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.log-btn {
  height: 28px;
  padding: 0 14px;
  font-size: 12px;
  color: var(--app-text);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  border: 1px solid var(--app-border);
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.log-btn:hover:not(:disabled) {
  background: var(--app-hover);
  color: var(--app-active-text);
}

.log-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.log-btn--danger:hover {
  color: #f56c6c;
  border-color: #f56c6c;
}
</style>
