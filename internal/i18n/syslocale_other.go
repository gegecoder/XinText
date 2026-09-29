//go:build !windows

package i18n

import (
	"fmt"
	"os"
)

// detectSystemLanguage 在 macOS/Linux 上按标准 locale 环境变量推断系统语言；
// 全部缺失时返回错误，由调用方走默认语言兜底。
func detectSystemLanguage() (string, error) {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if raw := os.Getenv(key); raw != "" {
			// 形如 zh_CN.UTF-8 / en_US.UTF-8，去掉编码段后映射
			if dot := indexByte(raw, '.'); dot >= 0 {
				raw = raw[:dot]
			}
			if lang := localeToLang(raw); lang != "" {
				return lang, nil
			}
		}
	}
	return "", fmt.Errorf("system locale is not available")
}

// indexByte 避免为单次查找额外引入 strings 包（保持平台文件精简）。
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
