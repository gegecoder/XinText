//go:build !windows

package closeconfirm

// AskCloseAction 非 Windows 平台没有「最小化到托盘」需求，
// 关闭窗口即直接退出（macOS/Linux 走各自默认关闭行为）。
// lang 参数仅为与 Windows 平台签名保持一致，此处不使用。
func AskCloseAction(_ string, _ string) string {
	return ActionQuit
}
