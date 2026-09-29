<template>
  <div class="toolbar">
    <div class="toolbar__left">
      <span v-if="tabDirty" class="toolbar__dot" :title="i18n('toolbar.unsavedDot')"></span>
      <span class="toolbar__name" :class="{ 'toolbar__name--path': tabPath }" :title="tabPath || i18n('toolbar.unsaved')">
        {{ tabPath || tabName || i18n('app.untitled') }}
      </span>
    </div>
    <div class="toolbar__segmented">
      <button
        class="toolbar__seg-btn"
        :class="{ 'is-active': mode === 'read' }"
        @click="emit('update:mode', 'read')"
      >
        {{ i18n('toolbar.readMode') }}
      </button>
      <button
        class="toolbar__seg-btn"
        :class="{ 'is-active': mode === 'wysiwyg' }"
        @click="emit('update:mode', 'wysiwyg')"
      >
        {{ i18n('toolbar.wysiwygMode') }}
      </button>
      <button
        class="toolbar__seg-btn"
        :class="{ 'is-active': mode === 'source' }"
        @click="emit('update:mode', 'source')"
      >
        {{ i18n('toolbar.sourceMode') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { i18n } from '../i18n'

defineProps<{
  tabName: string | null
  tabPath: string | null
  tabDirty: boolean
  mode: 'wysiwyg' | 'source' | 'read'
}>()

const emit = defineEmits<{
  (e: 'update:mode', mode: 'wysiwyg' | 'source' | 'read'): void
}>()
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  padding: 0px 6px 0px 10px;
  background: var(--app-bg);
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
}

.toolbar__left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.toolbar__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--app-active-text);
  flex-shrink: 0;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--app-active-text) 20%, transparent);
}

.toolbar__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: var(--app-text);
}

.toolbar__name--path {
  font-size: 12px;
  color: var(--app-text-muted);
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
}

/* 分段控件（segmented control） */
.toolbar__segmented {
  display: flex;
  align-items: center;
  background: var(--app-hover);
  border-radius: 7px;
  padding: 2px;
  gap: 2px;
}

.toolbar__seg-btn {
  height: 24px;
  padding: 0 12px;
  font-size: 12px;
  color: var(--app-text-muted);
  background: transparent;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  transition: all 0.18s cubic-bezier(0.4, 0, 0.2, 1);
  white-space: nowrap;
}

.toolbar__seg-btn:hover:not(.is-active) {
  color: var(--app-text);
  background: color-mix(in srgb, var(--app-bg) 60%, transparent);
}

.toolbar__seg-btn.is-active {
  color: var(--app-text);
  background: var(--app-bg);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.06);
  font-weight: 600;
}
</style>
