package background

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"SCLauncher/backend/config"
)

// Manager 背景图片管理器
type Manager struct {
	config *config.Config
	paths  *config.Paths
}

// NewManager 创建背景图片管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
		paths:  config.NewPaths(cfg),
	}
}

// SetBackground 设置背景图片
// sourcePath: 用户选择的图片文件路径
// 返回: 保存的文件名（不含路径）和错误信息
func (m *Manager) SetBackground(sourcePath string) (string, error) {
	// 检查源文件是否存在
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("source file not found: %w", err)
	}

	if sourceInfo.IsDir() {
		return "", fmt.Errorf("source path is a directory")
	}

	// 验证文件扩展名（支持常见的图片格式）
	fileName := filepath.Base(sourcePath)
	ext := strings.ToLower(filepath.Ext(fileName))
	validExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	if !validExts[ext] {
		return "", fmt.Errorf("invalid file extension, only image files are allowed")
	}

	// 确保背景图片目录存在
	backgroundDir := m.paths.GetBackgroundImageDir()
	if err := os.MkdirAll(backgroundDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create background directory: %w", err)
	}

	// 删除旧的背景图片
	if m.config.BackgroundImage != "" {
		oldBackgroundPath := m.paths.GetBackgroundImagePath(m.config.BackgroundImage)
		if _, err := os.Stat(oldBackgroundPath); err == nil {
			if err := os.Remove(oldBackgroundPath); err != nil {
				// 删除失败不影响设置新背景，只记录日志
				fmt.Printf("Warning: failed to remove old background: %v\n", err)
			}
		}
	}

	// 生成新的文件名（使用时间戳避免重名）
	targetFileName := fmt.Sprintf("background%s", ext)
	targetPath := filepath.Join(backgroundDir, targetFileName)

	// 如果文件已存在，先删除
	if _, err := os.Stat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return "", fmt.Errorf("failed to remove existing background file: %w", err)
		}
	}

	// 复制文件
	if err := copyFile(sourcePath, targetPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// 更新配置
	m.config.BackgroundImage = targetFileName
	if err := m.config.Save(); err != nil {
		// 保存配置失败，删除已复制的文件
		os.Remove(targetPath)
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	return targetFileName, nil
}

// ClearBackground 清除背景图片设置
func (m *Manager) ClearBackground() error {
	// 删除背景图片文件
	if m.config.BackgroundImage != "" {
		backgroundPath := m.paths.GetBackgroundImagePath(m.config.BackgroundImage)
		if _, err := os.Stat(backgroundPath); err == nil {
			if err := os.Remove(backgroundPath); err != nil {
				return fmt.Errorf("failed to remove background file: %w", err)
			}
		}
	}

	// 更新配置
	m.config.BackgroundImage = ""
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetBackgroundImage 获取背景图片路径
// 返回: 背景图片的完整路径，如果未设置则返回空字符串
func (m *Manager) GetBackgroundImage() string {
	if m.config.BackgroundImage == "" {
		return ""
	}
	return m.paths.GetBackgroundImagePath(m.config.BackgroundImage)
}

// HasBackground 检查是否设置了背景图片
func (m *Manager) HasBackground() bool {
	if m.config.BackgroundImage == "" {
		return false
	}

	backgroundPath := m.paths.GetBackgroundImagePath(m.config.BackgroundImage)
	_, err := os.Stat(backgroundPath)
	return err == nil
}

// GetBackgroundImageBase64 获取背景图片的base64编码
func (m *Manager) GetBackgroundImageBase64() (string, error) {
	if m.config.BackgroundImage == "" {
		return "", fmt.Errorf("no background image set")
	}

	backgroundPath := m.paths.GetBackgroundImagePath(m.config.BackgroundImage)

	// 读取文件
	data, err := os.ReadFile(backgroundPath)
	if err != nil {
		return "", fmt.Errorf("failed to read background image: %w", err)
	}

	// 根据文件扩展名确定 MIME 类型
	ext := strings.ToLower(filepath.Ext(m.config.BackgroundImage))
	var mimeType string
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".bmp":
		mimeType = "image/bmp"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "image/jpeg"
	}

	// 转换为base64
	base64Str := base64.StdEncoding.EncodeToString(data)

	// 返回data URI格式
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制内容
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// 复制文件权限
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
