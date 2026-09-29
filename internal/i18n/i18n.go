// Package i18n 维护应用语言代码、系统语言探测以及 Go 侧原生 UI
// （托盘菜单、关闭确认框）的多语言文案。
//
// 语言约定：空字符串表示「跟随系统」，持久化配置中用户未显式选择时即为空；
// 前端与后端统一通过 ResolveLanguage 解析为具体语言代码。
package i18n

import "strings"

// 支持的语言代码（BCP-47 简化形式，前端消息目录同名）。
const (
	LangZh       = "zh" // 简体中文
	LangEn       = "en" // English
	FallbackLang = LangZh
)

// Supported 判断 lang 是否为已支持的显式语言代码。
func Supported(lang string) bool {
	return lang == LangZh || lang == LangEn
}

// localeToLang 把系统 locale（如 zh-CN / zh-Hans / en-US / fr_FR）映射为
// 应用语言代码：中文语种 → zh；其他语种 → en；空串 → ""（交由上层兜底）。
func localeToLang(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" {
		return ""
	}
	if strings.HasPrefix(locale, LangZh) {
		return LangZh
	}
	return LangEn
}

// ResolveLanguage 解析最终生效的语言：
// 已持久化的显式选择优先；空值或非法值回退到系统语言；
// 系统语言获取不到时默认中文。
func ResolveLanguage(lang string) string {
	if Supported(lang) {
		return lang
	}
	sys, err := detectSystemLanguage()
	if err == nil && Supported(sys) {
		return sys
	}
	return FallbackLang
}

// NativeText 是 Go 侧原生界面使用的文案集合。
type NativeText struct {
	// 托盘菜单
	ShowMainWindow string
	Quit           string
	// 关闭确认框标题（使用时与应用名拼接：title + " " + appName）
	CloseTitle string
	// 关闭确认框正文
	CloseMessage string
}

var nativeCatalog = map[string]NativeText{
	LangZh: {
		ShowMainWindow: "显示主窗口",
		Quit:           "退出",
		CloseTitle:     "关闭",
		CloseMessage: "关闭窗口后，是否最小化到系统托盘继续在后台运行？\n\n" +
			"「是」最小化到托盘，「否」直接退出程序。",
	},
	LangEn: {
		ShowMainWindow: "Show Main Window",
		Quit:           "Quit",
		CloseTitle:     "Close",
		CloseMessage: "After closing the window, minimize it to the system tray " +
			"and keep running in the background?\n\n" +
			"\"Yes\" minimizes to tray, \"No\" quits the application.",
	},
}

// Native 返回指定语言的原生 UI 文案；未知语言按兜底语言处理。
func Native(lang string) NativeText {
	return nativeCatalog[ResolveLanguage(lang)]
}
