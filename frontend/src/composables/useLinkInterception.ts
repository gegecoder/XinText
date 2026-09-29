import type { Ref } from 'vue'
import { FileService } from '../../bindings/XinText/internal/service'

/** 链接分类结果 */
export type LinkKind =
  | { kind: 'md'; path: string } // 本地存在的 .md 文件绝对路径
  | { kind: 'web'; url: string } // http/https 网页链接
  | { kind: 'notFound'; href: string } // 非 md 文件、非标准链接或本地不存在的链接 → 404 兜底
  | { kind: 'skip' } // mailto / anchor：交给默认行为，不拦截

export interface LinkCallbacks {
  /** 命中本地存在的 .md 文件 */
  onOpenMd: (path: string) => void
  /** 命中 web 链接，或需要用浏览器标签页兜底显示（如 404） */
  onOpenBrowser: (url: string) => void
}

const WEB_RE = /^(https?:\/\/|\/\/|www\.)/i
const MAILTO_RE = /^mailto:/i
const ANCHOR_RE = /^#/

/**
 * 判断 href 是否为 web 链接（http/https/protocol-relative/www.xxx）。
 * mailto: / tel: / #anchor 等不算 web 链接。
 */
export function isWebUrl(href: string): boolean {
  if (!href) return false
  if (MAILTO_RE.test(href)) return false
  if (ANCHOR_RE.test(href)) return false
  return WEB_RE.test(href)
}

/** 将 www.xxx 补全为 http://www.xxx，其余原样返回 */
function normalizeWebUrl(href: string): string {
  if (/^www\./i.test(href)) return 'http://' + href
  return href
}

/**
 * 规范化容器内所有 <a> 的 href：
 * www.xxx → http://www.xxx
 * 这样浏览器原生右键菜单"在新窗口中打开"也能得到正确的绝对 URL。
 * 只改 www. 开头的 href，已规范化的 http://www. 不会重复匹配。
 */
function normalizeAnchorHrefs(root: HTMLElement) {
  const anchors = root.querySelectorAll('a[href]')
  anchors.forEach((a) => {
    const href = a.getAttribute('href') || ''
    if (/^www\./i.test(href)) {
      a.setAttribute('href', 'http://' + href)
    }
  })
}

/**
 * 内部 404 兜底 URL 协议：BrowserView 识别此协议后显示友好 404 界面而非加载 iframe。
 * 把原始 href 作为 query 参数附带，便于在 404 界面提示用户。
 */
export const NOT_FOUND_URL = (href: string) =>
  'XinText:notfound?href=' + encodeURIComponent(href)

/**
 * 把相对路径解析到当前文档所在目录的绝对路径。
 * - 已是绝对路径（Windows 形如 C:\ 或 /）则原样返回
 * - 否则拼接 dir + href
 */
async function resolveLocalPath(href: string, docPath: string | null): Promise<string | null> {
  if (!href) return null
  // 去掉 #anchor 与 ?query
  const clean = href.split('#')[0].split('?')[0]
  if (!clean) return null

  // Windows 绝对路径 (C:\) 或 unix 绝对路径 (/)
  if (/^[A-Za-z]:[\\/]/.test(clean) || clean.startsWith('/')) {
    return clean
  }
  if (!docPath) return null
  // 取文档所在目录
  const norm = docPath.replace(/\\/g, '/')
  const dir = norm.includes('/') ? norm.slice(0, norm.lastIndexOf('/')) : ''
  const joined = await FileService.JoinPath(dir, clean.replace(/\//g, '\\'))
  return joined
}

/**
 * 检测 href 是否"看起来像被 lute 错误去掉反斜杠的 Windows 绝对路径"。
 * 形如 `C:UsersadminDownloadstesttest.en.md`：盘符+冒号紧跟字母、整段无 `\\` 或 `/`。
 * CommonMark 规范要求 lute 在 link destination 中保留 backslash 后非标点字符，
 * 但 lute 实现可能不一致，会把 `\U` 当转义处理后丢弃反斜杠。
 */
function looksLikeStrippedWindowsPath(href: string): boolean {
  if (!/^[A-Za-z]:/.test(href)) return false
  const rest = href.slice(2)
  // 盘符后必须紧跟非分隔符的字母/数字（说明反斜杠被去掉了）
  if (!/^[A-Za-z0-9_]/.test(rest)) return false
  // 整段不能包含任何路径分隔符
  return !/[\\/]/.test(rest)
}

/**
 * 从 markdown 源码中查找匹配的 link destination：
 * 在源码里搜索形如 `](destination)` 的片段，如果 destination 去掉所有反斜杠后等于
 * 当前 href，则返回 destination 原文（保留反斜杠）。
 *
 * 这用于恢复 lute 渲染时丢失的反斜杠，让 `C:\Users\admin\test.md` 写法的链接也能打开。
 */
function recoverDestinationFromSource(href: string, source: string): string | null {
  if (!source) return null
  if (!looksLikeStrippedWindowsPath(href)) return null
  // 匹配 ]( ... ) 形式（不跨行；<> 包裹也支持）
  const re = /\]\((<[^>\n]*>|[^)\n]*)\)/g
  let m: RegExpExecArray | null
  while ((m = re.exec(source)) !== null) {
    let dest = m[1]
    if (dest.startsWith('<') && dest.endsWith('>')) {
      dest = dest.slice(1, -1)
    }
    // 去掉所有反斜杠后比较
    const stripped = dest.replace(/\\/g, '')
    if (stripped === href) return dest
  }
  return null
}

