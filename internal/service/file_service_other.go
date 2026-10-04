//go:build !windows

package service

import (
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"XinText/internal/runcmd"
)

// fileCreateTime 在非 Windows 平台返回 0：标准库不暴露文件创建时间，
// 属性弹窗会据此隐藏创建时间行。
func fileCreateTime(path string) int64 {
	return 0
}

// revealInFileManager 在系统文件管理器中显示并选中给定文件。
// macOS 用 open -R（Finder 中选中文本）；Linux 优先走 freedesktop
// FileManager1 DBus ShowItems 接口（Nautilus/Dolphin 等支持选中文件），
// 失败后回退 xdg-open 打开所在目录。
func revealInFileManager(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Start()
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	// 复合字面量不可寻址，不能直接在 url.URL{} 上调指针方法 String()，先取指针
	fileURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	args := []string{
		"--session", "--print-reply",
		"--dest=org.freedesktop.FileManager1",
		"/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowItems",
		`array:string:` + fileURL,
		`string:`,
	}
	if _, err := runcmd.Run(nil, 5*time.Second, "dbus-send", args...); err == nil {
		return nil
	}
	// DBus 不可用（极简桌面/无会话总线）：退化为打开所在目录
	return exec.Command("xdg-open", filepath.Dir(abs)).Start()
}
