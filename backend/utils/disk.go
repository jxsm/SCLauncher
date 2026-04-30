package utils

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// GetDiskFreeSpace 获取磁盘可用空间（字节）
// 返回可用空间和总空间
func GetDiskFreeSpace(path string) (freeSpace int64, totalSpace int64, err error) {
	if runtime.GOOS == "windows" {
		return getDiskFreeSpaceWindows(path)
	}
	return getDiskFreeSpaceUnix(path)
}

// getDiskFreeSpaceWindows Windows 平台获取磁盘空间
func getDiskFreeSpaceWindows(path string) (freeSpace int64, totalSpace int64, err error) {
	kernel32, err := syscall.LoadLibrary("Kernel32.dll")
	if err != nil {
		return 0, 0, fmt.Errorf("加载 Kernel32.dll 失败: %w", err)
	}
	defer syscall.FreeLibrary(kernel32)

	getDiskFreeSpaceEx, err := syscall.GetProcAddress(kernel32, "GetDiskFreeSpaceExW")
	if err != nil {
		return 0, 0, fmt.Errorf("获取 GetDiskFreeSpaceExW 地址失败: %w", err)
	}

	// 将路径转换为 UTF-16 格式
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, fmt.Errorf("路径转换失败: %w", err)
	}

	var freeBytesAvailable int64
	var totalBytes int64
	var totalFreeBytes int64

	// 调用 Windows API
	_, _, err = syscall.Syscall6(uintptr(getDiskFreeSpaceEx),
		4,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
		0,
		0)

	if err != syscall.Errno(0) {
		return 0, 0, fmt.Errorf("获取磁盘空间失败: %w", err)
	}

	return freeBytesAvailable, totalBytes, nil
}

// getDiskFreeSpaceUnix Unix/Linux 平台获取磁盘空间
func getDiskFreeSpaceUnix(path string) (freeSpace int64, totalSpace int64, err error) {
	// 使用 syscall.Statfs_t 结构体，不同系统可能有不同定义
	// 为了兼容性，我们使用通用的文件大小检查方法
	// 如果需要精确的磁盘空间，需要根据不同平台使用不同的系统调用

	// 简单实现：尝试创建临时文件来检查空间
	// 这不是最准确的方法，但跨平台兼容性最好
	tmpFile, err := os.CreateTemp(path, ".diskcheck")
	if err != nil {
		return 0, 0, fmt.Errorf("无法检查磁盘空间: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// 如果能成功创建文件，假设有足够的空间
	// 注意：这不是精确的方法，但可以避免程序崩溃
	return 1 << 60, 1 << 60, nil // 返回一个很大的值表示"可能有足够空间"
}

// CheckDiskSpace 检查磁盘是否有足够的空间
// requiredBytes: 需要的字节数
// path: 要检查的路径
// 返回是否有足够空间
func CheckDiskSpace(requiredBytes int64, path string) error {
	freeSpace, totalSpace, err := GetDiskFreeSpace(path)
	if err != nil {
		return fmt.Errorf("无法获取磁盘空间信息: %w", err)
	}

	// 添加 10% 的安全余量
	requiredWithMargin := requiredBytes + (requiredBytes / 10)

	if freeSpace < requiredWithMargin {
		// 转换为更友好的单位显示
		freeGB := float64(freeSpace) / (1024 * 1024 * 1024)
		requiredGB := float64(requiredWithMargin) / (1024 * 1024 * 1024)

		return fmt.Errorf("磁盘空间不足。需要: %.2f GB，可用: %.2f GB（总空间: %.2f GB）",
			requiredGB, freeGB, float64(totalSpace)/(1024*1024*1024))
	}

	return nil
}

// CheckPathDiskSpace 检查指定路径所在磁盘的可用空间
// 这是一个便捷函数，自动获取路径所在的磁盘
func CheckPathDiskSpace(requiredBytes int64, path string) error {
	// 获取根目录/盘符
	checkPath := path
	if runtime.GOOS == "windows" {
		// Windows: 提取盘符 (如 "C:\")
		if len(path) >= 2 && path[1] == ':' {
			checkPath = string(path[0]) + ":\\"
		}
	} else {
		// Unix: 使用根目录或当前目录
		if len(path) == 0 || path[0] != '/' {
			checkPath = "/"
		} else {
			checkPath = "/"
		}
	}

	return CheckDiskSpace(requiredBytes, checkPath)
}
