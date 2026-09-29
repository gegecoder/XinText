import { ImageService } from '../../bindings/XinText/internal/service'

/**
 * Markdown 中相对路径图片（如 assets/xxx.png）在 WebView 里按页面 URL 解析
 * 会 404；此模块在渲染层把这类 <img> 替换为 Go 读出的 data URL。
 * md 源文件保持相对路径不变（pandoc 导出 / 其他编辑器均可正常解析）。
 */

// 绝对路径 → data URL 缓存，避免同一图片重复 IPC
const dataUrlCache = new Map<string, string>()

/** 判断 src 是否为需要解析的本地图片路径（相对路径或 Windows 绝对路径） */
function isLocalRel(src: string): boolean {
  if (!src) return false
  if (/^(https?:|data:|blob:|file:|wails:|#|\/\/)/i.test(src)) return false
  if (src.startsWith('/')) return false // 站点根路径
  return true
}

/** 规范化为正斜杠路径 */
function toSlash(p: string): string {
  return p.replace(/\\/g, '/')
}

/** 判断 src 是否为 Windows 绝对路径（D:/... 或 D:\...） */
function isWindowsAbs(src: string): boolean {
  return /^[a-zA-Z]:[\\/]/.test(src)
}

/** 图片 src + 文档绝对路径 → 图片绝对路径（支持 ./ ../ 与 Windows 绝对路径） */
function toAbs(docPath: string, src: string): string {
  const s = toSlash(src)
  // Windows 绝对路径（D:/... 或 D:\...）直接使用
  if (isWindowsAbs(s)) return s
  const dirSegs = toSlash(docPath).split('/').slice(0, -1)
  for (const part of s.split('/')) {
    if (part === '..') dirSegs.pop()
    else if (part !== '' && part !== '.') dirSegs.push(part)
  }
  return dirSegs.join('/')
}

/**
 * 扫描 root 下的 <img>，把本地相对路径 src 替换为 data URL。
 * 已处理的 img 会打 data-local-resolved 标记，重复调用直接跳过。
 *
 * 绝对路径图片（如粘贴后插入的 D:/.../xxx.png）无需 docPath 即可解析；
 * 相对路径图片需要 docPath 推算目录，docPath 为空时跳过。
 */
export async function resolveImagesIn(
  root: HTMLElement | null | undefined,
  docPath: string | null | undefined
) {
  if (!root) return
  const imgs = root.querySelectorAll('img[src]')
  for (const img of Array.from(imgs)) {
    const el = img as HTMLImageElement
    const src = el.getAttribute('src') || ''
    if (el.dataset.localResolved === '1' || !isLocalRel(src)) continue
    // 绝对路径无需 docPath；相对路径需要 docPath，缺失时跳过
    const abs = isWindowsAbs(src)
    if (!abs && !docPath) continue
    el.dataset.localResolved = '1'
    const absPath = abs ? toSlash(src) : toAbs(docPath!, src)
    try {
      let data = dataUrlCache.get(absPath)
      if (!data) {
        data = await ImageService.ReadFileBase64(absPath)
        dataUrlCache.set(absPath, data)
      }
      if (el.isConnected) el.src = data
    } catch {
      // 图片文件缺失：保留原样（显示为裂图），不阻塞其余图片
    }
  }
}
