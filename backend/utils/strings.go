package utils

import "strings"

// ContainsInsensitive 不区分大小写的字符串包含检查
func ContainsInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ContainsAny 检查字符串是否包含任意一个子字符串
func ContainsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