/**
 * 对单个 href 做分类：
 * 1. web 链接 → web
 * 2. mailto / anchor → skip（交默认行为，不拦截）
 * 3. 以 .md 结尾 + 本地存在 → md
 * 4. 其他（.md 不存在 / 非 .md 相对路径 / 不可识别链接）→ notFound（用浏览器标签页 404 兜底）
 *
 * sourceMarkdown 可选：当 lute 把 link destination 中的 Windows 反斜杠路径
 * 错误地渲染成"无反斜杠的盘符串"时（如 `C:Users...md`），从源码恢复原始 destination。
 */
export async function classifyLink(
  href: string,
  docPath: string | null,
  sourceMarkdown?: string
): Promise<LinkKind> {
  if (!href) return { kind: 'skip' }
  if (isWebUrl(href)) return { kind: 'web', url: normalizeWebUrl(href) }
  if (MAILTO_RE.test(href) || ANCHOR_RE.test(href)) return { kind: 'skip' }

  // 如果 href 看起来像被截断的 Windows 路径，从 markdown 源码恢复原始 destination
  let effectiveHref = href
  if (sourceMarkdown) {
    const recovered = recoverDestinationFromSource(href, sourceMarkdown)
    if (recovered) effectiveHref = recovered
  }

  // 非 .md 结尾的链接：直接走 404 兜底（不再交给浏览器默认行为，
  // 避免 WebView2 在 about:blank 触发系统级 404 / 下载等）
  if (!/\.md$/i.test(effectiveHref)) {
    return { kind: 'notFound', href }
  }

  const abs = await resolveLocalPath(effectiveHref, docPath)
  if (!abs) return { kind: 'notFound', href }
  try {
    const exists = await FileService.PathExists(abs)
    if (!exists) return { kind: 'notFound', href }
    const isDir = await FileService.IsDirectory(abs)
    if (isDir) return { kind: 'notFound', href }
    return { kind: 'md', path: abs }
  } catch {
    return { kind: 'notFound', href }
  }
}

/**
 * 同步预判：href 是否「看起来像可拦截的链接」。
 * - web 链接（http/https/protocol-relative）→ 拦截
 * - mailto / anchor → 不拦截（交给默认行为，避免破坏邮件/锚点跳转）
 * - 其他任何非空链接（含 .md、.txt、相对路径、绝对路径）→ 拦截
 *   异步分类时再决定走 md / notFound；notFound 会用浏览器标签页 404 兜底
 *
 * 这一步必须同步完成：事件 handler 中的 preventDefault 必须在事件循环
 * 同步阶段调用，否则浏览器默认行为（相对路径会尝试导航 → 系统级 404）已发生。
 */
export function looksInterceptable(href: string): boolean {
  if (!href) return false
  if (MAILTO_RE.test(href)) return false
  if (ANCHOR_RE.test(href)) return false
  // 去掉 #anchor / ?query 后非空就拦截
  const clean = href.split('#')[0].split('?')[0]
  return !!clean
}

