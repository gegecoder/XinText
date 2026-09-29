import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * 全局提示弹窗状态：任意组件调用 showNotice()（见 composables/useNotice）
 * 即可弹出由 App.vue 单例挂载的 NoticeDialog，无需各自维护 visible/message。
 */
export const useNoticeStore = defineStore('notice', () => {
  const visible = ref(false)
  const message = ref('')

  function show(msg: string) {
    message.value = msg
    visible.value = true
  }

  function close() {
    visible.value = false
  }

  return { visible, message, show, close }
})
