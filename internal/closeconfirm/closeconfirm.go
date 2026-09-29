// Package closeconfirm 提供关闭窗口时的用户确认对话框：
// 最小化到托盘 / 直接退出 / 取消。
package closeconfirm

// 关闭确认对话框的语义动作（跨平台共用，与平台实现文件分离）
const (
	ActionMinimize = "minimize" // 最小化到托盘
	ActionQuit     = "quit"     // 直接退出
	ActionCancel   = "cancel"   // 取消，保持窗口
)
