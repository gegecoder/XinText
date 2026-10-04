import Vditor from 'vditor'
import { editorLangCode } from '../i18n'

/**
 * 导出 TXT 的前端实现（不依赖 pandoc）。
 *
 * 先用 Vditor.preview 把 markdown 静态渲染到离屏容器，再按 pandoc
 * `-t plain --wrap=none` 的规则把 DOM 提取为纯文本：
 *   - 块级元素之间空一行；
 *   - 标题/段落只保留文字，加粗/斜体/删除线/行内代码的标记全部剥掉；
 *   - 无序列表用 "• "、有序列表用真实序号，嵌套列表按标记宽度悬挂缩进；
 *   - 引用块与代码块整体缩进 4 空格；
 *   - 任务列表渲染 ☐/☑；
 *   - 表格用宽字符对齐的网格（表头下与表尾各一行短横线）；
 *   - 链接只保留文字、图片只保留 alt、水平线渲染为一行短横线；
 *   - KaTeX 公式取可视文本（跳过 MathML，避免内容重复）。
 */

/** displayWidth：宽字符（CJK/全角）按 2 计，其余按 1 计 */
function displayWidth(s: string): number {
  let w = 0
  for (const ch of s) {
    const cp = ch.codePointAt(0) ?? 0
    const wide =
      (cp >= 0x1100 && cp <= 0x115f) || // Hangul Jamo
      (cp >= 0x2e80 && cp <= 0x9fff) || // CJK 部首/假名/表意
      (cp >= 0xac00 && cp <= 0xd7a3) || // Hangul 音节
      (cp >= 0xf900 && cp <= 0xfaff) || // CJK 兼容表意
      (cp >= 0xff01 && cp <= 0xff60) || // 全角 ASCII
      (cp >= 0x20000 && cp <= 0x2fffd) // CJK 扩展
    w += wide ? 2 : 1
  }
  return w
}

/** 提取行内纯文本：剥掉所有格式标记；<br> 转为换行；图片取 alt；KaTeX 取可视文本 */
function inlineText(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) return node.textContent ?? ''
  if (node.nodeType !== Node.ELEMENT_NODE) return ''
  const el = node as HTMLElement
  // Vditor 代码块右上角的「复制」按钮、KaTeX 的 MathML 等不参与文本输出
  if (el.classList.contains('vditor-copy') || el.classList.contains('katex-mathml')) return ''
  if (el.tagName === 'BR') return '\n'
  if (el.tagName === 'IMG') return (el as HTMLImageElement).alt || ''
  if (el.tagName === 'INPUT') return '' // 任务列表 checkbox 在 li 层级单独处理
  if (el.classList.contains('katex')) {
    return el.querySelector('.katex-html')?.textContent ?? el.textContent ?? ''
  }
  let out = ''
  el.childNodes.forEach((c) => {
    out += inlineText(c)
  })
  return out
}

/** 块级元素选择器（用于判定 li 内是否存在子块） */
const BLOCK_SEL = 'p,ul,ol,blockquote,pre,table,hr,div'

/** 渲染一个块级元素为文本（不含块间空行，由调用方拼接） */
function renderBlock(el: HTMLElement, indent: string): string {
  const tag = el.tagName

  // 标题 / 段落：行内文本；内部换行的后续行同样补 indent
  if (/^H[1-6]$/.test(tag) || tag === 'P') {
    return indent + inlineText(el).split('\n').join('\n' + indent)
  }

  // 代码块：每行缩进 4 空格
  if (tag === 'PRE') {
    const code = el.querySelector('code')
    const text = (code ? code.textContent : el.textContent) ?? ''
    return text
      .replace(/\n$/, '')
      .split('\n')
      .map((line) => indent + '    ' + line)
      .join('\n')
  }

  // 引用块：子块整体缩进 4 空格
  if (tag === 'BLOCKQUOTE') {
    return renderChildren(el, indent + '    ')
  }

  // 列表
  if (tag === 'UL' || tag === 'OL') {
    const ordered = tag === 'OL'
    const startAttr = parseInt(el.getAttribute('start') || '1', 10)
    const start = Number.isNaN(startAttr) ? 1 : startAttr
    const chunks: string[] = []
    let idx = start
    el.childNodes.forEach((liNode) => {
      const li = liNode as HTMLElement
      if (li.nodeType !== Node.ELEMENT_NODE || li.tagName !== 'LI') return
      const marker = ordered ? `${idx}. ` : '• '
      idx++
      chunks.push(renderListItem(li, indent, marker))
    })
    return chunks.join('\n')
  }

  // 表格：宽字符对齐网格
  if (tag === 'TABLE') {
    return renderTable(el, indent)
  }

  // 水平线
  if (tag === 'HR') {
    return indent + '--------------------'
  }

  // 容器类元素：递归子块
  if (tag === 'DIV' || tag === 'SECTION' || tag === 'ARTICLE') {
    return renderChildren(el, indent)
  }

  // 兜底：按行内处理
  return indent + inlineText(el).split('\n').join('\n' + indent)
}

