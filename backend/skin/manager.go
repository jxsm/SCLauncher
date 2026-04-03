package skin

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"SCLauncher/backend/config"
)

// getImportedVersionOriginalPath 获取导入版本的原始路径
func getImportedVersionOriginalPath(versionPath string) (string, error) {
	importedMetaFile := filepath.Join(versionPath, ".imported")
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return "", fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				return originalPath, nil
			}
		}

		return "", fmt.Errorf("invalid import metadata file")
	}
	return "", nil // 不是导入版本
}

// Skin 皮肤信息
type Skin struct {
	FileName   string `json:"fileName"`   // 文件名
	Size       int64  `json:"size"`       // 文件大小（字节）
	ImportDate string `json:"importDate"` // 导入日期
}

// Manager 皮肤管理器
type Manager struct {
	config *config.Config
	paths  *config.Paths
}

// NewManager 创建皮肤管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
		paths:  config.NewPaths(cfg),
	}
}

// GetSkins 获取所有皮肤列表
func (m *Manager) GetSkins() ([]Skin, error) {
	skinsDir := m.paths.GetSkinsDir()

	// 检查目录是否存在
	if _, err := os.Stat(skinsDir); os.IsNotExist(err) {
		return []Skin{}, nil
	}

	// 读取目录
	entries, err := os.ReadDir(skinsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skins directory: %w", err)
	}

	var skins []Skin

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()

		// 只处理 .scskin 文件
		if !strings.HasSuffix(strings.ToLower(fileName), ".scskin") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		skins = append(skins, Skin{
			FileName:   fileName,
			Size:       info.Size(),
			ImportDate: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return skins, nil
}

// UploadSkin 上传皮肤文件
func (m *Manager) UploadSkin(sourcePath string) error {
	// 检查源文件是否存在
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source file not found: %w", err)
	}

	if sourceInfo.IsDir() {
		return fmt.Errorf("source path is a directory")
	}

	// 验证文件扩展名
	fileName := filepath.Base(sourcePath)
	if !strings.HasSuffix(strings.ToLower(fileName), ".scskin") {
		return fmt.Errorf("invalid file extension, only .scskin files are allowed")
	}

	// 确保皮肤目录存在
	skinsDir := m.paths.GetSkinsDir()
	if err := os.MkdirAll(skinsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skins directory: %w", err)
	}

	// 目标文件路径
	destPath := filepath.Join(skinsDir, fileName)

	// 检查文件是否已存在
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("skin file already exists: %s", fileName)
	}

	// 复制文件
	if err := copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// DeleteSkin 删除皮肤文件
func (m *Manager) DeleteSkin(fileName string) error {
	// 验证文件名
	if !strings.HasSuffix(strings.ToLower(fileName), ".scskin") {
		return fmt.Errorf("invalid file extension: %s", fileName)
	}

	skinsDir := m.paths.GetSkinsDir()
	filePath := filepath.Join(skinsDir, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("skin file not found: %s", fileName)
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// SyncSkinsToGame 同步皮肤到游戏目录
// 使用硬链接，如果失败则降级为复制文件
func (m *Manager) SyncSkinsToGame(versionID string) error {
	// 获取所有皮肤文件
	skins, err := m.GetSkins()
	if err != nil {
		return fmt.Errorf("failed to get skins list: %w", err)
	}

	// 如果没有皮肤，直接返回
	if len(skins) == 0 {
		return nil
	}

	// 检查是否是导入的版本，获取正确的游戏目录
	versionPath := m.paths.GetVersionPath(versionID)
	charSkinsDir, docCharSkinsDir := m.getGameSkinsDirs(versionPath, versionID)

	// 确保目标目录存在
	if err := os.MkdirAll(charSkinsDir, 0755); err != nil {
		return fmt.Errorf("failed to create CharacterSkins directory: %w", err)
	}
	if err := os.MkdirAll(docCharSkinsDir, 0755); err != nil {
		return fmt.Errorf("failed to create doc/CharacterSkins directory: %w", err)
	}

	skinsDir := m.paths.GetSkinsDir()

	// 同步到两个目录
	for _, skin := range skins {
		sourcePath := filepath.Join(skinsDir, skin.FileName)

		// 同步到 CharacterSkins 目录
		charDestPath := filepath.Join(charSkinsDir, skin.FileName)
		if err := m.linkOrCopyToGame(sourcePath, charDestPath); err != nil {
			return fmt.Errorf("failed to sync skin %s to CharacterSkins: %w", skin.FileName, err)
		}

		// 同步到 doc/CharacterSkins 目录
		docDestPath := filepath.Join(docCharSkinsDir, skin.FileName)
		if err := m.linkOrCopyToGame(sourcePath, docDestPath); err != nil {
			return fmt.Errorf("failed to sync skin %s to doc/CharacterSkins: %w", skin.FileName, err)
		}
	}

	return nil
}

// getGameSkinsDirs 获取游戏皮肤目录（处理导入版本）
func (m *Manager) getGameSkinsDirs(versionPath, versionID string) (charSkinsDir, docCharSkinsDir string) {
	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	if err == nil && originalPath != "" {
		// 是导入版本，使用原始路径
		charSkinsDir = filepath.Join(originalPath, "CharacterSkins")
		docCharSkinsDir = filepath.Join(originalPath, "doc", "CharacterSkins")
		return
	}

	// 正常版本，使用标准路径
	charSkinsDir = m.paths.GetGameCharacterSkinsDir(versionID)
	docCharSkinsDir = m.paths.GetGameDocCharacterSkinsDir(versionID)
	return
}

// linkOrCopyToGame 创建硬链接或复制文件到游戏目录
func (m *Manager) linkOrCopyToGame(sourcePath, destPath string) error {
	// 检查目标文件是否已存在
	if _, err := os.Stat(destPath); err == nil {
		// 文件已存在，检查是否是同一个文件（硬链接）
		sourceInfo, _ := os.Stat(sourcePath)
		destInfo, _ := os.Stat(destPath)

		// 如果inode相同（Unix）或文件信息相同，说明已经是硬链接
		if os.SameFile(sourceInfo, destInfo) {
			return nil // 已经是硬链接，无需处理
		}

		// 文件存在但不是硬链接，删除旧文件
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// 尝试创建硬链接
	err := os.Link(sourcePath, destPath)
	if err == nil {
		// 硬链接成功
		return nil
	}

	// 硬链接失败，降级为复制文件
	// 可能的原因：跨分区、文件系统不支持、权限问题
	return copyFile(sourcePath, destPath)
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

// GetSkinPath 获取皮肤文件的完整路径
func (m *Manager) GetSkinPath(fileName string) string {
	return filepath.Join(m.paths.GetSkinsDir(), fileName)
}

// EnsureSkinsDir 确保皮肤目录存在
func (m *Manager) EnsureSkinsDir() error {
	skinsDir := m.paths.GetSkinsDir()
	return os.MkdirAll(skinsDir, 0755)
}

// GetSkinImage 获取皮肤图片的base64编码
// .scskin文件实际是PNG格式
func (m *Manager) GetSkinImage(fileName string) (string, error) {
	// 验证文件名
	if !strings.HasSuffix(strings.ToLower(fileName), ".scskin") {
		return "", fmt.Errorf("invalid file extension: %s", fileName)
	}

	skinPath := m.GetSkinPath(fileName)

	// 读取文件
	data, err := os.ReadFile(skinPath)
	if err != nil {
		return "", fmt.Errorf("failed to read skin file: %w", err)
	}

	// 转换为base64
	base64 := base64.StdEncoding.EncodeToString(data)

	// 返回data URI格式
	return fmt.Sprintf("data:image/png;base64,%s", base64), nil
}
