package mod

import (
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

// Manager 模组管理器
type Manager struct {
	config *config.Config
	paths  *config.Paths
}

// Mod 模组信息
type Mod struct {
	ID          string `json:"id"`          // 模组 ID（文件名不含扩展名）
	Name        string `json:"name"`        // 模组名称（文件名）
	FileName     string `json:"fileName"`    // 文件名
	VersionID   string `json:"versionId"`   // 所属版本 ID
	Enabled     bool   `json:"enabled"`     // 是否启用
	Size        int64  `json:"size"`        // 文件大小
	InstallDate string `json:"installDate"` // 安装日期
}

// NewManager 创建模组管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
		paths:  config.NewPaths(cfg),
	}
}

// getModPath 获取模组目录路径（处理导入版本）
func (m *Manager) getModPath(versionID string) string {
	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	if err == nil && originalPath != "" {
		// 是导入版本，使用原始路径的mods目录
		return filepath.Join(originalPath, "mods")
	}

	// 正常版本，使用标准路径
	return m.paths.GetModPath(versionID)
}

// GetMods 获取指定版本的模组列表
func (m *Manager) GetMods(versionID string) ([]Mod, error) {
	modsDir := m.getModPath(versionID)

	// 检查目录是否存在
	if _, err := os.Stat(modsDir); os.IsNotExist(err) {
		return []Mod{}, nil
	}

	// 读取目录
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read mods directory: %w", err)
	}

	var mods []Mod

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 判断模组是否启用（检查是否以 .disable 结尾）
		enabled := true
		modName := fileName

		if strings.HasSuffix(strings.ToLower(fileName), ".disable") {
			enabled = false
			// 移除 .disable 后缀得到真实文件名
			modName = fileName[:len(fileName)-len(".disable")]
		}

		// 生成模组 ID（使用不含扩展名的文件名）
		id := strings.TrimSuffix(modName, filepath.Ext(modName))

		mods = append(mods, Mod{
			ID:          id,
			Name:        modName,
			FileName:    fileName,
			VersionID:   versionID,
			Enabled:     enabled,
			Size:        info.Size(),
			InstallDate: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return mods, nil
}

// ImportMod 导入模组（将文件复制到模组目录）
func (m *Manager) ImportMod(versionID, sourcePath string) error {
	// 检查源文件是否存在
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source file not found: %w", err)
	}

	if sourceInfo.IsDir() {
		return fmt.Errorf("source path is a directory")
	}

	// 确保模组目录存在
	modsDir := m.getModPath(versionID)
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return fmt.Errorf("failed to create mods directory: %w", err)
	}

	// 目标文件路径
	destPath := filepath.Join(modsDir, filepath.Base(sourcePath))

	// 复制文件
	if err := copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// ToggleMod 切换模组启用/禁用状态
func (m *Manager) ToggleMod(versionID, modID string, enabled bool) error {
	modsDir := m.getModPath(versionID)

	// 查找模组文件
	mods, err := m.GetMods(versionID)
	if err != nil {
		return err
	}

	var targetMod *Mod
	for _, mod := range mods {
		if mod.ID == modID {
			targetMod = &mod
			break
		}
	}

	if targetMod == nil {
		return fmt.Errorf("mod not found: %s", modID)
	}

	// 当前文件路径
	oldPath := filepath.Join(modsDir, targetMod.FileName)

	// 新文件路径
	var newPath string
	if enabled {
		// 启用：移除 .disable 后缀
		if strings.HasSuffix(strings.ToLower(oldPath), ".disable") {
			newPath = oldPath[:len(oldPath)-len(".disable")]
		} else {
			newPath = oldPath
		}
	} else {
		// 禁用：添加 .disable 后缀
		newPath = oldPath + ".disable"
	}

	// 重命名文件
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// DeleteMod 删除模组
func (m *Manager) DeleteMod(versionID, modID string) error {
	modsDir := m.getModPath(versionID)

	// 查找模组文件
	mods, err := m.GetMods(versionID)
	if err != nil {
		return err
	}

	var targetMod *Mod
	for _, mod := range mods {
		if mod.ID == modID {
			targetMod = &mod
			break
		}
	}

	if targetMod == nil {
		return fmt.Errorf("mod not found: %s", modID)
	}

	// 文件路径
	filePath := filepath.Join(modsDir, targetMod.FileName)

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
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
