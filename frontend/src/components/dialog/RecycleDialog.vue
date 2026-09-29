<template>
  <Teleport to="body">
    <div v-if="visible" class="recycle-overlay" @click.self="close">
      <div class="recycle-dialog">
        <div class="recycle-header">
          <span class="recycle-title">{{ i18n('recycle.title') }}</span>
          <span class="recycle-header__actions">
            <button class="recycle-tool-btn" :title="i18n('common.refresh')" @click="loadItems">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 12a9 9 0 1 1-2.64-6.36L21 8" /><path d="M21 3v5h-5" />
              </svg>
            </button>
            <button class="recycle-empty-btn" :disabled="items.length === 0" @click="askEmpty">
              {{ i18n('recycle.emptyBin') }}
            </button>
            <button class="recycle-tool-btn" :title="i18n('common.close')" @click="close">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <path d="M18 6 6 18M6 6l12 12" />
              </svg>
            </button>
          </span>
        </div>

        <div class="recycle-body">
          <div v-if="loading" class="recycle-state">{{ i18n('recycle.loading') }}</div>
          <div v-else-if="items.length === 0" class="recycle-state">{{ i18n('recycle.empty') }}</div>
          <table v-else class="recycle-table">
            <thead>
              <tr>
                <th class="col-name">{{ i18n('recycle.colName') }}</th>
                <th class="col-path">{{ i18n('recycle.colOriginalPath') }}</th>
                <th class="col-size">{{ i18n('recycle.colSize') }}</th>
                <th class="col-type">{{ i18n('recycle.colType') }}</th>
                <th class="col-time">{{ i18n('recycle.colDeletedAt') }}</th>
                <th class="col-actions">{{ i18n('recycle.colActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in items" :key="item.id">
                <td class="col-name">
                  <span class="recycle-name">
                    <svg v-if="item.isDir" class="recycle-name__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7z" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                    <svg v-else class="recycle-name__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                      <path d="M14 2v6h6" />
                      <path d="M9 13h6M9 17h6" stroke-width="1.6" />
                    </svg>
                    <span :title="item.name">{{ item.name }}</span>
                  </span>
                </td>
                <td class="col-path" :title="item.originalPath.replace(/\\/g, '/')">{{ item.originalPath.replace(/\\/g, '/') }}</td>
                <td class="col-size">{{ formatSize(item.size) }}</td>
                <td class="col-type">{{ i18n(item.isDir ? 'recycle.typeDir' : 'recycle.typeFile') }}</td>
                <td class="col-time">{{ formatTime(item.deletedAt) }}</td>
                <td class="col-actions">
                  <button class="recycle-op" @click="askRestore(item)">{{ i18n('recycle.restore') }}</button>
                  <button class="recycle-op" @click="askRestoreTo(item)">{{ i18n('recycle.restoreTo') }}</button>
                  <button class="recycle-op recycle-op--danger" @click="askDeleteForever(item)">{{ i18n('recycle.deleteForever') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 彻底删除 / 清空确认 -->
    <ConfirmDialog v-model:visible="confirmVisible" :message="confirmMessage" @confirm="confirmAction" />

    <!-- 还原到：复用复制/移动目录选择浮层 -->
    <CopyOrMoveToDialog
      v-model:visible="restoreToVisible"
      :root-path="rootPath"
      mode="restore"
      @confirm="doRestoreTo"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { RecycleService } from '../../../bindings/XinText/internal/service'
import type { RecycleItem } from '../../../bindings/XinText/internal/service/models'
import ConfirmDialog from './ConfirmDialog.vue'
import CopyOrMoveToDialog from './CopyOrMoveToDialog.vue'
import { useNotice } from '../../composables/useNotice'
import { i18n } from '../../i18n'

const props = defineProps<{ visible: boolean; rootPath: string }>()
const emit = defineEmits<{
  'update:visible': [v: boolean]
  restored: []
}>()

const { showNotice } = useNotice()

const items = ref<RecycleItem[]>([])
const loading = ref(false)

// 打开时拉取列表
watch(
  () => props.visible,
  (v) => {
    if (v) loadItems()
  }
)

async function loadItems() {
  loading.value = true
  try {
    items.value = (await RecycleService.ListRecycleItems()) || []
  } catch (e: any) {
    showNotice(i18n('recycle.opFailed', { reason: e?.message || String(e) }))
    items.value = []
  } finally {
    loading.value = false
  }
}

function close() {
  emit('update:visible', false)
}

function formatSize(bytes: number): string {
  if (bytes == null || bytes < 0) return '—'
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}

function formatTime(unixSeconds: number): string {
  if (!unixSeconds) return '—'
  return new Date(unixSeconds * 1000).toLocaleString()
}

function fail(e: any) {
  showNotice(i18n('recycle.opFailed', { reason: e?.message || String(e) }))
}

// ---- 还原到原路径（确认框确认后执行） ----
async function doRestore(item: RecycleItem) {
  try {
    await RecycleService.RestoreRecycleItem(item.id)
    showNotice(i18n('recycle.restoreSuccess', { name: item.name }))
    emit('restored')
    await loadItems()
  } catch (e: any) {
    fail(e)
  }
}

// ---- 还原到指定目录 ----
const restoreToVisible = ref(false)
let restoreToItem: RecycleItem | null = null

function askRestoreTo(item: RecycleItem) {
  restoreToItem = item
  restoreToVisible.value = true
}

async function doRestoreTo(destDirPath: string) {
  const item = restoreToItem
  if (!item) return
  try {
    await RecycleService.RestoreRecycleItemTo(item.id, destDirPath)
    showNotice(i18n('recycle.restoreToSuccess', { name: item.name, path: destDirPath }))
    emit('restored')
    await loadItems()
  } catch (e: any) {
    fail(e)
  } finally {
    restoreToItem = null
  }
}

// ---- 彻底删除 / 清空 / 还原：共用确认框 ----
const confirmVisible = ref(false)
const confirmMessage = ref('')
let confirmKind: 'restore' | 'delete' | 'empty' = 'delete'
let confirmItem: RecycleItem | null = null

function askRestore(item: RecycleItem) {
  confirmKind = 'restore'
  confirmItem = item
  confirmMessage.value = i18n('recycle.confirmRestore', { name: item.name })
  confirmVisible.value = true
}

function askDeleteForever(item: RecycleItem) {
  confirmKind = 'delete'
  confirmItem = item
  confirmMessage.value = i18n('recycle.confirmDeleteForever', { name: item.name })
  confirmVisible.value = true
}

function askEmpty() {
  confirmKind = 'empty'
  confirmItem = null
  confirmMessage.value = i18n('recycle.confirmEmpty')
  confirmVisible.value = true
}

async function confirmAction() {
  if (confirmKind === 'restore' && confirmItem) {
    await doRestore(confirmItem)
  } else if (confirmKind === 'delete' && confirmItem) {
    const item = confirmItem
    try {
      await RecycleService.DeleteRecycleItemPermanent(item.id)
      showNotice(i18n('recycle.deleteForeverSuccess', { name: item.name }))
      await loadItems()
    } catch (e: any) {
      fail(e)
    }
  } else if (confirmKind === 'empty') {
    try {
      await RecycleService.EmptyRecycle()
      showNotice(i18n('recycle.emptySuccess'))
      await loadItems()
    } catch (e: any) {
      fail(e)
    }
  }
  confirmItem = null
}
</script>

<style scoped>
.recycle-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: recycle-fade 0.18s ease-out;
}

@keyframes recycle-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.recycle-dialog {
  display: flex;
  flex-direction: column;
  width: 870px;
  max-width: 92vw;
  height: 650px;
  max-height: 80vh;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: recycle-pop 0.2s ease-out;
  overflow: hidden;
}

@keyframes recycle-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.recycle-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 14px 0 18px;
  border-bottom: 1px solid var(--app-border, #dcdfe6);
  flex-shrink: 0;
}

.recycle-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text, #303133);
}

.recycle-header__actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.recycle-tool-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: none;
  border: none;
  border-radius: 4px;
  color: var(--app-text-muted, #909399);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.recycle-tool-btn:hover {
  background: var(--app-hover, #f5f7fa);
  color: var(--app-text, #303133);
}

.recycle-empty-btn {
  height: 26px;
  padding: 0 12px;
  font-size: 12px;
  color: #f56c6c;
  background: none;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.12s;
}

.recycle-empty-btn:hover:not(:disabled) {
  background: #fef0f0;
}

.recycle-empty-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.recycle-body {
  flex: 1;
  min-height: 120px;
  overflow: hidden;
}

.recycle-state {
  padding: 48px 0;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted, #909399);
}

.recycle-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  color: var(--app-text, #303133);
}

/* 表头与数据区分离：仅 tbody 滚动，滚动条不覆盖标题 */
.recycle-table thead,
.recycle-table tbody {
  display: block;
}

.recycle-table thead {
  /* 与 tbody 的 scrollbar-gutter 对齐，保证列宽一致 */
  padding-right: 6px;
}

.recycle-table thead tr,
.recycle-table tbody tr {
  display: table;
  width: 100%;
  table-layout: fixed;
}

.recycle-table tbody {
  /* 10 行数据(~32px/行)，超过即滚动 */
  max-height: 320px;
  overflow-y: auto;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.recycle-table tbody::-webkit-scrollbar {
  width: 6px;
}

.recycle-table tbody::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.18);
  border-radius: 3px;
}

.recycle-table tbody::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}

.recycle-table tbody::-webkit-scrollbar-track {
  background: transparent;
}

.recycle-table th {
  padding: 8px 10px;
  text-align: left;
  font-weight: 600;
  color: var(--app-text-muted, #909399);
  background: var(--app-glass-inner, rgba(255, 255, 255, 0.85));
  border-bottom: 1px solid var(--app-border, #dcdfe6);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recycle-table td {
  padding: 7px 10px;
  border-bottom: 1px solid var(--app-border, #ebeef5);
  vertical-align: middle;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.recycle-table tbody tr:hover {
  background: var(--app-hover, #f5f7fa);
}

.col-name {
  width: 24%;
}

.col-path {
  width: 23%;
  color: var(--app-text-muted, #909399);
}

.col-size {
  width: 10%;
}

.col-type {
  width: 10%;
}

.col-time {
  width: 15%;
}

.col-actions {
  width: 18%;
  white-space: nowrap;
  overflow: visible;
}

.recycle-name {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.recycle-name__icon {
  flex-shrink: 0;
  color: var(--app-text-muted, #909399);
}

.recycle-name > span {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.recycle-op {
  padding: 2px 6px;
  margin-right: 4px;
  font-size: 12px;
  color: var(--app-active-text, #409eff);
  background: none;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.recycle-op:last-child {
  margin-right: 0;
}

.recycle-op:hover {
  background: var(--app-active-bg, #e6f0ff);
}

.recycle-op--danger {
  color: #f56c6c;
}

.recycle-op--danger:hover {
  background: #fef0f0;
}
</style>
