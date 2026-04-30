package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DetectEncoding 检测字符串编码
// 返回 "utf-8", "gbk", "unknown" 或其他编码
func DetectEncoding(data []byte) string {
	// 检查是否是有效的 UTF-8
	if utf8.Valid(data) {
		return "utf-8"
	}

	// 对于中文环境，尝试检测 GBK
	// 简单启发式：如果包含高字节字符，可能是 GBK
	for _, b := range data {
		if b > 127 {
			// 包含非ASCII字符，可能是GBK
			// 注意：这不是100%准确的检测，但对我们处理ZIP文件名通常足够
			return "gbk"
		}
	}

	return "unknown"
}

// SanitizeZipFilename 清理ZIP文件中的文件名，处理编码问题
// 尝试将文件名转换为有效的UTF-8字符串
func SanitizeZipFilename(filename string) string {
	// 检查是否已经是有效的UTF-8
	if utf8.ValidString(filename) {
		// 即使是有效的UTF-8，也要清理无效字符
		return sanitizeValidUTF8(filename)
	}

	// 尝试修复可能的编码问题
	return fixInvalidEncoding(filename)
}

// sanitizeValidUTF8 清理有效的UTF-8字符串，移除控制字符等
func sanitizeValidUTF8(filename string) string {
	// 移除控制字符（保留换行符和制表符在特定情况下，但文件名中不应该有）
	var cleaned strings.Builder
	for _, r := range filename {
		// 跳过控制字符（0x00-0x1F, 0x7F）
		if r < 32 || r == 127 {
			// 替换为下划线
			cleaned.WriteRune('_')
		} else {
			cleaned.WriteRune(r)
		}
	}

	result := cleaned.String()

	// 移除路径中的危险字符
	result = filepath.Base(result) // 只取文件名部分
	result = strings.Trim(result, " .") // 移除首尾空格和点

	// 如果结果为空，使用默认文件名
	if result == "" || result == "." || result == ".." {
		result = "file"
	}

	return result
}

// fixInvalidEncoding 尝试修复无效的编码字符串
func fixInvalidEncoding(filename string) string {
	// 尝试将每个字节解释为Latin-1（这是ZIP文件的默认编码）
	var cleaned strings.Builder

	for _, b := range []byte(filename) {
		if b < 32 || b == 127 {
			// 控制字符，替换为下划线
			cleaned.WriteRune('_')
		} else if b > 127 {
			// 高字节字符，尝试作为Latin-1解释
			cleaned.WriteRune(rune(b))
		} else {
			// ASCII字符
			cleaned.WriteRune(rune(b))
		}
	}

	result := cleaned.String()

	// 如果仍然不是有效的UTF-8，使用替代字符
	if !utf8.ValidString(result) {
		// 最后的补救：将每个无效字节替换为替换字符
		result = strings.ToValidUTF8(result, "�")
	}

	// 应用基本的清理
	return sanitizeValidUTF8(result)
}

// ValidateAndSanitizePath 验证并清理完整路径
func ValidateAndSanitizePath(basePath, relPath string) (string, error) {
	// 清理文件名
	cleanedPath := filepath.Clean(relPath)

	// 分割路径并清理每个组件
	components := strings.Split(cleanedPath, string(filepath.Separator))
	var cleanedComponents []string

	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("路径包含非法组件: %s", component)
		}

		// 清理每个文件名组件
		cleaned := SanitizeZipFilename(component)
		cleanedComponents = append(cleanedComponents, cleaned)
	}

	// 重新组合路径
	cleanedPath = filepath.Join(cleanedComponents...)

	// 验证最终路径
	if strings.ContainsAny(cleanedPath, "\x00\\:*?\"<>|") {
		return "", fmt.Errorf("路径包含非法字符")
	}

	return cleanedPath, nil
}

// ConvertEncoding 尝试将字符串从一种编码转换为UTF-8
// 这是一个简化的版本，实际应用中可能需要使用专门的编码转换库
func ConvertEncoding(input string, fromEncoding string) (string, error) {
	if fromEncoding == "utf-8" {
		if utf8.ValidString(input) {
			return input, nil
		}
		return "", fmt.Errorf("输入不是有效的UTF-8")
	}

	// 对于其他编码，这里只是占位实现
	// 实际应用中应该使用 golang.org/x/text/encoding 包
	// 这里我们返回清理后的结果
	result := SanitizeZipFilename(input)
	return result, nil
}
