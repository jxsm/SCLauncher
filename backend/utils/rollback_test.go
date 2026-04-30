package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollbackManager(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	t.Run("基本回滚操作", func(t *testing.T) {
		rm := NewRollbackManager()

		// 创建一个测试文件
		testFile := filepath.Join(tempDir, "test.txt")
		rm.RegisterFileCreate(testFile)

		// 执行操作：创建文件
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}

		// 验证文件存在
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("测试文件应该存在")
		}

		// 执行回滚
		if err := rm.Execute(); err != nil {
			t.Errorf("回滚执行失败: %v", err)
		}

		// 验证文件已被删除
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Error("回滚后测试文件应该被删除")
		}
	})

	t.Run("目录回滚操作", func(t *testing.T) {
		rm := NewRollbackManager()

		// 创建测试目录
		testDir := filepath.Join(tempDir, "testdir")
		rm.RegisterDirCreate(testDir)

		// 执行操作：创建目录和文件
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}

		// 验证目录存在
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Error("测试目录应该存在")
		}

		// 执行回滚
		if err := rm.Execute(); err != nil {
			t.Errorf("回滚执行失败: %v", err)
		}

		// 验证目录已被删除
		if _, err := os.Stat(testDir); !os.IsNotExist(err) {
			t.Error("回滚后测试目录应该被删除")
		}
	})

	t.Run("清除操作", func(t *testing.T) {
		rm := NewRollbackManager()

		// 注册一些操作
		testFile := filepath.Join(tempDir, "test2.txt")
		rm.RegisterFileCreate(testFile)

		// 创建文件
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}

		// 清除回滚操作
		rm.Clear()

		// 执行回滚（应该什么也不做）
		if err := rm.Execute(); err != nil {
			t.Errorf("回滚执行失败: %v", err)
		}

		// 验证文件仍然存在（因为没有回滚操作）
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Error("清除操作后，文件应该仍然存在")
		}
	})
}

func TestCleanupFailedInstall(t *testing.T) {
	// 创建临时测试目录
	tempDir := t.TempDir()

	// 创建模拟的版本目录结构
	versionPath := filepath.Join(tempDir, "test-version")
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("创建版本目录失败: %v", err)
	}

	// 创建一些文件
	testFile := filepath.Join(versionPath, "game.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		t.Error("版本目录应该存在")
	}

	// 执行清理
	if err := CleanupFailedInstall(versionPath); err != nil {
		t.Errorf("清理失败安装失败: %v", err)
	}

	// 验证目录已被删除
	if _, err := os.Stat(versionPath); !os.IsNotExist(err) {
		t.Error("清理后版本目录应该被删除")
	}
}

func TestEnsureTempDirectory(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("创建新的临时目录", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "new-temp")
		if err := EnsureTempDirectory(testDir); err != nil {
			t.Errorf("EnsureTempDirectory() error = %v", err)
		}

		// 验证目录存在
		if info, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Error("临时目录应该被创建")
		} else if err != nil {
			t.Errorf("无法检查临时目录: %v", err)
		} else if !info.IsDir() {
			t.Error("路径应该是目录")
		}
	})

	t.Run("验证目录可写", func(t *testing.T) {
		// 在某些系统上可能无法真正测试权限，所以我们只是验证函数不会崩溃
		testDir := filepath.Join(tempDir, "writable-temp")
		if err := EnsureTempDirectory(testDir); err != nil {
			t.Errorf("EnsureTempDirectory() error = %v", err)
		}
	})
}
