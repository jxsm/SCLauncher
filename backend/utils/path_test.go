package utils

import (
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		targetPath  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "正常路径",
			basePath:   "/tmp/test",
			targetPath: "/tmp/test/file.txt",
			wantErr:    false,
		},
		{
			name:       "子目录路径",
			basePath:   "/tmp/test",
			targetPath: "/tmp/test/subdir/file.txt",
			wantErr:    false,
		},
		{
			name:        "路径遍历攻击-父目录引用",
			basePath:    "/tmp/test",
			targetPath:  "/tmp/test/../../../etc/passwd",
			wantErr:     true,
			errContains: "路径遍历",
		},
		{
			name:        "路径遍历攻击-隐藏的父目录",
			basePath:    "/tmp/test",
			targetPath:  "/tmp/test/sub/../../etc/passwd",
			wantErr:     true,
			errContains: "父目录引用",
		},
		{
			name:        "绝对路径攻击",
			basePath:    "/tmp/test",
			targetPath:  "/etc/passwd",
			wantErr:     true,
			errContains: "基础目录",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.basePath, tt.targetPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidatePath() error = %v, 期望包含 %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常文件名",
			input:    "test.txt",
			expected: "test.txt",
		},
		{
			name:     "包含路径遍历",
			input:    "../../../etc/passwd",
			expected: "_________etc_passwd",
		},
		{
			name:     "包含危险字符",
			input:    "test<>:\"|?*.txt",
			expected: "test________.txt",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "file",
		},
		{
			name:     "只有点和空格",
			input:    "  .  ",
			expected: "file",
		},
		{
			name:     "中文文件名",
			input:    "测试文件.txt",
			expected: "测试文件.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateZipEntry(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		entryPath   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "正常ZIP条目",
			basePath:  "/tmp/test",
			entryPath: "file.txt",
			wantErr:   false,
		},
		{
			name:      "ZIP条目包含子目录",
			basePath:  "/tmp/test",
			entryPath: "subdir/file.txt",
			wantErr:   false,
		},
		{
			name:        "ZIP条目路径遍历攻击",
			basePath:    "/tmp/test",
			entryPath:   "../../etc/passwd",
			wantErr:     true,
			errContains: "路径遍历",
		},
		{
			name:        "ZIP条目绝对路径攻击",
			basePath:    "/tmp/test",
			entryPath:   "/etc/passwd",
			wantErr:     true,
			errContains: "绝对路径",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateZipEntry(tt.basePath, tt.entryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZipEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateZipEntry() error = %v, 期望包含 %v", err, tt.errContains)
				}
			}
		})
	}
}
