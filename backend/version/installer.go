package version

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v4"
)

// Installer 版本安装器
type Installer struct{}

// NewInstaller 创建安装器
func NewInstaller() *Installer {
	return &Installer{}
}

// Install 从压缩文件安装版本（支持 zip、7z、rar 等格式）
func (i *Installer) Install(archivePath, destPath string, progress func(current, total int64)) error {
	// 创建目标目录
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 打开压缩文件
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer file.Close()

	// 使用 archiver 库自动检测格式并解压
	ctx := context.Background()
	format, input, err := archiver.Identify(ctx, archivePath, file)
	if err != nil {
		return fmt.Errorf("failed to identify archive format: %w", err)
	}

	// 如果输入实现了 Close 方法，关闭它
	if closer, ok := input.(io.Closer); ok {
		defer closer.Close()
	}

	// 检查是否是解压器
	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return fmt.Errorf("archive format does not support extraction")
	}

	// 如果需要进度回调，先计算总大小
	var totalSize int64
	if progress != nil {
		// 使用 FileHandler 来计算总大小
		handler := func(ctx context.Context, f archiver.FileInfo) error {
			info, err := f.Stat()
			if err != nil {
				return err
			}
			totalSize += info.Size()
			return nil
		}

		// 先遍历一次获取总大小
		if err := extractor.Extract(ctx, input, handler); err != nil {
			return fmt.Errorf("failed to calculate archive size: %w", err)
		}

		// 重新打开文件（因为第一次读取已经消耗了流）
		file.Close()
		file, err = os.Open(archivePath)
		if err != nil {
			return fmt.Errorf("failed to reopen archive file: %w", err)
		}
		defer file.Close()

		_, input, err = archiver.Identify(ctx, archivePath, file)
		if err != nil {
			return fmt.Errorf("failed to re-identify archive format: %w", err)
		}
		if closer, ok := input.(io.Closer); ok {
			defer closer.Close()
		}
	}

	var currentSize int64
	// 解压文件
	handler := func(ctx context.Context, f archiver.FileInfo) error {
		info, err := f.Stat()
		if err != nil {
			return err
		}

		// 构建目标路径
		destFilePath := filepath.Join(destPath, f.NameInArchive)

		// 创建目录
		if info.IsDir() {
			if err := os.MkdirAll(destFilePath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			return nil
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(destFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// 打开源文件
		srcFile, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in archive: %w", err)
		}
		defer srcFile.Close()

		// 创建目标文件
		destFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer destFile.Close()

		// 复制文件内容
		written, err := io.Copy(destFile, srcFile)
		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		// 更新进度
		if progress != nil {
			currentSize += written
			progress(currentSize, totalSize)
		}

		return nil
	}

	// 执行解压
	if err := extractor.Extract(ctx, input, handler); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
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
