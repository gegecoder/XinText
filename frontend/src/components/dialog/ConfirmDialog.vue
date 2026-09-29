<template>
  <Teleport to="body">
    <div v-if="visible" class="confirm-overlay" @click.self="cancel">
      <div class="confirm-dialog">
        <div class="confirm-body">
          <p class="confirm-message">{{ message }}</p>
        </div>
        <div class="confirm-actions">
          <!-- save 模式：不保存 / 取消 / 保存（未保存文档关闭确认） -->
          <button v-if="mode === 'save'" class="confirm-btn confirm-btn--cancel" @click="discard">{{ i18n('confirm.dontSave') }}</button>
          <!-- alert 模式仅显示「确定」，用于纯提示场景 -->
          <button v-if="mode !== 'alert'" class="confirm-btn confirm-btn--cancel" @click="cancel">{{ i18n('confirm.cancel') }}</button>
          <button class="confirm-btn confirm-btn--ok" @click="confirm">{{ mode === 'save' ? i18n('confirm.save') : i18n('confirm.ok') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { i18n } from '../../i18n'

withDefaults(
  defineProps<{ visible: boolean; message: string; mode?: 'confirm' | 'alert' | 'save' }>(),
  { mode: 'confirm' }
)
const emit = defineEmits<{
  'update:visible': [v: boolean]
  'confirm': []
  'discard': []
}>()

function cancel() {
  emit('update:visible', false)
}

function confirm() {
  emit('update:visible', false)
  emit('confirm')
}

function discard() {
  emit('update:visible', false)
  emit('discard')
}
</script>

<style scoped>
.confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: confirm-fade 0.18s ease-out;
}

@keyframes confirm-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.confirm-dialog {
  min-width: 320px;
  max-width: 90vw;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: confirm-pop 0.2s ease-out;
}

@keyframes confirm-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.confirm-body {
  padding: 24px 28px 16px;
  text-align: center;
}

.confirm-message {
  margin: 0;
  font-size: 14px;
  color: var(--app-text, #303133);
  line-height: 1.6;
  white-space: pre-line;
  word-break: break-all;
}

.confirm-actions {
  display: flex;
  gap: 10px;
  padding: 0 28px 20px;
  justify-content: center;
}

.confirm-btn {
  min-width: 80px;
  height: 34px;
  font-size: 13px;
  border: 1px solid var(--app-border, #dcdfe6);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.confirm-btn--cancel {
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.5));
  color: var(--app-text, #606266);
}

.confirm-btn--cancel:hover {
  background: var(--app-hover, #f5f7fa);
}

.confirm-btn--ok {
  background: var(--app-active-text, #409eff);
  color: #fff;
  border-color: var(--app-active-text, #409eff);
}

.confirm-btn--ok:hover {
  opacity: 0.9;
}
</style>
