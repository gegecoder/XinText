import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './styles/theme.css'
import appConfig from '@app-config'
import { LogService } from '../bindings/XinText/internal/service'

// 每次新版本（含首次运行）清空 localStorage，避免旧版本残留脏数据影响初始化
const INIT_VERSION_KEY = 'XinText.initialized.version'
if (localStorage.getItem(INIT_VERSION_KEY) !== appConfig.version) {
  localStorage.clear()
  localStorage.setItem(INIT_VERSION_KEY, appConfig.version)
}

// 拦截 console.log/info/warn/error，把前端日志异步写入后端 frontend.log。
// 保留原始行为（devtools 仍可见），失败用原始 console.error 输出方便调试。
;(['log', 'info', 'warn', 'error'] as const).forEach((level) => {
  const orig = console[level].bind(console)
  console[level] = (...args: any[]) => {
    orig(...args)
    const msg = args
      .map((a) => {
        if (a instanceof Error) return a.stack || a.message
        if (typeof a === 'object') {
          try { return JSON.stringify(a) } catch { return String(a) }
        }
        return String(a)
      })
      .join(' ')
    // 失败时用原始 console.error 输出（避免递归触发拦截器）
    LogService.WriteFrontend(level, msg).catch((e) => orig('WriteFrontend failed:', e))
  }
})

const app = createApp(App)
app.use(createPinia())
app.mount('#app')

// 主动写一条启动日志，让 frontend.log 不为空（也用于验证拦截器工作）
console.log('XinText 前端启动')
