<template>
  <Teleport to="body">
    <div v-if="visible && info" class="update-overlay" @click.self="close">
      <div class="update-dialog">
        <h3 class="update-title">{{ i18n('update.foundTitle') }}</h3>

        <div class="update-versions">
          <span class="ver-pill ver-old">v{{ info.currentVersion }}</span>
          <span class="ver-arrow">→</span>
          <span class="ver-pill ver-new">v{{ info.latestVersion }}</span>
          <span v-if="info.prerelease" class="ver-beta">{{ i18n('update.prerelease') }}</span>
        </div>
        <p v-if="info.publishedAt" class="update-date">{{ i18n('update.publishedAt', { date: info.publishedAt }) }}</p>

        <div v-if="info.releaseNotes" class="update-notes">
          <p class="notes-title">{{ i18n('update.notesTitle') }}</p>
          <pre class="notes-body">{{ info.releaseNotes }}</pre>
        </div>

        <div class="update-actions">
          <button class="btn-plain" @click="close">{{ i18n('update.later') }}</button>
          <button class="btn-primary" @click="download">{{ i18n('update.downloadNow') }}</button>
        </div>
        <a href="javascript:void(0)" class="update-page" @click="openReleasePage">{{ i18n('update.viewPage') }}</a>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Browser } from '@wailsio/runtime'
import { useUpdate } from '../../composables/useUpdate'
import { i18n } from '../../i18n'

// 发现新版本弹窗：展示当前/最新版本、发布时间、更新说明，支持跳转下载或发布页
const { updateInfo, showUpdateModal } = useUpdate()

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: [] }>()

const info = computed(() => updateInfo.value)

function close() {
  showUpdateModal.value = false
  emit('close')
}

// 立即下载：优先 .exe 附件，其次任意附件，再退回发布页
function download() {
  const url = info.value?.downloadUrl || info.value?.releaseUrl
  if (url) Browser.OpenURL(url).catch(() => window.open(url, '_blank'))
}

function openReleasePage() {
  if (info.value?.releaseUrl) {
    Browser.OpenURL(info.value.releaseUrl).catch(() => window.open(info.value!.releaseUrl, '_blank'))
  }
}
</script>

<style scoped>
.update-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: update-fade 0.18s ease-out;
}

@keyframes update-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.update-dialog {
  width: 460px;
  max-width: 90vw;
  max-height: 80vh;
  overflow-y: auto;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  padding: 22px 26px 18px;
  text-align: center;
  animation: update-pop 0.2s ease-out;
}

@keyframes update-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.update-title {
  margin: 0 0 14px;
  font-size: 16px;
  font-weight: 700;
  color: var(--app-text, #303133);
}

.update-versions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-bottom: 6px;
}

.ver-pill {
  padding: 3px 12px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 700;
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
}

.ver-old {
  background: var(--app-hover-bg, rgba(0, 0, 0, 0.04));
  color: var(--app-text-muted, #909399);
  border: 1px solid var(--app-border, #e4e7ed);
}

.ver-new {
  background: rgba(64, 158, 255, 0.1);
  color: var(--app-active-text, #409eff);
  border: 1px solid rgba(64, 158, 255, 0.4);
}

.ver-arrow {
  color: var(--app-text-muted, #909399);
}

.ver-beta {
  font-size: 11px;
  color: #e6a23c;
  border: 1px solid currentColor;
  border-radius: 4px;
  padding: 0 4px;
}

.update-date {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}

.update-notes {
  text-align: left;
  margin-bottom: 14px;
}

.notes-title {
  margin: 0 0 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.notes-body {
  margin: 0;
  max-height: 180px;
  overflow-y: auto;
  padding: 8px 10px;
  border: 1px solid var(--app-border, #e4e7ed);
  border-radius: 6px;
  background: var(--app-hover-bg, rgba(0, 0, 0, 0.02));
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--app-text, #303133);
}

.update-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}

.btn-plain,
.btn-primary {
  flex: 1;
  height: 34px;
  font-size: 13px;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s, background 0.15s;
}

.btn-plain {
  color: var(--app-text, #303133);
  background: transparent;
  border: 1px solid var(--app-border, #dcdfe6);
}

.btn-plain:hover {
  background: var(--app-hover-bg, rgba(0, 0, 0, 0.04));
}

.btn-primary {
  color: #fff;
  background: var(--app-active-text, #409eff);
  border: none;
}

.btn-primary:hover {
  opacity: 0.9;
}

.update-page {
  font-size: 12px;
  color: var(--app-text-muted, #909399);
  text-decoration: none;
}

.update-page:hover {
  color: var(--app-active-text, #409eff);
  text-decoration: underline;
}
</style>
