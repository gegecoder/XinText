import Vditor from 'vditor'
import { resolveImagesIn } from './resolveImages'
import { editorLangCode } from '../i18n'

/**
 * 打印当前文档：用 Vditor 静态渲染 markdown 到一个离屏容器，
 * 解析本地图片为 data URL，再用 @media print CSS 隐藏主界面其余元素、
 * 仅显示渲染结果，最后调用 window.print() 触发 WebView2 打印对话框。
 *
 * 离屏容器位于 body 下，屏幕上不可见（left:-99999px）但参与正常渲染，
 * 保证 Vditor 代码高亮 / 公式 / 图片加载都能正常工作；打印时通过
 * `body > *:not(.XinText-print-root) { display:none }` 只保留正文。
 */
let printRoot: HTMLDivElement | null = null
let printStyleEl: HTMLStyleElement | null = null

const PRINT_CSS = `
.XinText-print-root {
  position: fixed;
  left: -99999px;
  top: 0;
  width: 820px;
  padding: 0;
  margin: 0;
  background: #fff;
  color: #303133;
  z-index: -1;
  pointer-events: none;
}
@media print {
  /* 隐藏整个应用界面，只保留打印容器 */
  body > *:not(.XinText-print-root) { display: none !important; }
  .XinText-print-root {
    position: static !important;
    left: auto !important;
    top: auto !important;
    width: 100% !important;
    z-index: 0 !important;
    pointer-events: auto !important;
  }
  @page { margin: 14mm 16mm; }
  /* 分页：标题不落在页底，代码块/表格/图片不被截断 */
  .XinText-print-root h1,
  .XinText-print-root h2,
  .XinText-print-root h3,
  .XinText-print-root h4,
  .XinText-print-root h5,
  .XinText-print-root h6 { page-break-after: avoid; break-after: avoid; }
  .XinText-print-root pre,
  .XinText-print-root table,
  .XinText-print-root img,
  .XinText-print-root blockquote { page-break-inside: avoid; break-inside: avoid; }
  .XinText-print-root pre { white-space: pre-wrap; word-wrap: break-word; }
  .XinText-print-root img { max-width: 100%; }
  .XinText-print-root table { width: 100%; border-collapse: collapse; }
  .XinText-print-root th,
  .XinText-print-root td { border: 1px solid #999; padding: 4px 8px; }
  /* 打印用纯黑文字，避免彩色链接在黑白打印机上偏淡 */
  .XinText-print-root a { color: #000; text-decoration: underline; }
}
`

function ensurePrintStyle() {
  if (printStyleEl && printStyleEl.isConnected) return
  const el = document.createElement('style')
  el.setAttribute('data-XinText-print', '')
  el.textContent = PRINT_CSS
  document.head.appendChild(el)
  printStyleEl = el
}

function ensurePrintRoot(): HTMLDivElement {
  if (printRoot && printRoot.isConnected) return printRoot
  const el = document.createElement('div')
  el.className = 'XinText-print-root'
  document.body.appendChild(el)
  printRoot = el
  return el
}

function waitForImages(root: HTMLElement): Promise<void> {
  const imgs = Array.from(root.querySelectorAll('img'))
  if (imgs.length === 0) return Promise.resolve()
  return Promise.all(
    imgs.map((img) => {
      if (img.complete) return Promise.resolve()
      return new Promise<void>((resolve) => {
        img.addEventListener('load', () => resolve(), { once: true })
        img.addEventListener('error', () => resolve(), { once: true })
      })
    })
  ).then(() => undefined)
}

/**
 * 从文档绝对路径推导打印默认文件名：取 basename 并去掉 .md 后缀。
 * WebView2 打印对话框「另存为 PDF」的默认文件名取自 document.title，
 * 所以打印前临时改写 title 即可让默认文件名跟随当前文档。
 */
function derivePrintName(docPath: string | null): string {
  if (!docPath) return 'document'
  const base = docPath.split(/[\\/]/).pop() || 'document'
  return base.replace(/\.md$/i, '') || 'document'
}

/**
 * 把 markdown 渲染为自包含 HTML 字符串：用 Vditor.preview 静态渲染到离屏容器，
 * 通过 resolveImagesIn 把相对路径图片替换为 data URL，等待图片解码完成。
 * 渲染产物同时保留在离屏容器中，供 printDocument 直接调用 window.print()。
 *
 * 后端 ExportPDFFromHTML 接收此 HTML 字符串，用 msedge headless 渲染为 PDF，
 * 因此前端打印路径与后端 PDF 导出路径共享同一份渲染产物，公式 / 代码高亮 /
 * 表格样式完全一致。
 *
 * @param markdown 文档内容
 * @param docPath 文档绝对路径（用于解析相对路径图片）；新建未保存文档传 null
 * @param theme 主题，与 ReadView 保持一致
 * @returns 渲染后的 HTML 字符串（含 KaTeX 公式 HTML、代码高亮 class、图片 data URL）
 */
export async function renderMarkdownToHTML(
  markdown: string,
  docPath: string | null,
  theme?: 'classic' | 'dark'
): Promise<string> {
  ensurePrintStyle()
  const root = ensurePrintRoot()
  // 每次渲染前清空旧内容，避免上一份文档残留
  root.innerHTML = ''
  const dark = theme === 'dark'
  await Vditor.preview(root, markdown, {
    cdn: '/vditor',
    lang: editorLangCode() as 'zh_CN' | 'en_US',
    mode: dark ? 'dark' : 'light',
    theme: { current: theme ?? 'classic' },
    hljs: { enable: true, style: dark ? 'github-dark' : 'github' },
  })
  // 把本地相对路径图片替换为 data URL，否则打印时图片为裂图
  await resolveImagesIn(root, docPath)
  // 等所有图片解码完成，避免打印时图片尚未加载
  await waitForImages(root)
  return root.innerHTML
}

/**
 * 渲染并打印 markdown 文档。
 * @param markdown 文档内容
 * @param docPath 文档绝对路径（用于解析相对路径图片）；新建未保存文档传 null
 * @param theme 主题，与 ReadView 保持一致
 */
export async function printDocument(
  markdown: string,
  docPath: string | null,
  theme?: 'classic' | 'dark'
): Promise<void> {
  // 共用渲染逻辑：渲染产物同时留在离屏容器中，window.print() 时通过
  // @media print CSS 隐藏主界面、仅显示 .XinText-print-root
  await renderMarkdownToHTML(markdown, docPath, theme)

  // 临时改写 document.title，使 WebView2 打印对话框（含「另存为 PDF」）默认文件名跟随当前文档
  const prevTitle = document.title
  document.title = derivePrintName(docPath)
  try {
    // 触发 WebView2 打印对话框（阻塞，用户关闭对话框后返回）
    window.focus()
    window.print()
  } finally {
    document.title = prevTitle
  }
}
