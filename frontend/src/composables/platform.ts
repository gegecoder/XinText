/**
 * 平台检测与快捷键标签转换。
 *
 * 快捷键处理逻辑（App.vue onWebviewKeydown）已用 ctrlKey || metaKey 同时兼容
 * macOS Cmd 与 Windows/Linux Ctrl；这里只负责「显示层」：macOS 用户看到的是
 * ⌘/⇧/⌥ 符号而非 Ctrl/Shift/Alt 文字。
 */

/** 是否 macOS（WebView 环境，userAgent 含 Mac 即视为 macOS） */
export const isMac =
  typeof navigator !== 'undefined' && /macintosh|mac os x/i.test(navigator.userAgent)

/**
 * 把「Ctrl+Shift+N」形式的快捷键描述转换为当前平台的显示标签：
 * - macOS：Ctrl→⌘、Shift→⇧、Alt→⌥（mac 惯例符号后直接跟键名，无 + 号）
 * - 其他平台：原样返回
 *
 * 也可直接包裹整段含快捷键的文案（如 i18n 的 tooltip），只转换其中的组合键片段。
 */
export function shortcutLabel(shortcut: string): string {
  if (!isMac) return shortcut
  return shortcut
    .replace(/Ctrl\+/g, '⌘')
    .replace(/Shift\+/g, '⇧')
    .replace(/Alt\+/g, '⌥')
}
