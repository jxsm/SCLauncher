package utils

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// ValidatePath 验证路径是否在目标目录内（防止路径遍历攻击）
// basePath: 允许的基础路径
// targetPath: 要验证的目标路径
// 返回清理后的安全路径或错误
func ValidatePath(basePath, targetPath string) (string, error) {
	// 先将路径转换为绝对路径
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("无法获取基础路径的绝对路径: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("无法获取目标路径的绝对路径: %w", err)
	}

	// 检查目标路径是否以基础路径为前缀
	// 使用 filepath.Rel 检查相对路径
	relPath, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return "", fmt.Errorf("路径验证失败: %w", err)
	}

	// 检查相对路径是否以 ".." 开头（表示路径遍历）
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("安全错误：路径遍历攻击检测到（目标路径在基础目录之外）: %s", targetPath)
	}

	// 检查路径中是否包含 ".." 组件（Windows 和 Unix 都需要检查）
	if runtime.GOOS == "windows" {
		// Windows 上还要检查类似 "..\\" 的情况
		pathParts := strings.Split(filepath.Clean(targetPath), string(filepath.Separator))
		for _, part := range pathParts {
			if part == ".." {
				return "", fmt.Errorf("安全错误：路径包含父目录引用: %s", targetPath)
			}
		}
	} else {
		// Unix/Linux
		if strings.Contains(targetPath, "../") {
			return "", fmt.Errorf("安全错误：路径包含父目录引用: %s", targetPath)
		}
	}

	// 确保最终的绝对路径确实在基础路径内
	if !strings.HasPrefix(absTarget+string(filepath.Separator), absBase+string(filepath.Separator)) {
		if absTarget != absBase {
			return "", fmt.Errorf("安全错误：目标路径不在基础目录内\n基础: %s\n目标: %s", absBase, absTarget)
		}
	}

	return absTarget, nil
}

// SanitizeFilename 清理文件名，移除危险字符
// 保留字母、数字、中文、下划线、连字符、点和空格
func SanitizeFilename(filename string) string {
	// 移除路径分隔符和危险字符
	dangerousChars := []string{
		"..", "\\", "/", ":", "*", "?", "\"", "<", ">", "|",
		"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07",
		"\x08", "\x09", "\x0a", "\x0b", "\x0c", "\x0d", "\x0e", "\x0f",
	}

	sanitized := filename
	for _, char := range dangerousChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// 移除首尾空格和点
	sanitized = strings.Trim(sanitized, " .")
	// 如果文件名为空，使用默认名称
	if sanitized == "" {
		sanitized = "file"
	}

	return sanitized
}

// SafeJoinPath 安全地连接路径，防止路径遍历
// 类似 filepath.Join，但会验证结果是否在基础目录内
func SafeJoinPath(basePath, relPath string) (string, error) {
	// 清理相对路径
	cleanRelPath := filepath.Clean(relPath)

	// 检查相对路径是否包含 ".."
	if strings.Contains(cleanRelPath, "..") {
		return "", fmt.Errorf("安全错误：拒绝包含父目录引用的路径: %s", relPath)
	}

	// 连接路径
	joinedPath := filepath.Join(basePath, cleanRelPath)

	// 验证最终路径
	return ValidatePath(basePath, joinedPath)
}

// ValidateZipEntry 验证 ZIP 压缩包中的文件路径是否安全
// 返回清理后的安全路径
func ValidateZipEntry(basePath, entryPath string) (string, error) {
	// ZIP 文件中的路径总是使用正斜杠，需要转换
	cleanPath := filepath.FromSlash(entryPath)

	// 检查路径是否包含 ".."（Windows 和 Unix 格式都要检查）
	if strings.Contains(entryPath, "..") || strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("安全错误：ZIP 条目包含路径遍历尝试: %s", entryPath)
	}

	// 检查是否是绝对路径（ZIP 攻击的常见方式）
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("安全错误：ZIP 条目包含绝对路径: %s", entryPath)
	}

	// 构建目标路径
	targetPath := filepath.Join(basePath, cleanPath)

	// 验证最终路径
	return ValidatePath(basePath, targetPath)
}