/** 渲染 li：首个内容块与标记同行，后续块按标记宽度悬挂缩进；任务列表前缀 ☐/☑ */
function renderListItem(li: HTMLElement, indent: string, marker: string): string {
  const taskInput = li.querySelector(':scope > input[type="checkbox"]') as HTMLInputElement | null
  const checkbox = taskInput ? (taskInput.checked ? '☑ ' : '☐ ') : ''
  const childIndent = indent + ' '.repeat(displayWidth(marker))

  // 收集 li 的块级子元素（松散列表的 <p>、嵌套列表、引用等）
  const blocks: HTMLElement[] = []
  li.childNodes.forEach((c) => {
    if (c.nodeType === Node.ELEMENT_NODE && (c as HTMLElement).matches(BLOCK_SEL)) {
      blocks.push(c as HTMLElement)
    }
  })

  let firstLine = checkbox
  let restStart = 0
  if (blocks.length > 0 && blocks[0].tagName === 'P') {
    // 松散列表：首个段落内容与标记同行
    firstLine += inlineText(blocks[0])
    restStart = 1
  } else {
    // 紧凑列表：标记后同行内内容（首个块级子元素之前的行内节点）
    const inlineNodes: Node[] = []
    for (const c of Array.from(li.childNodes)) {
      if (c.nodeType === Node.ELEMENT_NODE && (c as HTMLElement).matches(BLOCK_SEL)) break
      if (c.nodeType === Node.ELEMENT_NODE && (c as HTMLElement).tagName === 'INPUT') continue
      inlineNodes.push(c)
    }
    firstLine += inlineNodes.map(inlineText).join('')
  }

  const parts = [firstLine.replace(/\n/g, '\n' + childIndent)]
  for (let i = restStart; i < blocks.length; i++) {
    parts.push(renderBlock(blocks[i], childIndent))
  }
  return indent + marker + parts.join('\n')
}

/** 渲染子块序列，块间空一行 */
function renderChildren(el: HTMLElement, indent: string): string {
  const chunks: string[] = []
  el.childNodes.forEach((c) => {
    if (c.nodeType !== Node.ELEMENT_NODE) return
    chunks.push(renderBlock(c as HTMLElement, indent))
  })
  return chunks.filter((c) => c.trim() !== '').join('\n\n')
}

/** 渲染表格为 pandoc simple 风格网格 */
function renderTable(table: HTMLElement, indent: string): string {
  const header: string[] = []
  const rows: string[][] = []
  table.querySelectorAll('thead tr').forEach((tr) => {
    tr.querySelectorAll('th').forEach((th) => header.push(inlineText(th)))
  })
  table.querySelectorAll('tbody tr').forEach((tr) => {
    const row: string[] = []
    tr.querySelectorAll('td').forEach((td) => row.push(inlineText(td)))
    rows.push(row)
  })
  const colCount = Math.max(header.length, ...rows.map((r) => r.length), 0)
  if (colCount === 0) return ''

  const widths = new Array(colCount).fill(0)
  const measure = (r: string[]) => {
    for (let i = 0; i < colCount; i++) {
      widths[i] = Math.max(widths[i], displayWidth(r[i] ?? ''))
    }
  }
  measure(header)
  rows.forEach(measure)

  const writeRow = (cells: string[]) =>
    indent +
    Array.from({ length: colCount }, (_, i) => {
      const cell = cells[i] ?? ''
      return i < colCount - 1 ? cell + ' '.repeat(widths[i] - displayWidth(cell) + 2) : cell
    }).join('')
  const writeSep = () => indent + widths.map((w) => '-'.repeat(w)).join('  ')

  const lines: string[] = []
  if (header.length > 0) {
    lines.push(writeRow(header))
    lines.push(writeSep())
  } else {
    lines.push(writeSep())
  }
  rows.forEach((r) => lines.push(writeRow(r)))
  lines.push(writeSep())
  return lines.join('\n')
}

/**
 * 把 markdown 转换为纯文本（pandoc plain 风格），用于导出 TXT。
 * @param markdown 文档内容
 */
export async function renderMarkdownToText(markdown: string): Promise<string> {
  const root = document.createElement('div')
  // 渲染不需要上屏，但 Vditor 的代码高亮/公式渲染要求容器在文档中且具备尺寸，
  // 因此挂到 body 下并移出视口，用完即移除
  root.style.position = 'fixed'
  root.style.left = '-99999px'
  root.style.top = '0'
  root.style.width = '820px'
  root.style.visibility = 'hidden'
  document.body.appendChild(root)
  try {
    await Vditor.preview(root, markdown, {
      cdn: '/vditor',
      lang: editorLangCode() as 'zh_CN' | 'en_US',
      mode: 'light',
    })
    const text = renderChildren(root, '')
    return text.replace(/\n+$/, '') + (text.trim() === '' ? '' : '\n')
  } finally {
    root.remove()
  }
}