/**
 * 在指定容器上挂载 capture-phase 的 click 委托监听器，
 * 命中 <a> 时阻止默认行为并按分类结果回调。
 *
 * 用法：
 *   const cleanup = mountLinkInterception(elRef, docPathRef, {
 *     onOpenMd: path => editorStore.openFile(path),
 *     onOpenBrowser: url => editorStore.openBrowserTab(url),
 *   }, markdownRef)
 *
 * 监听器只在挂载期间生效；docPath 变化无需重新挂载，
 * 因为每次 click 都会实时读取 docPathRef.value。
 *
 * sourceMarkdown 用于从 markdown 源码恢复 lute 渲染时丢失的反斜杠，
 * 让 `C:\Users\admin\test.md` 写法的链接也能正常打开。
 *
 * 重要：preventDefault 必须在事件同步阶段调用，浏览器默认行为
 * （相对 URL 导航 / 404）一旦发生就不可逆。所以策略是：
 *   1. 同步预判 href 看起来可拦截 → preventDefault + stopPropagation
 *   2. 再异步分类（PathExists 校验本地 .md 是否存在）
 *   3. 分类结果决定调用 onOpenMd / onOpenBrowser；若分类为 skip
 *      （如本地 .md 不存在），默认行为已被阻止，记录但不导航
 *
 * @returns 卸载函数，用于在 onUnmounted 或 watcher cleanup 中调用
 */
export function mountLinkInterception(
  target: Ref<HTMLElement | null>,
  docPath: Ref<string | null>,
  cb: LinkCallbacks,
  sourceMarkdown?: Ref<string>
): () => void {
  // 判断 mousedown/click 的 target 是否落在 <a> 上且需要拦截
  const findInterceptableAnchor = (e: MouseEvent): HTMLAnchorElement | null => {
    if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return null
    const target_ = e.target as HTMLElement | null
    const anchor = target_?.closest('a') as HTMLAnchorElement | null
    if (!anchor) return null
    const href = anchor.getAttribute('href') || ''
    if (!href || !looksInterceptable(href)) return null
    return anchor
  }

  // mousedown：在 contenteditable（IR 模式）中，阻止浏览器默认的光标定位行为，
  // 否则 click 事件还没触发，光标就已经移入链接文本，编辑器接管了交互。
  // preventDefault 不影响后续 click 事件的触发。
  const mousedownHandler = (e: MouseEvent) => {
    const anchor = findInterceptableAnchor(e)
    if (!anchor) return
    e.preventDefault()
    e.stopPropagation()
  }

  // contextmenu：右键链接时阻止原生菜单（避免"在新窗口中打开"使用相对路径）
  const contextmenuHandler = (e: MouseEvent) => {
    const target_ = e.target as HTMLElement | null
    const anchor = target_?.closest('a') as HTMLAnchorElement | null
    if (anchor && anchor.getAttribute('href')) {
      e.preventDefault()
    }
  }

  const clickHandler = (e: MouseEvent) => {
    const anchor = findInterceptableAnchor(e)
    if (!anchor) return
    const href = anchor.getAttribute('href') || ''

    // 必须在事件同步阶段阻止默认行为，否则相对 URL 会被浏览器尝试导航 → 404
    e.preventDefault()
    e.stopPropagation()

    // 异步分类：web 链接直接打开；.md 链接校验本地存在性后再决定
    classifyLink(href, docPath.value, sourceMarkdown?.value).then((kind) => {
      if (kind.kind === 'md') {
        cb.onOpenMd(kind.path)
      } else if (kind.kind === 'web') {
        cb.onOpenBrowser(kind.url)
      } else if (kind.kind === 'notFound') {
        cb.onOpenBrowser(NOT_FOUND_URL(kind.href))
      }
    })
  }

  let attached: HTMLElement | null = null
  let hrefObserver: MutationObserver | null = null
  const bind = () => {
    if (attached === target.value) return
    if (attached) {
      attached.removeEventListener('click', clickHandler, true)
      attached.removeEventListener('mousedown', mousedownHandler, true)
      attached.removeEventListener('contextmenu', contextmenuHandler, true)
      attached = null
    }
    if (hrefObserver) {
      hrefObserver.disconnect()
      hrefObserver = null
    }
    if (target.value) {
      target.value.addEventListener('click', clickHandler, true)
      target.value.addEventListener('mousedown', mousedownHandler, true)
      target.value.addEventListener('contextmenu', contextmenuHandler, true)
      attached = target.value
      // Vditor 渲染后规范化 <a> href，让右键"在新窗口中打开"也能得到正确的绝对 URL
      normalizeAnchorHrefs(target.value)
      hrefObserver = new MutationObserver(() => {
        if (target.value) normalizeAnchorHrefs(target.value)
      })
      hrefObserver.observe(target.value, { childList: true, subtree: true })
    }
  }
  bind()

  return () => {
    if (attached) {
      attached.removeEventListener('click', clickHandler, true)
      attached.removeEventListener('mousedown', mousedownHandler, true)
      attached.removeEventListener('contextmenu', contextmenuHandler, true)
      attached = null
    }
    if (hrefObserver) {
      hrefObserver.disconnect()
      hrefObserver = null
    }
  }
}
