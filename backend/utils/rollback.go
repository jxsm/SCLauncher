package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// RollbackManager 回滚管理器，用于在操作失败时清理已完成的更改
type RollbackManager struct {
	operations []RollbackOperation
}

// RollbackOperation 回滚操作
type RollbackOperation struct {
	Name     string
	Rollback func() error
}

// NewRollbackManager 创建回滚管理器
func NewRollbackManager() *RollbackManager {
	return &RollbackManager{
		operations: make([]RollbackOperation, 0),
	}
}

// Register 注册一个回滚操作
func (rm *RollbackManager) Register(name string, rollback func() error) {
	rm.operations = append(rm.operations, RollbackOperation{
		Name:     name,
		Rollback: rollback,
	})
}

// RegisterFileCreate 注册文件创建操作的回滚
func (rm *RollbackManager) RegisterFileCreate(filePath string) {
	rm.Register(fmt.Sprintf("删除文件: %s", filePath), func() error {
		return os.Remove(filePath)
	})
}

// RegisterDirCreate 注册目录创建操作的回滚
func (rm *RollbackManager) RegisterDirCreate(dirPath string) {
	rm.Register(fmt.Sprintf("删除目录: %s", dirPath), func() error {
		return os.RemoveAll(dirPath)
	})
}

// RegisterFileCopy 注册文件复制操作的回滚
func (rm *RollbackManager) RegisterFileCopy(destPath string) {
	rm.Register(fmt.Sprintf("删除复制的文件: %s", destPath), func() error {
		return os.Remove(destPath)
	})
}

// Execute 执行回滚（按相反顺序）
func (rm *RollbackManager) Execute() error {
	// 按相反顺序执行回滚
	for i := len(rm.operations) - 1; i >= 0; i-- {
		op := rm.operations[i]
		if err := op.Rollback(); err != nil {
			// 记录错误但继续执行其他回滚
			fmt.Printf("回滚操作失败 [%s]: %v\n", op.Name, err)
		}
	}
	rm.operations = make([]RollbackOperation, 0)
	return nil
}

// Clear 清除所有回滚操作（操作成功时调用）
func (rm *RollbackManager) Clear() {
	rm.operations = make([]RollbackOperation, 0)
}

// HasOperations 检查是否有注册的操作
func (rm *RollbackManager) HasOperations() bool {
	return len(rm.operations) > 0
}

// CleanupFailedInstall 清理失败的安装
func CleanupFailedInstall(versionPath string) error {
	// 检查版本路径是否存在
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return nil // 不存在，无需清理
	}

	// 删除整个版本目录
	if err := os.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("清理失败安装目录失败: %w", err)
	}

	return nil
}

// CleanupTempFiles 清理临时文件
func CleanupTempFiles(tempDir string) error {
	// 检查临时目录是否存在
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return nil // 不存在，无需清理
	}

	// 删除临时目录
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("清理临时文件失败: %w", err)
	}

	return nil
}

// EnsureTempDirectory 确保临时目录存在并可访问
func EnsureTempDirectory(tempDir string) error {
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 测试是否可以写入
	testFile := filepath.Join(tempDir, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("临时目录不可写: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}
