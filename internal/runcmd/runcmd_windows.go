//go:build windows

// Package runcmd 安全执行外部命令：通过 HideWindow + CREATE_NO_WINDOW
// 隐藏控制台窗口，防止 pandoc/msedge 等子进程导致 cmd 黑窗闪烁。
package runcmd

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// Command 创建一个隐藏控制台窗口的 exec.Cmd，调用方可继续定制
// Dir/Stdin/Stderr 等字段（需自行处理超时，推荐传入带超时的 ctx）。
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}

// Run 执行命令并返回 stdout。timeout > 0 时超时自动终止（兜底防止卡死）。
// ctx 为 nil 时使用 context.Background()。
func Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return Command(ctx, name, args...).Output()
}
