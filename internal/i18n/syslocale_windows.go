//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

// LOCALE_NAME_MAX_LENGTH 是 GetUserDefaultLocaleName 缓冲区最大长度（含结尾 \x00）。
const localeNameMaxLength = 85

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleName   = kernel32.NewProc("GetUserDefaultLocaleName")
)

// detectSystemLanguage 通过 Win32 GetUserDefaultLocaleName 读取当前用户区域设置
// （形如 zh-CN / en-US），再映射为应用语言代码。
func detectSystemLanguage() (string, error) {
	buf := make([]uint16, localeNameMaxLength)
	ret, _, callErr := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return "", callErr
	}
	return localeToLang(syscall.UTF16ToString(buf)), nil
}
