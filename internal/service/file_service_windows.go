//go:build windows

package service

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetFileAttributesExW = kernel32.NewProc("GetFileAttributesExW")
)

// winFiletime 对应 Win32 FILETIME：自 1601-01-01 起的 100ns 计数。
type winFiletime struct {
	lowDateTime  uint32
	highDateTime uint32
}

// winFileAttributeData 对应 WIN32_FILE_ATTRIBUTE_DATA，仅取需要的字段。
type winFileAttributeData struct {
	dwFileAttributes uint32
	ftCreationTime   winFiletime
	ftLastAccessTime winFiletime
	ftLastWriteTime  winFiletime
	nFileSizeHigh    uint32
	nFileSizeLow     uint32
}

// fileCreateTime 返回文件创建时间（Unix 秒）。
// 通过 GetFileAttributesExW 读取 WIN32_FILE_ATTRIBUTE_DATA.ftCreationTime，
// 再把 FILETIME（1601 起 100ns）换算为 1970 起 Unix 秒。取不到时返回 0。
func fileCreateTime(path string) int64 {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var data winFileAttributeData
	ret, _, _ := procGetFileAttributesExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0, // GetFileExInfoStandard
		uintptr(unsafe.Pointer(&data)),
	)
	if ret == 0 {
		return 0
	}
	ft := int64(data.ftCreationTime.highDateTime)<<32 | int64(data.ftCreationTime.lowDateTime)
	if ft <= 0 {
		return 0
	}
	// 1601-01-01 到 1970-01-01 的 100ns 计数：116444736000000000
	return (ft - 116444736000000000) / 10000000
}
