package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeleteModpackTempFileValidation 验证临时文件删除的安全校验：
// 只允许删除系统临时目录内、scmodpack- 前缀的文件。
func TestDeleteModpackTempFileValidation(t *testing.T) {
	app := &App{}

	// 合法：临时目录内 + 正确前缀 → 删除成功
	valid, err := os.CreateTemp("", "scmodpack-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	validPath := valid.Name()
	valid.Close()
	if err := app.DeleteModpackTempFile(validPath); err != nil {
		t.Errorf("合法临时文件应删除成功: %v", err)
	}
	if _, err := os.Stat(validPath); !os.IsNotExist(err) {
		t.Error("文件应已被删除")
	}

	// 已删除的合法路径再删一次不报错（幂等）
	if err := app.DeleteModpackTempFile(validPath); err != nil {
		t.Errorf("重复删除应幂等: %v", err)
	}

	// 非法：临时目录外（工作目录下的文件）
	outside, err := os.CreateTemp("", "evil-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	outsidePath := outside.Name()
	outside.Close()
	defer os.Remove(outsidePath)
	if err := app.DeleteModpackTempFile(outsidePath); err == nil {
		t.Error("非 scmodpack- 前缀的临时文件应被拒绝")
	}

	// 非法：路径穿越到临时目录外
	tempRoot := os.TempDir()
	if err := app.DeleteModpackTempFile(filepath.Join(tempRoot, "..", "scmodpack-evil.zip")); err == nil {
		t.Error("临时目录外的路径应被拒绝")
	}

	// 非法：纯路径字符串注入（不存在的盘符路径等）
	if err := app.DeleteModpackTempFile(`\\127.0.0.1\share\scmodpack-x.zip`); err == nil {
		t.Error("UNC 路径应被拒绝")
	}
}

// TestDeleteModpackTempFilePrefixBoundary 验证前缀判断基于文件名而非整段路径
func TestDeleteModpackTempFilePrefixBoundary(t *testing.T) {
	app := &App{}
	dir, err := os.MkdirTemp("", "not-scmodpack-dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// 目录本身有 scmodpack- 内容会误导 HasPrefix(abs)，但 base 名不含前缀 → 拒绝
	target := filepath.Join(dir, "innocent.txt")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteModpackTempFile(target); err == nil || !strings.Contains(err.Error(), "拒绝") {
		t.Errorf("临时目录外文件应被拒绝: %v", err)
	}
}
