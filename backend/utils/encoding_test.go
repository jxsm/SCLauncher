package utils

import (
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeArchiveEntryName(t *testing.T) {
	// ASCII 与合法 UTF-8 名称应原样保留
	for _, name := range []string{
		"Survivalcraft.exe",
		"manifest.json",
		"中文目录/文件.txt",
		"地形-01.png",
	} {
		if got := DecodeArchiveEntryName(name); got != name {
			t.Errorf("DecodeArchiveEntryName(%q) = %q, 期望原样保留", name, got)
		}
	}

	// GBK 编码的名称应解码回中文（模拟中文 Windows 压缩工具生成的压缩包）
	gbkName, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("中文目录/文件.txt"))
	if err != nil {
		t.Fatalf("GBK 编码测试名失败: %v", err)
	}
	if got := DecodeArchiveEntryName(string(gbkName)); got != "中文目录/文件.txt" {
		t.Errorf("GBK 名称解码结果 = %q, 期望 %q", got, "中文目录/文件.txt")
	}

	// 无法按 GBK 识别的字节会被替换为 U+FFFD（宽松解码），结果仍是合法
	// UTF-8 且 ASCII 部分保持不变（不比修复前更差）
	raw := "\xff\xfe\x01garbage"
	got := DecodeArchiveEntryName(raw)
	if !utf8.ValidString(got) {
		t.Errorf("解码结果应是合法 UTF-8: %q", got)
	}
	if !strings.Contains(got, "garbage") {
		t.Errorf("ASCII 部分应保持不变: %q", got)
	}
}
