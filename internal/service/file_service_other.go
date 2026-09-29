//go:build !windows

package service

// fileCreateTime 在非 Windows 平台返回 0：标准库不暴露文件创建时间，
// 属性弹窗会据此隐藏创建时间行。
func fileCreateTime(path string) int64 {
	return 0
}
