package texture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Texture 材质信息
type Texture struct {
	ID       string `json:"id"`       // 材质 ID（文件名，不含扩展名）
	Name     string `json:"name"`     // 材质名称（显示名称）
	FileName string `json:"fileName"` // 完整文件名（含扩展名）
}

// Manager 材质管理器
type Manager struct {
 getVersionPath func(versionID string) string
}

// NewManager 创建材质管理器
func NewManager(getVersionPath func(versionID string) string) *Manager {
	return &Manager{
		getVersionPath: getVersionPath,
	}
}

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

// GetTextures 获取指定版本的材质列表
// 如果材质文件夹不存在，返回 nil, nil
func (m *Manager) GetTextures(versionID string) ([]Texture, error) {
	// 获取版本路径
	versionPath := m.getVersionPath(versionID)

	// 检查版本路径是否存在
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("version path not found: %s", versionPath)
	}

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		originalPath, err := getImportedVersionOriginalPath(versionPath)
		if err != nil {
			return nil, err
		}
		if originalPath == "" {
			return nil, fmt.Errorf("invalid import metadata file")
		}
		basePath = originalPath
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "TexturePacks"),
		filepath.Join(basePath, "TexturePacks"),
	}

	var texturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			texturePath = path
			break
		}
	}

	// 如果都没有找到，返回 nil（表示文件夹不存在）
	if texturePath == "" {
		return nil, nil
	}

	// 读取目录中的所有文件
	entries, err := os.ReadDir(texturePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read texture directory: %w", err)
	}

	// 初始化为空切片，确保JSON序列化时返回[]而不是null
	textures := make([]Texture, 0)

	for _, entry := range entries {
		// 跳过目录和隐藏文件
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fileName := entry.Name()

		// 只处理 .scbtex 文件
		if !strings.HasSuffix(strings.ToLower(fileName), ".scbtex") {
			continue
		}

		// 去掉扩展名作为 ID 和 Name
		id := strings.TrimSuffix(fileName, ".scbtex")
		name := id

		textures = append(textures, Texture{
			ID:       id,
			Name:     name,
			FileName: fileName,
		})
	}

	return textures, nil
}

// GetTexturePath 获取材质文件夹路径
// 如果材质文件夹不存在，返回空字符串
func (m *Manager) GetTexturePath(versionID string) (string, error) {
	// 获取版本路径
	versionPath := m.getVersionPath(versionID)

	// 检查版本路径是否存在
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return "", fmt.Errorf("version path not found: %s", versionPath)
	}

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		originalPath, err := getImportedVersionOriginalPath(versionPath)
		if err != nil {
			return "", err
		}
		if originalPath == "" {
			return "", fmt.Errorf("invalid import metadata file")
		}
		basePath = originalPath
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "TexturePacks"),
		filepath.Join(basePath, "TexturePacks"),
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}
	}

	// 都没有找到，返回空字符串
	return "", nil
}

// DeleteTexture 删除材质
func (m *Manager) DeleteTexture(versionID, textureID string) error {
	// 获取材质文件夹路径
	texturePath, err := m.GetTexturePath(versionID)
	if err != nil {
		return err
	}

	if texturePath == "" {
		return fmt.Errorf("texture directory not found")
	}

	// 构建文件路径（添加 .scbtex 扩展名）
	fileName := textureID + ".scbtex"
	filePath := filepath.Join(texturePath, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("texture file not found: %s", fileName)
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete texture: %w", err)
	}

	return nil
}

// RenameTexture 重命名材质
func (m *Manager) RenameTexture(versionID, textureID, newName string) error {
	// 验证新名称
	if newName == "" {
		return fmt.Errorf("new name cannot be empty")
	}

	// 检查新名称是否已包含扩展名
	if !strings.HasSuffix(strings.ToLower(newName), ".scbtex") {
		newName = newName + ".scbtex"
	}

	// 获取材质文件夹路径
	texturePath, err := m.GetTexturePath(versionID)
	if err != nil {
		return err
	}

	if texturePath == "" {
		return fmt.Errorf("texture directory not found")
	}

	// 原文件路径
	oldFileName := textureID + ".scbtex"
	oldFilePath := filepath.Join(texturePath, oldFileName)

	// 新文件路径
	newFilePath := filepath.Join(texturePath, newName)

	// 检查原文件是否存在
	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		return fmt.Errorf("texture file not found: %s", oldFileName)
	}

	// 检查新文件名是否已存在
	if _, err := os.Stat(newFilePath); err == nil {
		return fmt.Errorf("texture with name '%s' already exists", newName)
	}

	// 重命名文件
	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to rename texture: %w", err)
	}

	return nil
}
