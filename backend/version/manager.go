package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"SCLauncher/backend/config"
	"SCLauncher/backend/storage"
)

// Manager 版本管理器
type Manager struct {
	config      *config.Config
	paths       *config.Paths
	repository  *storage.Repository
	parser      *ManifestParser
	downloader  *Downloader
	installer   *Installer
}

// NewManager 创建版本管理器
func NewManager(cfg *config.Config, repo *storage.Repository) *Manager {
	paths := config.NewPaths(cfg)

	return &Manager{
		config:     cfg,
		paths:      paths,
		repository: repo,
		parser:     NewManifestParser(),
		downloader: NewDownloader(cfg.MaxConcurrent),
		installer:  NewInstaller(),
	}
}

// FetchVersions 从清单文件获取版本列表
func (m *Manager) FetchVersions() ([]Version, error) {
	manifest, err := m.parser.ParseFromURL(m.config.ManifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	versions := manifest.ToVersions()

	// 更新数据库中的版本信息
	for _, version := range versions {
		// 检查版本是否已存在
		existing, err := m.repository.GetVersion(version.ID)
		if err != nil {
			// 版本不存在，创建新记录
			model := m.versionToModel(&version)
			if err := m.repository.CreateVersion(model); err != nil {
				return nil, fmt.Errorf("failed to save version: %w", err)
			}
		} else {
			// 版本已存在，更新信息
			model := m.versionToModel(&version)
			model.Installed = existing.Installed
			model.LocalPath = existing.LocalPath
			if err := m.repository.UpdateVersion(model); err != nil {
				return nil, fmt.Errorf("failed to update version: %w", err)
			}
		}
	}

	return versions, nil
}

// GetVersions 获取所有版本
func (m *Manager) GetVersions() ([]Version, error) {
	models, err := m.repository.ListVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	versions := make([]Version, len(models))
	for i, model := range models {
		versions[i] = m.modelToVersion(&model)
	}

	return versions, nil
}

// GetVersionsByType 按类型获取版本
func (m *Manager) GetVersionsByType(vtype VersionType) ([]Version, error) {
	models, err := m.repository.ListVersionsByType(string(vtype))
	if err != nil {
		return nil, fmt.Errorf("failed to get versions by type: %w", err)
	}

	versions := make([]Version, len(models))
	for i, model := range models {
		versions[i] = m.modelToVersion(&model)
	}

	return versions, nil
}

// GetInstalledVersions 获取已安装的版本
func (m *Manager) GetInstalledVersions() ([]Version, error) {
	models, err := m.repository.ListInstalledVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed versions: %w", err)
	}

	versions := make([]Version, len(models))
	for i, model := range models {
		versions[i] = m.modelToVersion(&model)
	}

	return versions, nil
}

