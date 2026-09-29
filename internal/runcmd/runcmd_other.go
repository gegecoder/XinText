//go:build !windows

// Package runcmd 非 Windows 平台：无需隐藏控制台窗口，直接透传 exec。
package runcmd

import (
	"context"
	"os/exec"
	"time"
)

// Command 创建 exec.Cmd（非 Windows 无控制台窗口问题）。
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// Run 执行命令并返回 stdout。timeout > 0 时超时自动终止。
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
