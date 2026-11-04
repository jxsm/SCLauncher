package version

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// buildTestZip 构造测试用 zip。
// 以 GBK 字节作为条目名写入时，archive/zip 的 writer 不会为非法 UTF-8 名称
// 设置 UTF-8 标志位，读取端拿到的就是原始 GBK 字节——与 WinRAR、好压、
// 360 压缩等中文 Windows 工具生成的压缩包行为一致。
func buildTestZip(t *testing.T, zipPath string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("创建 zip 失败: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		fw, err := w.CreateHeader(h)
		if err != nil {
			t.Fatalf("写入条目 %q 失败: %v", name, err)
		}
		if content != "" {
			if _, err := fw.Write([]byte(content)); err != nil {
				t.Fatalf("写入条目内容 %q 失败: %v", name, err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("关闭 zip 失败: %v", err)
	}
}

// gbkName 将 UTF-8 名称编码为 GBK 字节串（用作 zip 条目名）
func gbkName(t *testing.T, name string) string {
	t.Helper()
	b, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(name))
	if err != nil {
		t.Fatalf("GBK 编码 %q 失败: %v", name, err)
	}
	return string(b)
}

func TestInstallDecodesGbkFilenames(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "game.zip")
	destPath := filepath.Join(dir, "extracted")

	buildTestZip(t, zipPath, map[string]string{
		gbkName(t, "中文资源") + "/":            "", // 目录条目
		gbkName(t, "中文资源/说明文件.txt"):       "你好",
		"Survivalcraft/Bin/x86/Survivalcraft.exe": "exe",
	})

	inst := NewInstaller()
	if err := inst.Install(zipPath, destPath, nil); err != nil {
		t.Fatalf("Install 失败: %v", err)
	}

	// 解压出的文件名应是可读中文，而不是 GBK 乱码
	gotPath := filepath.Join(destPath, "中文资源", "说明文件.txt")
	data, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("期望存在解压后的中文路径 %q: %v", gotPath, err)
	}
	if string(data) != "你好" {
		t.Errorf("文件内容不匹配: %q", data)
	}

	// ASCII 条目不受影响
	if _, err := os.Stat(filepath.Join(destPath, "Survivalcraft", "Bin", "x86", "Survivalcraft.exe")); err != nil {
		t.Errorf("ASCII 条目解压失败: %v", err)
	}
}

func TestInstallRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	destPath := filepath.Join(dir, "extracted")

	buildTestZip(t, zipPath, map[string]string{
		"../evil.txt": "malicious",
	})

	inst := NewInstaller()
	err := inst.Install(zipPath, destPath, nil)
	if err == nil {
		t.Fatal("期望包含路径穿越条目（../evil.txt）的压缩包被拒绝")
	}
	if !strings.Contains(err.Error(), "evil.txt") {
		t.Errorf("错误信息应包含问题条目名: %v", err)
	}
	// 目标目录之外不应出现文件
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Error("路径穿越文件不应被写出")
	}
}
