import { defineStore } from 'pinia'
import { ConfigService } from '../../bindings/XinText/internal/service'
import type { Config } from '../../bindings/XinText/internal/model/models'

const DEFAULT_CONFIG: Config = {
  windowWidth: 1200,
  windowHeight: 800,
  theme: 'light',
  language: '',
  recentFiles: [],
  recentFolders: [],
  projectPath: '',
  pandocPath: '',
  imageDir: '',
  logDir: '',
  recycleDir: '',
  browserHomeURL: 'https://kevin.blog.csdn.net',
  autoSave: false
}

export const usePreferencesStore = defineStore('preferences', {
  state: () => ({
    config: { ...DEFAULT_CONFIG } as Config,
    loaded: false
  }),
  actions: {
    async load() {
      try {
        const cfg = await ConfigService.GetConfig()
        // autoSave 后端以 *bool 返回，null（未设置）兜底为默认关闭
        this.config = { ...DEFAULT_CONFIG, ...cfg, autoSave: cfg.autoSave ?? false }
      } catch (e) {
        console.error('failed to load config', e)
        this.config = { ...DEFAULT_CONFIG }
      }
      this.loaded = true
    },
    async save() {
      await ConfigService.SetConfig(this.config)
    },
    async setWindowSize(width: number, height: number) {
      this.config.windowWidth = width
      this.config.windowHeight = height
      await this.save()
    },
    async setProjectPath(path: string) {
      this.config.projectPath = path
      await this.save()
    },
    async setTheme(theme: string) {
      this.config.theme = theme
      await this.save()
    },
    /** 持久化界面语言（zh / en）；界面切换由调用方通过 i18n.setLocale 完成 */
    async setLanguage(language: string) {
      this.config.language = language
      await this.save()
    },
    async setPandocPath(path: string) {
      this.config.pandocPath = path
      await this.save()
    },
    async setImageDir(path: string) {
      this.config.imageDir = path
      await this.save()
    },
    async setLogDir(path: string) {
      this.config.logDir = path
      await this.save()
    },
    async setRecycleDir(path: string) {
      this.config.recycleDir = path
      await this.save()
    },
    /** 持久化浏览器标签页主页 URL（点击主页按钮 / 搜索兜底都使用此值） */
    async setBrowserHomeURL(url: string) {
      this.config.browserHomeURL = url
      await this.save()
    },
    /** 持久化自动保存开关（true=开启编辑后 10s 自动写盘） */
    async setAutoSave(enabled: boolean) {
      this.config.autoSave = enabled
      await this.save()
    },
    async addRecentFile(path: string) {
      await ConfigService.AddRecentFile(path)
      await this.load()
    },
    async addRecentFolder(path: string) {
      await ConfigService.AddRecentFolder(path)
      await this.load()
    },
    async removeRecentFolder(path: string) {
      await ConfigService.RemoveRecentFolder(path)
      await this.load()
    },
    async clearRecentFolders() {
      await ConfigService.ClearRecentFolders()
      await this.load()
    },
    async pathExists(path: string): Promise<boolean> {
      return await ConfigService.PathExists(path)
    }
  }
})