// DownloadVersion 下载版本
func (m *Manager) DownloadVersion(versionID string, progress func(downloaded, total, speed int64)) error {
	// 获取版本信息
	model, err := m.repository.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 构建保存路径
	tempPath := m.paths.GetDownloadTempPath(fmt.Sprintf("%s.zip", versionID))

	// 开始下载
	if err := m.downloader.Download(versionID, model.DownloadURL, tempPath, progress); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// 验证校验和
	if model.Checksum != "" {
		valid, err := VerifyChecksum(tempPath, model.Checksum)
		if err != nil {
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		if !valid {
			return fmt.Errorf("checksum verification failed")
		}
	}

	return nil
}

// DownloadVersionWithCustomName 下载版本（使用自定义名称）
func (m *Manager) DownloadVersionWithCustomName(originalVersionID, uniqueVersionID, customName string, progress func(downloaded, total, speed int64)) error {
	// 获取原始版本信息
	model, err := m.repository.GetVersion(originalVersionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 创建新的版本记录（使用自定义名称）
	customVersion := &Version{
		ID:          uniqueVersionID,
		VersionType: VersionType(model.VersionType),
		GameVersion: model.GameVersion,
		SubVersion:  model.SubVersion,
		Name:        customName,
		Size:        model.Size,
		DownloadURL: model.DownloadURL,
		Checksum:    model.Checksum,
		FileFormat:  model.FileFormat,
		Illustrate:  model.Illustrate,
		ReleaseDate: model.CreatedAt,
	}

	// 保存到数据库
	customModel := m.versionToModel(customVersion)
	if err := m.repository.CreateVersion(customModel); err != nil {
		return fmt.Errorf("failed to save version: %w", err)
	}

	// 构建保存路径
	tempPath := m.paths.GetDownloadTempPath(fmt.Sprintf("%s.zip", uniqueVersionID))

	// 开始下载
	if err := m.downloader.Download(uniqueVersionID, model.DownloadURL, tempPath, progress); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// 验证校验和
	if model.Checksum != "" {
		valid, err := VerifyChecksum(tempPath, model.Checksum)
		if err != nil {
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		if !valid {
			return fmt.Errorf("checksum verification failed")
		}
	}

	return nil
}

// InstallVersion 安装版本
func (m *Manager) InstallVersion(versionID string, progress func(current, total int64)) error {
	// 获取版本信息
	model, err := m.repository.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 检查是否已安装
	if model.Installed {
		return fmt.Errorf("version already installed")
	}

	// 获取下载的 zip 文件路径
	tempPath := m.paths.GetDownloadTempPath(fmt.Sprintf("%s.zip", versionID))

	// 检查文件是否存在
	if err := CheckDownloadFile(tempPath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if err := m.config.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// 创建版本目录
	versionPath := m.paths.GetVersionPath(versionID)

	// 解压安装
	if err := m.installer.Install(tempPath, versionPath, progress); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	// 更新数据库
	if err := m.repository.UpdateVersionInstalledStatus(versionID, true, versionPath); err != nil {
		return fmt.Errorf("failed to update version status: %w", err)
	}

	// 创建模组目录
	if err := m.paths.EnsureModDir(versionID); err != nil {
		return fmt.Errorf("failed to create mods directory: %w", err)
	}

	// 删除临时 zip 文件
	os.Remove(tempPath)

	// 安装成功后，自动设置主要版本（如果没有的话）
	_ = m.repository.AutoSetPrimaryVersion()

	return nil
}

// DeleteVersion 删除版本
func (m *Manager) DeleteVersion(versionID string) error {
	// 获取版本信息
	model, err := m.repository.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 检查是否已安装
	if !model.Installed {
		return fmt.Errorf("version is not installed")
	}

	// 如果是主要版本，先取消主要标记
	if model.IsPrimary {
		// 清除所有主要标记（传入空字符串）
		if err := m.repository.SetPrimaryVersion(""); err != nil {
			return fmt.Errorf("failed to clear primary version: %w", err)
		}
	}

	// 删除版本目录
	if err := m.paths.DeleteVersion(versionID); err != nil {
		return fmt.Errorf("failed to delete version directory: %w", err)
	}

	// 删除模组记录
	if err := m.repository.DeleteModsByVersion(versionID); err != nil {
		return fmt.Errorf("failed to delete mods: %w", err)
	}

	// 更新数据库
	if err := m.repository.UpdateVersionInstalledStatus(versionID, false, ""); err != nil {
		return fmt.Errorf("failed to update version status: %w", err)
	}

	// 删除成功后，自动重新设置主要版本（如果需要）
	_ = m.repository.AutoSetPrimaryVersion()

	return nil
}

// RenameVersion 重命名版本
func (m *Manager) RenameVersion(versionID, newName string) error {
	// 检查版本是否存在
	model, err := m.repository.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	// 检查新名称是否与其他版本重复
	exists, err := m.repository.CheckVersionNameExists(newName, versionID)
	if err != nil {
		return fmt.Errorf("failed to check name: %w", err)
	}
	if exists {
		return fmt.Errorf("version name '%s' already exists", newName)
	}

	// 更新数据库名称
	if err := m.repository.RenameVersion(versionID, newName); err != nil {
		return fmt.Errorf("failed to rename version: %w", err)
	}

	// 记录日志
	_ = fmt.Sprintf("Version renamed: %s -> %s (ID: %s)", model.Name, newName, versionID)

	return nil
}

// AutoSetPrimaryVersion 自动设置主要版本
func (m *Manager) AutoSetPrimaryVersion() error {
	return m.repository.AutoSetPrimaryVersion()
}

// CancelDownload 取消下载
func (m *Manager) CancelDownload(versionID string) error {
	return m.downloader.Cancel(versionID)
}

// versionToModel 转换 Version 到 VersionModel
func (m *Manager) versionToModel(v *Version) *storage.VersionModel {
	return &storage.VersionModel{
		ID:          v.ID,
		VersionType: string(v.VersionType),
		GameVersion: v.GameVersion,
		SubVersion:  v.SubVersion,
		Name:        v.Name,
		Size:        v.Size,
		DownloadURL: v.DownloadURL,
		Checksum:    v.Checksum,
		FileFormat:  v.FileFormat,
		Illustrate:  v.Illustrate,
	}
}

// modelToVersion 转换 VersionModel 到 Version
func (m *Manager) modelToVersion(model *storage.VersionModel) Version {
	// 检查版本路径是否存在
	pathExists := m.checkVersionPathExists(model.ID, model.LocalPath)

	return Version{
		ID:          model.ID,
		VersionType: VersionType(model.VersionType),
		GameVersion: model.GameVersion,
		SubVersion:  model.SubVersion,
		Name:        model.Name,
		Size:        model.Size,
		DownloadURL: model.DownloadURL,
		Checksum:    model.Checksum,
		FileFormat:  model.FileFormat,
		Illustrate:  model.Illustrate,
		ReleaseDate: model.CreatedAt,
		Installed:   model.Installed,
		LocalPath:   model.LocalPath,
		PathExists:  pathExists,
	}
}

// checkVersionPathExists 检查版本路径是否存在
func (m *Manager) checkVersionPathExists(versionID string, localPath string) bool {
	fmt.Printf("[PathCheck] Checking version: %s, localPath: %s\n", versionID, localPath)

	// 对于导入的版本，检查导入的元数据文件
	if strings.HasPrefix(versionID, "imported-") {
		fmt.Println("[PathCheck] This is an imported version")
		// 检查元数据文件是否存在
		versionPath := m.paths.GetVersionPath(versionID)
		metaFile := filepath.Join(versionPath, ".imported")
		fmt.Printf("[PathCheck] Meta file: %s\n", metaFile)

		// 先检查元数据文件是否存在
		_, statErr := os.Stat(metaFile)
		if os.IsNotExist(statErr) {
			// 元数据文件不存在，说明导入的版本有问题
			fmt.Println("[PathCheck] Meta file does not exist")
			return false
		}

		// 读取元数据文件中的原始路径
		content, readErr := os.ReadFile(metaFile)
		if readErr != nil {
			fmt.Printf("[PathCheck] Failed to read meta file: %v\n", readErr)
			return false
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				fmt.Printf("[PathCheck] Original path from meta: %s\n", originalPath)
				// 检查原始路径是否存在
				_, pathErr := os.Stat(originalPath)
				if pathErr == nil {
					fmt.Println("[PathCheck] Original path exists")
					return true
				}
				// 原始路径不存在
				fmt.Printf("[PathCheck] Original path does not exist: %v\n", pathErr)
				return false
			}
		}

		// 没有找到原始路径信息
		fmt.Println("[PathCheck] No original_path found in meta file")
		return false
	}

	// 对于本地安装的版本（从压缩包安装的）
	if strings.HasPrefix(versionID, "local-") {
		fmt.Println("[PathCheck] This is a local installed version")
		// localPath 应该指向安装目录
		if localPath != "" {
			fmt.Printf("[PathCheck] Checking localPath: %s\n", localPath)
			_, statErr := os.Stat(localPath)
			if statErr == nil {
				fmt.Println("[PathCheck] localPath exists")
				return true
			}
			fmt.Printf("[PathCheck] localPath does not exist: %v\n", statErr)
			return false
		}

		// 如果没有localPath，尝试检查版本目录
		versionPath := m.paths.GetVersionPath(versionID)
		fmt.Printf("[PathCheck] No localPath, checking versionPath: %s\n", versionPath)
		_, statErr := os.Stat(versionPath)
		if statErr == nil {
			fmt.Println("[PathCheck] versionPath exists")
			return true
		}
		fmt.Printf("[PathCheck] versionPath does not exist: %v\n", statErr)
		return false
	}

	// 对于正常下载安装的版本，检查版本目录
	fmt.Println("[PathCheck] This is a normal installed version")
	versionPath := m.paths.GetVersionPath(versionID)
	fmt.Printf("[PathCheck] Checking versionPath: %s\n", versionPath)
	_, statErr := os.Stat(versionPath)
	if statErr == nil {
		fmt.Println("[PathCheck] versionPath exists")
		return true
	}
	fmt.Printf("[PathCheck] versionPath does not exist: %v\n", statErr)
	return false
}

// getVersion 获取版本（带已安装状态检查）
func (m *Manager) getVersionWithInstalledStatus(model *storage.VersionModel) Version {
	v := m.modelToVersion(model)
	// 检查目录是否存在来判断是否已安装
	v.Installed = m.paths.IsVersionInstalled(model.ID)
	return v
}
