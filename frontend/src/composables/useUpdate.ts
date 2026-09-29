// 版本更新检测组合式函数：模块级单例状态，关于弹窗检查更新与发现新版本弹窗共用
import { ref } from 'vue'
import { UpdateService } from '../../bindings/XinText/internal/service'
import appConfig from '@app-config'
import { i18n } from '../i18n'

export interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  hasUpdate: boolean
  prerelease: boolean
  releaseName: string
  releaseNotes: string
  publishedAt: string
  releaseUrl: string
  downloadUrl: string
  msg: string
}

// 模块级单例状态
const appVersion = ref(appConfig.version)
const updateInfo = ref<UpdateInfo | null>(null)
const checking = ref(false)
const showUpdateModal = ref(false)
// idle / checking / latest / available / error
const checkStatus = ref<'idle' | 'checking' | 'latest' | 'available' | 'error'>('idle')
const checkMessage = ref('')

// 检测更新：结果回写 checkStatus/checkMessage，发现新版本时弹出更新窗口
export async function checkForUpdate(): Promise<UpdateInfo | null> {
  if (checking.value) return updateInfo.value
  checking.value = true
  checkStatus.value = 'checking'
  checkMessage.value = i18n('update.checking')
  try {
    const res = (await UpdateService.CheckForUpdate()) as unknown as UpdateInfo
    updateInfo.value = res
    if (res && res.msg === 'ok') {
      if (res.hasUpdate) {
        checkStatus.value = 'available'
        checkMessage.value = i18n('update.foundNew', { version: res.latestVersion })
        showUpdateModal.value = true
      } else {
        checkStatus.value = 'latest'
        checkMessage.value = i18n('update.latest')
      }
    } else {
      checkStatus.value = 'error'
      checkMessage.value = (res && res.msg) || i18n('update.checkFailed')
    }
    return res
  } catch (e: unknown) {
    checkStatus.value = 'error'
    checkMessage.value = i18n('update.checkFailedDetail', { reason: e instanceof Error ? e.message : String(e) })
    return null
  } finally {
    checking.value = false
  }
}

export function useUpdate() {
  return {
    appVersion,
    updateInfo,
    checking,
    showUpdateModal,
    checkStatus,
    checkMessage,
    checkForUpdate
  }
}
