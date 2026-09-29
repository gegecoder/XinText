<template>
  <Teleport to="body">
    <div v-if="noticeStore.visible" class="notice-overlay" @click.self="close" @keydown.esc="close">
      <div class="notice-dialog" role="alertdialog" aria-modal="true">
        <div class="notice-body">
          <svg class="notice-icon" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" stroke-linecap="round" />
            <line x1="12" y1="16" x2="12.01" y2="16" stroke-linecap="round" />
          </svg>
          <p class="notice-message">{{ noticeStore.message }}</p>
        </div>
        <div class="notice-actions">
          <button class="notice-btn" autofocus @click="close">{{ i18n('notice.ok') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { useNoticeStore } from '../../store/notice'
import { i18n } from '../../i18n'

const noticeStore = useNoticeStore()

function close() {
  noticeStore.close()
}

/** Esc 关闭（遮罩层无焦点时也能响应） */
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && noticeStore.visible) close()
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.notice-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: notice-fade 0.18s ease-out;
}

@keyframes notice-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.notice-dialog {
  min-width: 320px;
  max-width: 90vw;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: notice-pop 0.2s ease-out;
}

@keyframes notice-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.notice-body {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 24px 28px 16px;
}

.notice-icon {
  flex-shrink: 0;
  margin-top: 1px;
  color: var(--app-active-text, #409eff);
}

.notice-message {
  margin: 0;
  font-size: 14px;
  color: var(--app-text, #303133);
  line-height: 1.6;
  white-space: pre-line;
  word-break: break-all;
}

.notice-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 0 28px 20px;
}

.notice-btn {
  min-width: 80px;
  height: 34px;
  font-size: 13px;
  border: 1px solid var(--app-active-text, #409eff);
  border-radius: 6px;
  cursor: pointer;
  background: var(--app-active-text, #409eff);
  color: #fff;
  transition: opacity 0.15s;
}

.notice-btn:hover {
  opacity: 0.9;
}
</style>
