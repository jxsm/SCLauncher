package version

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Installer 版本安装器
type Installer struct{}

// NewInstaller 创建安装器
func NewInstaller() *Installer {
	return &Installer{}
}

// Install 从 zip 文件安装版本
func (i *Installer) Install(zipPath, destPath string, progress func(current, total int64)) error {
	// 打开 zip 文件
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// 创建目标目录
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 计算总大小
	var totalSize int64
	for _, file := range reader.File {
		totalSize += int64(file.UncompressedSize64)
	}

	var currentSize int64

	// 解压文件
	for _, file := range reader.File {
		// 构建目标路径
		destFilePath := filepath.Join(destPath, file.Name)

		// 创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destFilePath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(destFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// 创建文件
		destFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		// 打开 zip 中的文件
		srcFile, err := file.Open()
		if err != nil {
			destFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		// 复制文件内容
		written, err := io.Copy(destFile, srcFile)
		if err != nil {
			srcFile.Close()
			destFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		srcFile.Close()
		destFile.Close()

		// 更新进度
		currentSize += written
		if progress != nil {
			progress(currentSize, totalSize)
		}
	}

	return nil
}

// Uninstall 卸载版本
func (i *Installer) Uninstall(versionPath string) error {
	// 检查路径是否存在
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("version path does not exist: %s", versionPath)
	}

	// 删除整个目录
	if err := os.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	return nil
}

// IsInstalled 检查版本是否已安装
func (i *Installer) IsInstalled(versionPath string) bool {
	info, err := os.Stat(versionPath)
	return err == nil && info.IsDir()
}

// GetInstallSize 获取安装大小
func (i *Installer) GetInstallSize(versionPath string) (int64, error) {
	var size int64

	err := filepath.Walk(versionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// ContainsGameExecutable 检查目录中是否包含游戏可执行文件
func (i *Installer) ContainsGameExecutable(versionPath string) bool {
	var found bool

	err := filepath.Walk(versionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 检查是否是 .exe 文件且文件名包含 "Survivalcraft"
		if strings.HasSuffix(strings.ToLower(info.Name()), ".exe") &&
		   strings.Contains(strings.ToLower(info.Name()), "survivalcraft") {
			found = true
			return filepath.SkipDir
		}

		return nil
	})

	return err == nil && found
}
