<template>
  <Teleport to="body">
    <div v-if="visible" class="about-overlay" @click.self="close">
      <div class="about-dialog">
        <div class="about-body">
          <img class="about-logo" src="/logo.png" :alt="appName" />
          <h2 class="about-name">{{ appName }}</h2>
          <p class="about-desc">{{ appDesc }}</p>

          <dl class="about-list">
            <!-- 版本行内联检查更新 -->
            <div class="about-row">
              <dt>{{ i18n('about.version') }}</dt>
              <dd class="about-value ver-row">
                <span class="ver-num">{{ appVersion || '—' }}</span>
                <button class="check-btn" :disabled="checking" @click="checkForUpdate()">
                  {{ checking ? i18n('about.checking') : i18n('about.checkUpdate') }}
                </button>
                <span v-if="checkMessage" class="update-state" :class="checkStatus">{{ checkMessage }}</span>
              </dd>
            </div>
            <div class="about-row">
              <dt>{{ i18n('about.author') }}</dt>
              <dd class="about-value">{{ authorName }}</dd>
            </div>
            <div class="about-row" v-if="authorBlog">
              <dt>{{ i18n('about.blog') }}</dt>
              <dd class="about-value">
                <a href="javascript:void(0)" class="about-link" @click="openBlog">{{ authorBlog }}</a>
              </dd>
            </div>
          </dl>

          <!-- 发现新版本弹窗 -->
          <UpdateDialog :visible="showUpdateModal" @close="showUpdateModal = false" />

          <!-- 赞助作者（图片加载失败时整块隐藏） -->
          <div v-if="rewardVisible" class="about-reward">
            <p class="reward-title">{{ i18n('about.rewardTitle') }}</p>
            <img class="reward-qr" src="/donate.png" :alt="i18n('about.rewardTip')" @error="rewardVisible = false" />
            <p class="reward-tip">{{ i18n('about.rewardTip') }}</p>
          </div>

          <button class="about-ok" @click="close">{{ i18n('about.gotIt') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Browser } from '@wailsio/runtime'
import appConfig from '@app-config'
import UpdateDialog from './UpdateDialog.vue'
import { useUpdate } from '../../composables/useUpdate'
import { i18n } from '../../i18n'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [v: boolean] }>()

const appName = appConfig.name
const appDesc = appConfig.description
const authorName = appConfig.author?.name ?? ''
const authorBlog = appConfig.author?.blog ?? ''

// 版本与更新检测（模块级单例，与 UpdateDialog 共享状态）
const { appVersion, checking, checkStatus, checkMessage, showUpdateModal, checkForUpdate } = useUpdate()

const rewardVisible = ref(true)

function close() {
  emit('update:visible', false)
}

function openBlog() {
  if (authorBlog) {
    // 通过系统默认浏览器打开（WebView 内 window.open 无法访问外链）
    Browser.OpenURL(authorBlog).catch(() => window.open(authorBlog, '_blank'))
  }
}
</script>

<style scoped>
.about-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-glass-scrim, rgba(0, 0, 0, 0.3));
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: about-fade 0.18s ease-out;
}

@keyframes about-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}

.about-dialog {
  width: 500px;
  max-width: 90vw;
  max-height: 86vh;
  overflow-y: auto;
  background: var(--app-glass-bg, rgba(255, 255, 255, 0.72));
  border: 1px solid var(--app-glass-border, rgba(255, 255, 255, 0.65));
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  animation: about-pop 0.2s ease-out;
}

@keyframes about-pop {
  from { opacity: 0; }
  to { opacity: 1; }
}

.about-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 28px 28px 20px;
}

.about-logo {
  width: 64px;
  height: 64px;
  margin-bottom: 14px;
  border-radius: 14px;
  box-shadow: 0 8px 20px rgba(79, 158, 255, 0.24);
}

.about-name {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  color: var(--app-text, #303133);
  letter-spacing: 0.3px;
}

.about-desc {
  margin: 6px 0 16px;
  font-size: 13px;
  color: var(--app-text-muted, #909399);
}

.about-list {
  width: 100%;
  margin: 0 0 16px;
  padding: 0;
  border-top: 1px solid var(--app-border, #e4e7ed);
  text-align: left;
}

.about-row {
  display: flex;
  gap: 16px;
  padding: 8px 2px;
  border-bottom: 1px solid var(--app-border, #e4e7ed);
  font-size: 13px;
}

.about-row dt {
  flex-shrink: 0;
  width: 56px;
  color: var(--app-text-muted, #909399);
  font-weight: 600;
  margin: 0;
}

.about-value {
  margin: 0;
  color: var(--app-text, #303133);
  font-family: Consolas, 'SFMono-Regular', Menlo, monospace;
  word-break: break-all;
}

/* 版本行：版本号 + 检查更新按钮 + 状态文案 */
.ver-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.ver-num {
  font-weight: 700;
}

.check-btn {
  height: 24px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--app-active-text, #409eff);
  background: transparent;
  border: 1px solid var(--app-active-text, #409eff);
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.15s, opacity 0.15s;
}

.check-btn:hover:not(:disabled) {
  background: rgba(64, 158, 255, 0.08);
}

.check-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.update-state {
  font-family: 'Helvetica Neue', Arial, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}

.update-state.latest {
  color: #67c23a;
}

.update-state.available {
  color: var(--app-active-text, #409eff);
  font-weight: 700;
}

.update-state.error {
  color: #f56c6c;
}

.about-link {
  color: var(--app-active-text, #409eff);
  text-decoration: none;
}

.about-link:hover {
  text-decoration: underline;
}

/* 赞助作者 */
.about-reward {
  width: 100%;
  margin-bottom: 16px;
  padding-top: 14px;
}

.reward-title {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 700;
  color: var(--app-text, #303133);
}

.reward-qr {
  display: block;
  width: 180px;
  max-width: 100%;
  height: auto;
  margin: 0 auto 6px;
  border-radius: 10px;
  border: 1px solid var(--app-border, #e4e7ed);
  background: #fff;
}

.reward-tip {
  margin: 0;
  font-size: 12px;
  color: var(--app-text-muted, #909399);
}

.about-ok {
  width: 100%;
  height: 36px;
  font-size: 14px;
  color: #fff;
  background: var(--app-active-text, #409eff);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.about-ok:hover {
  opacity: 0.9;
}
</style>
