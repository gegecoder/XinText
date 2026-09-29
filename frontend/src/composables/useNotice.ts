import { useNoticeStore } from '../store/notice'

/**
 * 全局提示：任意组件调用
 *   const { showNotice } = useNotice()
 *   showNotice('保存成功')
 * NoticeDialog 由 App.vue 单例挂载，无需重复声明弹窗与状态。
 */
export function useNotice() {
  const noticeStore = useNoticeStore()
  return {
    showNotice: (msg: string) => noticeStore.show(msg)
  }
}
