package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	stdruntime "runtime"
	"path/filepath"
	"strings"
	"time"
	"SCLauncher/backend/appinfo"
	"SCLauncher/backend/background"
	"SCLauncher/backend/config"
	"SCLauncher/backend/game"
	"SCLauncher/backend/mod"
	"SCLauncher/backend/savegame"
	"SCLauncher/backend/skin"
	"SCLauncher/backend/storage"
	"SCLauncher/backend/texture"
	"SCLauncher/backend/version"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用结构体
type App struct {
	ctx         context.Context
	config      *config.Config
	paths       *config.Paths
	db          *storage.Database
	repository  *storage.Repository
	versionMgr  *version.Manager
	gameMgr     *game.GameManager
	modMgr      *mod.Manager
	skinMgr     *skin.Manager
	savegameMgr *savegame.Manager
	textureMgr  *texture.Manager
	backgroundMgr *background.Manager
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{}
}

// startup 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 获取应用数据目录（启动器目录下的 .Survivalcraft）
	appDataDir := config.GetAppDataDir()

	runtime.LogInfo(a.ctx, fmt.Sprintf("=== SCLauncher 启动 ==="))
	runtime.LogInfo(a.ctx, fmt.Sprintf("启动器目录: %s", appDataDir))

	// 加载配置
	configPath := filepath.Join(appDataDir, "config.json")
	cfg, err := config.Load(configPath)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("配置加载失败，创建新配置: %v", err))
		// 使用默认配置
		cfg = config.DefaultConfig()
		// 保存默认配置
		if saveErr := cfg.Save(); saveErr != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("配置保存失败: %v", saveErr))
		}
	}
	a.config = cfg
	a.paths = config.NewPaths(cfg)

	// 自动检测并设置语言（仅在首次启动时）
	if err := a.AutoDetectLanguage(); err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("自动检测语言失败: %v", err))
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("版本目录: %s", cfg.GetVersionsDir()))
	runtime.LogInfo(a.ctx, fmt.Sprintf("下载目录: %s", cfg.GetDownloadsDir()))

	// 初始化数据库
	db, err := storage.New(appDataDir)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("数据库初始化失败: %v", err))
		panic(err)
	}
	a.db = db
	a.repository = storage.NewRepository(db)

	// 初始化管理器
	a.versionMgr = version.NewManager(cfg, a.repository)
	a.gameMgr = game.NewGameManager(cfg, a.repository)
	a.gameMgr.SetContext(ctx) // 设置上下文用于发送事件
	a.modMgr = mod.NewManager(cfg)
	a.skinMgr = skin.NewManager(cfg)
	a.savegameMgr = savegame.NewManager(cfg)
	a.textureMgr = texture.NewManager(a.paths.GetVersionPath)
	a.backgroundMgr = background.NewManager(cfg)

	// 自动设置主要版本（如果没有的话）
	if err := a.versionMgr.AutoSetPrimaryVersion(); err != nil {
		runtime.LogInfo(a.ctx, fmt.Sprintf("自动设置主要版本: %v", err))
	}

	// 自动导入未导入的版本
	if err := a.AutoImportVersions(); err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("自动导入版本: %v", err))
	}

	runtime.LogInfo(a.ctx, "SCLauncher 初始化完成！")
}

// shutdown 应用关闭时调用
func (a *App) shutdown(ctx context.Context) {
	// 停止游戏
	if a.gameMgr.GetStatus() == game.StatusRunning {
		a.gameMgr.Stop()
	}

	// 关闭数据库
	if a.db != nil {
		a.db.Close()
	}
}

// ========== 配置相关 API ==========

// GetAppInfo 获取应用信息
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"version":  appinfo.Version,
		"repoOwner": appinfo.RepoOwner,
		"repoName":  appinfo.RepoName,
	}
}

// GetConfig 获取配置（返回相对路径格式给前端显示）
func (a *App) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"manifestUrl":     a.config.ManifestURL,
		"versionsDir":     a.config.GetRelativePathForDisplay(a.config.GetVersionsDir()),
		"dataDir":         a.config.GetRelativePathForDisplay(a.config.GetDataDir()),
		"downloadsDir":    a.config.GetRelativePathForDisplay(a.config.GetDownloadsDir()),
		"maxConcurrent":   a.config.MaxConcurrent,
		"currentVersion":  a.config.CurrentVersion,
		"theme":           a.config.Theme,
		"language":        a.config.Language,
		"backgroundImage": a.config.BackgroundImage,
	}
}

// SetManifestURL 设置清单文件 URL
func (a *App) SetManifestURL(url string) error {
	return a.config.SetManifestURL(url)
}

// SetMaxConcurrent 设置最大并发下载数
func (a *App) SetMaxConcurrent(max int) error {
	return a.config.SetMaxConcurrent(max)
}

// SetLanguage 设置语言
func (a *App) SetLanguage(lang string) error {
	return a.config.SetLanguage(lang)
}

// GetSystemLanguage 获取系统语言
func (a *App) GetSystemLanguage() string {
	// 获取系统语言环境变量
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}

	// 如果环境变量也没有，尝试通过平台特定方法获取
	if lang == "" {
		lang = a.detectSystemLanguage()
	}

	// 格式化语言代码 (例如: "zh_CN.UTF-8" -> "zh-CN")
	return a.formatSystemLanguage(lang)
}

// detectSystemLanguage 平台特定方法检测系统语言
func (a *App) detectSystemLanguage() string {
	switch stdruntime.GOOS {
	case "windows":
		return a.getWindowsLanguage()
	case "darwin":
		return a.getMacOSLanguage()
	default: // Linux 和其他
		return "en-US" // Linux 默认英语
	}
}

// getWindowsLanguage 获取Windows系统语言
func (a *App) getWindowsLanguage() string {
	// 尝试通过 PowerShell 获取语言列表
	cmd := exec.Command("powershell", "-Command", "Get-WinSystemLanguageList")
	output, err := cmd.Output()
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("Failed to get Windows language: %v", err))
		return "en-US"
	}

	// 解析输出，获取第一个语言
	langs := strings.Split(string(output), "\n")
	if len(langs) > 0 {
		// PowerShell 返回格式如 "en-US"
		langCode := strings.TrimSpace(langs[0])
		return a.formatSystemLanguage(langCode)
	}

	return "en-US"
}

// getMacOSLanguage 获取macOS系统语言
func (a *App) getMacOSLanguage() string {
	// 使用 defaults 命令获取系统语言
	cmd := exec.Command("defaults", "read", "-g", "AppleLanguages")
	output, err := cmd.Output()
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("Failed to get macOS language: %v", err))
		return "en-US"
	}

	// 解析输出，提取语言代码
	langs := strings.Split(string(output), ",")
	if len(langs) > 0 {
		// 返回第一个语言代码
		langCode := strings.Replace(strings.TrimSpace(langs[0]), "\"", "", -1)
		return a.formatSystemLanguage(langCode)
	}

	return "en-US"
}

// formatSystemLanguage 格式化系统语言代码为应用语言代码
func (a *App) formatSystemLanguage(systemLang string) string {
	// 移除编码部分 (例如: "zh_CN.UTF-8" -> "zh_CN")
	systemLang = strings.Split(systemLang, ".")[0]
	systemLang = strings.ReplaceAll(systemLang, "_", "-")
	systemLang = strings.ToLower(systemLang)

	// 语言映射表：系统语言代码 -> 应用语言代码
	langMap := map[string]string{
		"zh-cn": "zh-CN",
		"zh":    "zh-CN",
		"en-us": "en-US",
		"en":    "en-US",
		"en-gb": "en-US",
		"ru-ru": "ru-RU",
		"ru":    "ru-RU",
		"pt-br": "pt-BR",
		"pt":    "pt-BR",
		"hi-in": "hi-IN",
		"hi":    "hi-IN",
		"id-id": "id-ID",
		"id":    "id-ID",
		"ar-sa": "ar-SA",
		"ar":    "ar-SA",
	}

	// 查找映射
	for sysLang, appLang := range langMap {
		if strings.HasPrefix(systemLang, sysLang) {
			return appLang
		}
	}

	// 如果没有找到匹配的语言，返回默认的英语
	return "en-US"
}

// AutoDetectLanguage 自动检测并设置语言（仅在首次启动时）
func (a *App) AutoDetectLanguage() error {
	// 如果配置文件中语言已设置且不为空，则不覆盖
	if a.config.Language != "" {
		runtime.LogInfo(a.ctx, fmt.Sprintf("Language already set to: %s", a.config.Language))
		return nil
	}

	// 获取系统语言
	systemLang := a.GetSystemLanguage()
	runtime.LogInfo(a.ctx, fmt.Sprintf("Auto-detected system language: %s", systemLang))

	// 设置语言
	return a.config.SetLanguage(systemLang)
}

// GetPrimaryVersion 获取主要版本
func (a *App) GetPrimaryVersion() (map[string]interface{}, error) {
	model, err := a.repository.GetPrimaryVersion()
	if err != nil {
		// 没有主要版本不是错误，返回 nil
		return nil, nil
	}

	return map[string]interface{}{
		"id":           model.ID,
		"versionType":  model.VersionType,
		"gameVersion":  model.GameVersion,
		"subVersion":   model.SubVersion,
		"name":         model.Name,
		"size":         model.Size,
		"downloadUrl":  model.DownloadURL,
		"checksum":     model.Checksum,
		"fileFormat":   model.FileFormat,
		"illustrate":   model.Illustrate,
		"installed":    model.Installed,
		"isPrimary":    model.IsPrimary,
		"localPath":    model.LocalPath,
		"releaseDate":  model.CreatedAt,
	}, nil
}

// SetPrimaryVersion 设置主要版本
func (a *App) SetPrimaryVersion(versionID string) error {
	return a.repository.SetPrimaryVersion(versionID)
}

// AutoSetPrimaryVersion 自动设置主要版本
func (a *App) AutoSetPrimaryVersion() error {
	return a.versionMgr.AutoSetPrimaryVersion()
}

// ========== 版本管理 API ==========

// FetchVersions 从清单文件获取版本列表
func (a *App) FetchVersions() ([]version.Version, error) {
	return a.versionMgr.FetchVersions()
}

// GetVersions 获取所有版本
func (a *App) GetVersions() ([]version.Version, error) {
	return a.versionMgr.GetVersions()
}

// GetVersionsByType 按类型获取版本
func (a *App) GetVersionsByType(vtype string) ([]version.Version, error) {
	return a.versionMgr.GetVersionsByType(version.VersionType(vtype))
}

// GetInstalledVersions 获取已安装的版本
func (a *App) GetInstalledVersions() ([]version.Version, error) {
	return a.versionMgr.GetInstalledVersions()
}

// DownloadVersion 下载版本
func (a *App) DownloadVersion(versionID string) error {
	err := a.versionMgr.DownloadVersion(versionID, func(downloaded, total, speed int64) {
		// 发送进度事件到前端
		runtime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
			"versionId":  versionID,
			"downloaded": downloaded,
			"total":      total,
			"speed":      speed,
		})
	})

	// 下载完成后发送确认事件
	if err == nil {
		runtime.EventsEmit(a.ctx, "download:complete", map[string]interface{}{
			"versionId":  versionID,
			"originalId": versionID,
		})
	}

	return err
}

// DownloadVersionWithCustomName 下载版本（使用自定义名称）
func (a *App) DownloadVersionWithCustomName(versionID, customName string) error {
	// 生成唯一的版本 ID
	uniqueID := fmt.Sprintf("%s-%s", versionID, generateUniqueID())

	// 先发送开始事件，让前端知道uniqueID
	runtime.EventsEmit(a.ctx, "download:start", map[string]interface{}{
		"originalId": versionID,
		"uniqueId":   uniqueID,
		"customName": customName,
	})

	err := a.versionMgr.DownloadVersionWithCustomName(versionID, uniqueID, customName, func(downloaded, total, speed int64) {
		// 发送进度事件到前端（同时包含原始ID和唯一ID）
		runtime.EventsEmit(a.ctx, "download:progress", map[string]interface{}{
			"versionId":  uniqueID,
			"originalId": versionID,
			"downloaded": downloaded,
			"total":      total,
			"speed":      speed,
		})
	})

	// 下载完成后发送确认事件
	if err == nil {
		runtime.EventsEmit(a.ctx, "download:complete", map[string]interface{}{
			"versionId":  uniqueID,
			"originalId": versionID,
		})
	}

	return err
}

// generateUniqueID 生成唯一 ID
func generateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// InstallVersion 安装版本
func (a *App) InstallVersion(versionID string) error {
	return a.versionMgr.InstallVersion(versionID, func(current, total int64) {
		// 发送安装进度事件到前端
		runtime.EventsEmit(a.ctx, "install:progress", map[string]interface{}{
			"versionId": versionID,
			"current":   current,
			"total":     total,
		})
	})
}

// DeleteVersion 删除版本
func (a *App) DeleteVersion(versionID string) error {
	return a.versionMgr.DeleteVersion(versionID)
}

// RenameVersion 重命名版本
func (a *App) RenameVersion(versionID, newName string) error {
	return a.versionMgr.RenameVersion(versionID, newName)
}

// CancelDownload 取消下载
func (a *App) CancelDownload(versionID string) error {
	return a.versionMgr.CancelDownload(versionID)
}

// OpenVersionFolder 打开版本文件夹
func (a *App) OpenVersionFolder(versionID string) error {
	// 检查是否是导入的版本
	versionPath := a.paths.GetVersionPath(versionID)
	importedMetaFile := filepath.Join(versionPath, ".imported")

	var folderPath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				folderPath = originalPath
				break
			}
		}

		if folderPath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		folderPath = versionPath
	}

	runtime.BrowserOpenURL(a.ctx, "file:///"+folderPath)
	return nil
}

// OpenVersionModsFolder 打开版本的mods文件夹
func (a *App) OpenVersionModsFolder(versionID string) error {
	// 检查是否是导入的版本
	versionPath := a.paths.GetVersionPath(versionID)
	importedMetaFile := filepath.Join(versionPath, ".imported")

	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	modsPath := filepath.Join(basePath, "mods")
	runtime.BrowserOpenURL(a.ctx, "file:///"+modsPath)
	return nil
}

// ========== 游戏管理 API ==========

// SelectGameFolder 选择游戏文件夹
func (a *App) SelectGameFolder() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择游戏文件夹",
	})
	if err != nil {
		return "", err
	}
	return selection, nil
}

// AutoImportVersions 自动导入未导入的版本
func (a *App) AutoImportVersions() error {
	versionsDir := a.paths.GetVersionsDir()

	// 读取 versions 目录中的所有子目录
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return fmt.Errorf("读取 versions 目录失败: %w", err)
	}

	importedCount := 0

	for _, entry := range entries {
		// 跳过文件和隐藏目录
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		versionID := entry.Name()
		versionPath := filepath.Join(versionsDir, versionID)

		// 检查是否已经在数据库中
		existingVersion, err := a.repository.GetVersion(versionID)
		if err == nil && existingVersion != nil {
			// 已经在数据库中，跳过
			continue
		}

		// 检查是否有 .imported 标记文件（说明是从外部导入的）
		metaFile := filepath.Join(versionPath, ".imported")
		if _, err := os.Stat(metaFile); err == nil {
			// 这是外部导入的版本，已经在 ImportGameVersion 时处理过
			// 但可能数据库记录丢失了，尝试重新导入
			runtime.LogInfo(a.ctx, fmt.Sprintf("发现外部导入版本但数据库记录缺失: %s", versionID))
			continue
		}

		// 检查是否是有效的游戏文件夹（包含游戏可执行文件）
		exePath, err := a.findGameExecutableInFolder(versionPath)
		if err != nil {
			// 不是有效的游戏文件夹，跳过
			runtime.LogInfo(a.ctx, fmt.Sprintf("跳过无效的游戏文件夹: %s", versionPath))
			continue
		}

		// 这是一个有效的游戏文件夹，但未在数据库中，自动添加
		runtime.LogInfo(a.ctx, fmt.Sprintf("发现未记录的游戏版本: %s", versionID))

		// 尝试从文件夹名解析版本信息
		versionModel := &storage.VersionModel{
			ID:          versionID,
			Name:        versionID, // 使用文件夹名作为名称
			VersionType: "unknown", // 未知版本类型
			GameVersion: "unknown", // 未知游戏版本
			Installed:   true,
			Illustrate:  fmt.Sprintf("用户手动放置的游戏版本\n路径: %s\n游戏文件: %s", versionPath, filepath.Base(exePath)),
		}

		// 保存到数据库
		if err := a.repository.CreateVersion(versionModel); err != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("自动导入版本失败 %s: %v", versionID, err))
			continue
		}

		importedCount++
		runtime.LogInfo(a.ctx, fmt.Sprintf("成功自动注册游戏版本: %s", versionID))
	}

	if importedCount > 0 {
		runtime.LogInfo(a.ctx, fmt.Sprintf("自动导入完成！共导入 %d 个版本", importedCount))
	}

	return nil
}

// ImportGameVersion 导入游戏版本
func (a *App) ImportGameVersion(folderPath string) (string, error) {
	// 验证文件夹是否存在
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件夹不存在: %s", folderPath)
	}

	// 查找游戏可执行文件
	exePath, err := a.findGameExecutableInFolder(folderPath)
	if err != nil {
		return "", fmt.Errorf("未找到游戏文件: %v", err)
	}

	// 生成版本ID（使用时间戳）
	versionID := fmt.Sprintf("imported-%d", time.Now().UnixNano())

	// 创建版本模型
	versionModel := &storage.VersionModel{
		ID:          versionID,
		Name:        fmt.Sprintf("导入的游戏 (%s)", filepath.Base(folderPath)),
		VersionType: "unknown", // 未知版本类型
		GameVersion: "unknown", // 未知游戏版本
		Installed:   true,
		LocalPath:   folderPath, // 存储原始路径
		Illustrate:  fmt.Sprintf("从外部导入的游戏文件\n原始路径: %s\n游戏文件: %s", folderPath, filepath.Base(exePath)),
	}

	// 保存到数据库
	if err := a.repository.CreateVersion(versionModel); err != nil {
		return "", fmt.Errorf("保存版本信息失败: %v", err)
	}

	// 创建版本目录（用于存储元数据）
	versionPath := a.paths.GetVersionPath(versionID)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return "", fmt.Errorf("创建版本目录失败: %v", err)
	}

	// 创建标记文件来记录原始路径和exe路径
	metaFile := filepath.Join(versionPath, ".imported")
	metaContent := fmt.Sprintf("original_path=%s\nexe_path=%s\n", folderPath, exePath)
	if err := os.WriteFile(metaFile, []byte(metaContent), 0644); err != nil {
		return "", fmt.Errorf("写入元数据失败: %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功导入游戏版本: %s from %s", versionID, folderPath))
	return versionID, nil
}

// SelectArchiveFile 选择压缩包文件
func (a *App) SelectArchiveFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择游戏压缩包",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "压缩包文件",
				Pattern:     "*.zip;*.7z;*.rar",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	return filename, err
}

// InstallFromArchive 从压缩包安装游戏
func (a *App) InstallFromArchive(archivePath string, customName string) (string, error) {
	// 验证文件是否存在
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", archivePath)
	}

	// 生成版本ID
	versionID := fmt.Sprintf("local-%d", time.Now().UnixNano())

	// 创建临时目录用于解压
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("sc-install-%s", versionID))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	runtime.LogInfo(a.ctx, fmt.Sprintf("开始解压文件: %s 到 %s", archivePath, tempDir))

	// 解压文件（支持 zip、7z、rar 等多种格式）
	installer := version.NewInstaller()

	// 检查文件类型（仅用于提示信息）
	ext := strings.ToLower(filepath.Ext(archivePath))
	runtime.LogInfo(a.ctx, fmt.Sprintf("检测到压缩包格式: %s", ext))

	// 解压文件
	if err := installer.Install(archivePath, tempDir, func(current, total int64) {
		// 发送解压进度事件到前端
		runtime.EventsEmit(a.ctx, "install:progress", map[string]interface{}{
			"versionId": versionID,
			"current":   current,
			"total":     total,
		})
	}); err != nil {
		return "", fmt.Errorf("解压文件失败: %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("解压完成，检查游戏文件..."))

	// 检查解压后的目录是否包含游戏可执行文件
	exePath, err := a.findGameExecutableInFolder(tempDir)
	if err != nil {
		return "", fmt.Errorf("压缩包中未找到游戏文件(Survivalcraft.exe): %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("找到游戏文件: %s", exePath))

	// 创建最终版本目录
	versionPath := a.paths.GetVersionPath(versionID)
	if err := os.MkdirAll(filepath.Dir(versionPath), 0755); err != nil {
		return "", fmt.Errorf("创建版本目录失败: %v", err)
	}

	// 移动解压的文件到最终目录
	if err := os.Rename(tempDir, versionPath); err != nil {
		// 如果重命名失败（可能在不同的驱动器），尝试复制
		runtime.LogWarning(a.ctx, fmt.Sprintf("重命名失败，尝试复制文件: %v", err))
		if err := installer.Install(archivePath, versionPath, nil); err != nil {
			return "", fmt.Errorf("移动文件到最终目录失败: %v", err)
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("文件已移动到: %s", versionPath))

	// 如果没有提供自定义名称，使用压缩包文件名
	if customName == "" {
		customName = strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))
	}

	// 创建版本模型
	versionModel := &storage.VersionModel{
		ID:          versionID,
		Name:        customName,
		VersionType: "unknown", // 未知版本类型
		GameVersion: "unknown", // 未知游戏版本
		Installed:   true,
		LocalPath:   versionPath,
		Illustrate:  fmt.Sprintf("从压缩包安装\n原始文件: %s\n游戏文件: %s", filepath.Base(archivePath), filepath.Base(exePath)),
	}

	// 保存到数据库
	if err := a.repository.CreateVersion(versionModel); err != nil {
		return "", fmt.Errorf("保存版本信息失败: %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功从压缩包安装游戏版本: %s", versionID))
	return versionID, nil
}

// findGameExecutableInFolder 在指定文件夹中查找游戏可执行文件
func (a *App) findGameExecutableInFolder(folderPath string) (string, error) {
	var exePath string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// 检查是否是 .exe 文件且文件名包含 "Survivalcraft"
		if strings.HasSuffix(strings.ToLower(info.Name()), ".exe") &&
			strings.Contains(strings.ToLower(info.Name()), "survivalcraft") {
			exePath = path
			return filepath.SkipDir // 找到了，停止遍历
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if exePath == "" {
		return "", fmt.Errorf("在文件夹中未找到Survivalcraft游戏文件")
	}

	return exePath, nil
}

// LaunchGame 启动游戏
func (a *App) LaunchGame(versionID string) error {
	return a.gameMgr.Launch(versionID)
}

// StopGame 停止游戏
func (a *App) StopGame() error {
	return a.gameMgr.Stop()
}

// GetGameStatus 获取游戏状态
func (a *App) GetGameStatus() string {
	return string(a.gameMgr.GetStatus())
}

// GetGameProcessInfo 获取游戏进程信息
func (a *App) GetGameProcessInfo() (interface{}, error) {
	return a.gameMgr.GetProcessInfo()
}

// ========== 工具函数 API ==========

// FormatSize 格式化文件大小
func (a *App) FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// VersionExists 检查版本是否存在
func (a *App) VersionExists(versionID string) bool {
	return a.paths.VersionExists(versionID)
}

// ========== 版本更新检查 API ==========

// GitHubRelease GitHub 发布信息
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HtmlUrl     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
}

// CheckUpdate 检查更新
func (a *App) CheckUpdate() (map[string]interface{}, error) {
	return a.CheckUpdateWithForce(false)
}

// CheckUpdateWithForce 检查更新（可强制检查）
func (a *App) CheckUpdateWithForce(force bool) (map[string]interface{}, error) {
	// 如果不是强制检查，检查是否应该在期限内跳过
	if !force && !a.config.ShouldCheckUpdate() {
		runtime.LogInfo(a.ctx, "Update check is disabled by user preference")
		return map[string]interface{}{
			"currentVersion": appinfo.Version,
			"latestVersion":  appinfo.Version,
			"hasUpdate":      false,
			"skipped":        true,
		}, nil
	}

	// 获取最新 release
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", appinfo.RepoOwner, appinfo.RepoName)

	runtime.LogInfo(a.ctx, fmt.Sprintf("Checking for updates from: %s", url))
	runtime.LogInfo(a.ctx, fmt.Sprintf("Current version: %s", appinfo.Version))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("获取更新信息失败: %v", err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GitHub API 返回错误状态码: %d", resp.StatusCode)
		runtime.LogWarning(a.ctx, err.Error())
		return nil, err
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("解析 release 信息失败: %v", err))
		return nil, err
	}

	// 移除版本号前的 'v' 前缀
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	runtime.LogInfo(a.ctx, fmt.Sprintf("Latest version from GitHub: %s (tag: %s)", latestVersion, release.TagName))

	// 比较版本号
	hasUpdate := compareVersions(appinfo.Version, latestVersion)

	runtime.LogInfo(a.ctx, fmt.Sprintf("Has update: %v (current: %s, latest: %s)", hasUpdate, appinfo.Version, latestVersion))

	return map[string]interface{}{
		"currentVersion": appinfo.Version,
		"latestVersion":  latestVersion,
		"hasUpdate":      hasUpdate,
		"tagName":        release.TagName,
		"name":           release.Name,
		"url":            release.HtmlUrl,
		"publishedAt":    release.PublishedAt,
		"body":           release.Body,
	}, nil
}

// compareVersions 比较版本号，返回 true 表示有新版本
func compareVersions(current, latest string) bool {
	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := 0; i < 3; i++ {
		var currentVal, latestVal int

		if i < len(currentParts) {
			fmt.Sscanf(currentParts[i], "%d", &currentVal)
		}
		if i < len(latestParts) {
			fmt.Sscanf(latestParts[i], "%d", &latestVal)
		}

		if latestVal > currentVal {
			return true
		}
		if latestVal < currentVal {
			return false
		}
	}

	return false
}

// SetUpdateRemindDisabled 设置不再提醒更新（30天内不再检查）
func (a *App) SetUpdateRemindDisabled(disabled bool) error {
	var timestamp int64
	if disabled {
		// 设置为30天后的时间戳
		timestamp = time.Now().Add(30 * 24 * time.Hour).Unix()
	} else {
		timestamp = 0
	}
	return a.config.SetUpdateRemindDisableUntil(timestamp)
}

// ========== 模组管理 API ==========

// SelectModFile 选择模组文件
func (a *App) SelectModFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择模组文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "模组文件",
				Pattern:     "*.zip;*.scmod;*.disable",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	return filename, err
}

// GetMods 获取指定版本的模组列表
func (a *App) GetMods(versionID string) ([]mod.Mod, error) {
	return a.modMgr.GetMods(versionID)
}

// ImportMod 导入模组
func (a *App) ImportMod(versionID, sourcePath string) error {
	return a.modMgr.ImportMod(versionID, sourcePath)
}

// ToggleMod 切换模组启用/禁用状态
func (a *App) ToggleMod(versionID, modID string, enabled bool) error {
	return a.modMgr.ToggleMod(versionID, modID, enabled)
}

// DeleteMod 删除模组
func (a *App) DeleteMod(versionID, modID string) error {
	return a.modMgr.DeleteMod(versionID, modID)
}

// DownloadModFromURL 从URL下载模组
func (a *App) DownloadModFromURL(downloadURL, versionID, fileName string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载模组: %s -> %s", downloadURL, fileName))

	// 创建临时文件保存下载内容
	tempFile, err := os.CreateTemp("", "scmod-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保临时文件被删除

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 保存到临时文件
	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("模组下载完成，正在导入到版本: %s，文件名: %s", versionID, fileName))

	// 导入模组，使用正确的文件名
	if err := a.modMgr.ImportModWithName(versionID, tempPath, fileName); err != nil {
		return fmt.Errorf("导入模组失败: %w", err)
	}

	runtime.LogInfo(a.ctx, "模组下载并安装成功")
	return nil
}

// GetModSources 获取模组下载源配置
func (a *App) GetModSources() ([]map[string]interface{}, error) {
	// 获取应用数据目录（.Survivalcraft）
	appDataDir := config.GetAppDataDir()
	// 下载源配置目录
	sourcesDir := filepath.Join(appDataDir, "mod-sources")

	// 确保目录存在
	if _, err := os.Stat(sourcesDir); os.IsNotExist(err) {
		// 目录不存在，返回空数组
		return []map[string]interface{}{}, nil
	}

	// 读取目录中的所有 JSON 文件
	files, err := os.ReadDir(sourcesDir)
	if err != nil {
		return nil, fmt.Errorf("读取下载源目录失败: %w", err)
	}

	var sources []map[string]interface{}

	for _, file := range files {
		// 只处理 .json 文件
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 读取 JSON 文件
		filePath := filepath.Join(sourcesDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("读取下载源配置失败: %s, 错误: %v", filePath, err))
			continue
		}

		// 解析 JSON
		var source map[string]interface{}
		if err := json.Unmarshal(data, &source); err != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("解析下载源配置失败: %s, 错误: %v", filePath, err))
			continue
		}

		sources = append(sources, source)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功加载 %d 个自定义下载源", len(sources)))
	return sources, nil
}

// SaveModSources 保存模组下载源配置
func (a *App) SaveModSources(sources []map[string]interface{}) error {
	// 获取应用数据目录（.Survivalcraft）
	appDataDir := config.GetAppDataDir()
	// 下载源配置目录
	sourcesDir := filepath.Join(appDataDir, "mod-sources")

	// 确保目录存在
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return fmt.Errorf("创建下载源目录失败: %w", err)
	}

	// 收集有效的源 ID
	validSourceIDs := make(map[string]bool)

	// 保存每个下载源
	for _, source := range sources {
		// 获取源 ID
		sourceID, ok := source["id"].(string)
		if !ok || sourceID == "" {
			continue
		}

		// 移除内置源，只保存自定义源
		if sourceID == "suancaixianyu" {
			continue
		}

		validSourceIDs[sourceID] = true

		// 序列化为 JSON
		data, err := json.MarshalIndent(source, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化下载源配置失败: %w", err)
		}

		// 保存到文件
		filePath := filepath.Join(sourcesDir, fmt.Sprintf("%s.json", sourceID))
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("保存下载源配置失败: %w", err)
		}

		runtime.LogInfo(a.ctx, fmt.Sprintf("保存下载源: %s -> %s", sourceID, filePath))
	}

	// 清理不再存在的源文件
	files, err := os.ReadDir(sourcesDir)
	if err != nil {
		return fmt.Errorf("读取下载源目录失败: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 从文件名提取源 ID（去掉 .json 后缀）
		sourceID := strings.TrimSuffix(file.Name(), ".json")

		// 如果这个源不在有效源列表中，删除文件
		if !validSourceIDs[sourceID] {
			filePath := filepath.Join(sourcesDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				runtime.LogWarning(a.ctx, fmt.Sprintf("删除旧源配置文件失败: %s, 错误: %v", filePath, err))
			} else {
				runtime.LogInfo(a.ctx, fmt.Sprintf("删除旧源配置文件: %s", filePath))
			}
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功保存 %d 个自定义下载源", len(sources)))
	return nil
}

// ========== 皮肤管理 API ==========

// SelectSkinFile 选择皮肤文件
func (a *App) SelectSkinFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择皮肤文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "皮肤文件",
				Pattern:     "*.scskin",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	return filename, err
}

// GetSkins 获取所有皮肤列表
func (a *App) GetSkins() ([]skin.Skin, error) {
	return a.skinMgr.GetSkins()
}

// ImportSkin 导入皮肤
func (a *App) ImportSkin(sourcePath string) error {
	return a.skinMgr.UploadSkin(sourcePath)
}

// DeleteSkin 删除皮肤
func (a *App) DeleteSkin(fileName string) error {
	return a.skinMgr.DeleteSkin(fileName)
}

// SyncSkinsToGame 同步皮肤到游戏目录
func (a *App) SyncSkinsToGame(versionID string) error {
	return a.skinMgr.SyncSkinsToGame(versionID)
}

// GetSkinImage 获取皮肤图片的base64编码
func (a *App) GetSkinImage(fileName string) (string, error) {
	return a.skinMgr.GetSkinImage(fileName)
}

// DownloadSkinFromURL 从URL下载皮肤
func (a *App) DownloadSkinFromURL(downloadURL, fileName string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载皮肤: %s -> %s", downloadURL, fileName))

	// 验证文件扩展名
	if !strings.HasSuffix(strings.ToLower(fileName), ".scskin") {
		return fmt.Errorf("invalid file extension: %s", fileName)
	}

	// 创建临时文件保存下载内容
	tempFile, err := os.CreateTemp("", "scskin-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保临时文件被删除

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 保存到临时文件
	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("皮肤下载完成，正在导入: %s", fileName))

	// 确保皮肤目录存在
	skinsDir := filepath.Join(config.GetAppDataDir(), "skins")
	if err := os.MkdirAll(skinsDir, 0755); err != nil {
		return fmt.Errorf("创建皮肤目录失败: %w", err)
	}

	// 目标文件路径
	destPath := filepath.Join(skinsDir, fileName)

	// 检查文件是否已存在
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("皮肤文件已存在: %s", fileName)
	}

	// 复制文件
	sourceFile, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	// 复制内容
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	runtime.LogInfo(a.ctx, "皮肤下载并安装成功")
	return nil
}

// ========== 背景图片管理 API ==========

// SelectBackgroundFile 选择背景图片文件
func (a *App) SelectBackgroundFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择背景图片",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "图片文件",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})
	return filename, err
}

// SetBackground 设置背景图片
func (a *App) SetBackground(sourcePath string) (string, error) {
	return a.backgroundMgr.SetBackground(sourcePath)
}

// ClearBackground 清除背景图片
func (a *App) ClearBackground() error {
	return a.backgroundMgr.ClearBackground()
}

// GetBackgroundImage 获取背景图片路径
func (a *App) GetBackgroundImage() string {
	return a.backgroundMgr.GetBackgroundImage()
}

// HasBackground 检查是否设置了背景图片
func (a *App) HasBackground() bool {
	return a.backgroundMgr.HasBackground()
}

// GetBackgroundImageBase64 获取背景图片的base64编码
func (a *App) GetBackgroundImageBase64() (string, error) {
	return a.backgroundMgr.GetBackgroundImageBase64()
}

// ========== 存档管理 API ==========

// GetSaveGames 获取指定版本的存档列表
func (a *App) GetSaveGames(versionID string) ([]savegame.SaveGame, error) {
	return a.savegameMgr.GetSaveGames(versionID)
}

// DeleteSaveGame 删除存档
func (a *App) DeleteSaveGame(versionID, saveID string) error {
	return a.savegameMgr.DeleteSaveGame(versionID, saveID)
}

// OpenSaveGameFolder 打开存档文件夹
func (a *App) OpenSaveGameFolder(versionID, saveID string) error {
	folderPath, err := a.savegameMgr.GetSaveGamePath(versionID, saveID)
	if err != nil {
		return err
	}

	runtime.BrowserOpenURL(a.ctx, "file:///"+folderPath)
	return nil
}

// RenameSaveGame 重命名存档
func (a *App) RenameSaveGame(versionID, saveID, newName string) error {
	return a.savegameMgr.RenameSaveGame(versionID, saveID, newName)
}

// ExportSaveGame 导出存档
func (a *App) ExportSaveGame(versionID, saveID string) (bool, error) {
	// 获取存档信息以使用存档名称作为默认文件名
	saveGames, err := a.savegameMgr.GetSaveGames(versionID)
	if err != nil {
		return false, fmt.Errorf("failed to get save games: %w", err)
	}

	// 查找对应的存档
	var saveName string
	for _, sg := range saveGames {
		if sg.ID == saveID {
			saveName = sg.Name
			break
		}
	}

	// 如果没有找到，使用ID作为名称
	if saveName == "" {
		saveName = saveID
	}

	// 让用户选择保存位置和文件名
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "选择导出位置",
		DefaultFilename: saveName + ".scworld",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "存档文件",
				Pattern:     "*.scworld",
			},
		},
	})

	if err != nil {
		return false, fmt.Errorf("failed to open save dialog: %w", err)
	}

	if filename == "" {
		return false, nil // 用户取消，返回 false 但不返回错误
	}

	err = a.savegameMgr.ExportSaveGame(versionID, saveID, filename)
	if err != nil {
		return false, err
	}

	return true, nil // 导出成功
}

// ImportSaveGame 导入存档
func (a *App) ImportSaveGame(versionID, sourcePath string) error {
	return a.savegameMgr.ImportSaveGame(versionID, sourcePath)
}

// DownloadSaveGameFromURL 从URL下载存档
func (a *App) DownloadSaveGameFromURL(downloadURL, versionID, fileName string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载存档: %s -> %s", downloadURL, fileName))

	// 确定文件扩展名
	var ext string
	lowerFileName := strings.ToLower(fileName)
	if strings.HasSuffix(lowerFileName, ".scworld") {
		ext = ".scworld"
	} else if strings.HasSuffix(lowerFileName, ".scword") {
		ext = ".scword"
	} else if strings.HasSuffix(lowerFileName, ".zip") {
		ext = ".zip"
	} else {
		// 如果没有扩展名或扩展名未知，默认使用.scworld
		ext = ".scworld"
	}

	// 创建临时文件，使用正确的扩展名
	tempFile, err := os.CreateTemp("", "scsave-*"+ext)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保临时文件被删除

	runtime.LogInfo(a.ctx, fmt.Sprintf("临时文件路径: %s", tempPath))

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，服务器返回状态码: %d", resp.StatusCode)
	}

	// 写入临时文件
	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("保存下载内容失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("存档下载完成，正在导入: %s", tempPath))

	// 导入存档
	err = a.savegameMgr.ImportSaveGame(versionID, tempPath)
	if err != nil {
		return fmt.Errorf("导入存档失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("存档导入成功: %s", fileName))
	return nil
}

// SelectSaveGameFile 选择要导入的存档文件
func (a *App) SelectSaveGameFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要导入的存档文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "存档文件",
				Pattern:     "*.scworld;*.scword;*.zip",
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}

	if filename == "" {
		return "", nil // 用户取消
	}

	return filename, nil
}

// PreviewSaveGame 预览存档信息
func (a *App) PreviewSaveGame(sourcePath string) (savegame.SaveGame, error) {
	return a.savegameMgr.PreviewSaveGame(sourcePath)
}

// ========== 家具管理 API ==========

// Furniture 家具信息
type Furniture struct {
	ID       string `json:"id"`       // 家具 ID（文件名，不含扩展名）
	Name     string `json:"name"`     // 显示名称
	FileName string `json:"fileName"` // 完整文件名（含扩展名）
}

// GetFurnitures 获取指定版本的家具列表
func (a *App) GetFurnitures(versionID string) ([]Furniture, error) {
	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return nil, fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	// 如果都没有找到，返回 nil（表示文件夹不存在）
	if furniturePath == "" {
		return nil, nil
	}

	// 读取目录中的所有文件
	entries, err := os.ReadDir(furniturePath)
	if err != nil {
		return nil, fmt.Errorf("读取家具目录失败: %w", err)
	}

	// 初始化为空切片，确保JSON序列化时返回[]而不是null
	furnitures := make([]Furniture, 0)
	for _, entry := range entries {
		// 跳过目录和隐藏文件
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// 只处理 .scfpack 文件
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".scfpack") {
			continue
		}

		fileName := entry.Name()
		// ID 是不含扩展名的文件名
		id := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		furnitures = append(furnitures, Furniture{
			ID:       id,
			Name:     id, // 显示名称使用文件名
			FileName: fileName,
		})
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("找到 %d 个家具包", len(furnitures)))
	return furnitures, nil
}

// SelectFurnitureFile 选择要导入的家具文件
func (a *App) SelectFurnitureFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择家具文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "家具包文件",
				Pattern:     "*.scfpack",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}

	if filename == "" {
		return "", nil // 用户取消
	}

	return filename, nil
}

// ImportFurniture 导入家具
func (a *App) ImportFurniture(versionID, sourcePath string) error {
	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	// 如果文件夹不存在，返回错误提示用户先启动游戏
	if furniturePath == "" {
		return fmt.Errorf("家具包文件夹不存在，请先启动一次游戏")
	}

	// 检查源文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("源文件不存在: %s", sourcePath)
	}

	// 获取文件名
	fileName := filepath.Base(sourcePath)

	// 目标文件路径
	destPath := filepath.Join(furniturePath, fileName)

	// 复制文件
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功导入家具包: %s -> %s", fileName, furniturePath))
	return nil
}

// DownloadFurnitureFromURL 从URL下载家具
func (a *App) DownloadFurnitureFromURL(downloadURL, versionID, fileName string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载家具: %s -> %s", downloadURL, fileName))

	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	// 如果文件夹不存在，返回错误
	if furniturePath == "" {
		return fmt.Errorf("家具包文件夹不存在，请先启动一次游戏")
	}

	// 创建临时文件保存下载内容
	tempFile, err := os.CreateTemp("", "scfurniture-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保临时文件被删除

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 保存到临时文件
	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("家具下载完成，正在导入到版本: %s，文件名: %s", versionID, fileName))

	// 目标文件路径
	destPath := filepath.Join(furniturePath, fileName)

	// 移动临时文件到目标位置
	if err := os.Rename(tempPath, destPath); err != nil {
		// 如果重命名失败，尝试复制
		srcFile, err := os.Open(tempPath)
		if err != nil {
			return fmt.Errorf("打开临时文件失败: %w", err)
		}
		defer srcFile.Close()

		dstFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %w", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("复制文件失败: %w", err)
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("家具下载并安装成功: %s", fileName))
	return nil
}

// DeleteFurniture 删除家具
func (a *App) DeleteFurniture(versionID, furnitureID string) error {
	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	if furniturePath == "" {
		return fmt.Errorf("家具包文件夹不存在")
	}

	// 删除文件（需要添加 .scfpack 扩展名）
	filePath := filepath.Join(furniturePath, furnitureID+".scfpack")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除家具包失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功删除家具包: %s", furnitureID))
	return nil
}

// RenameFurniture 重命名家具
func (a *App) RenameFurniture(versionID, furnitureID, newName string) error {
	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	if furniturePath == "" {
		return fmt.Errorf("家具包文件夹不存在")
	}

	// 重命名文件
	oldPath := filepath.Join(furniturePath, furnitureID+".scfpack")
	newPath := filepath.Join(furniturePath, newName+".scfpack")

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重命名家具包失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功重命名家具包: %s -> %s", furnitureID, newName))
	return nil
}

// OpenFurnitureFolder 打开家具文件夹
func (a *App) OpenFurnitureFolder(versionID string) error {
	// 获取版本路径
	versionPath := a.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	var basePath string
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				basePath = originalPath
				break
			}
		}

		if basePath == "" {
			return fmt.Errorf("invalid import metadata file")
		}
	} else {
		// 正常安装的版本，使用版本目录
		basePath = versionPath
	}

	// 尝试两个可能的路径
	possiblePaths := []string{
		filepath.Join(basePath, "doc", "FurniturePacks"),
		filepath.Join(basePath, "FurniturePacks"),
	}

	var furniturePath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			furniturePath = path
			break
		}
	}

	if furniturePath == "" {
		return fmt.Errorf("家具包文件夹不存在，请先启动一次游戏")
	}

	runtime.BrowserOpenURL(a.ctx, "file:///"+furniturePath)
	return nil
}

// ========== 材质管理 API ==========

// Texture 材质信息
type Texture struct {
	ID       string `json:"id"`       // 材质 ID（文件名，不含扩展名）
	Name     string `json:"name"`     // 显示名称
	FileName string `json:"fileName"` // 完整文件名（含扩展名）
}

// GetTextures 获取指定版本的材质列表
func (a *App) GetTextures(versionID string) ([]Texture, error) {
	textures, err := a.textureMgr.GetTextures(versionID)
	if err != nil {
		return nil, err
	}

	// 如果材质文件夹不存在，返回 nil
	if textures == nil {
		return nil, nil
	}

	// 转换为 API 类型
	result := make([]Texture, len(textures))
	for i, t := range textures {
		result[i] = Texture{
			ID:       t.ID,
			Name:     t.Name,
			FileName: t.FileName,
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("找到 %d 个材质包", len(textures)))
	return result, nil
}

// OpenTextureFolder 打开材质文件夹
func (a *App) OpenTextureFolder(versionID string) error {
	texturePath, err := a.textureMgr.GetTexturePath(versionID)
	if err != nil {
		return err
	}

	if texturePath == "" {
		return fmt.Errorf("材质包文件夹不存在，请先启动一次游戏")
	}

	runtime.BrowserOpenURL(a.ctx, "file:///"+texturePath)
	return nil
}

// DeleteTexture 删除材质
func (a *App) DeleteTexture(versionID, textureID string) error {
	return a.textureMgr.DeleteTexture(versionID, textureID)
}

// RenameTexture 重命名材质
func (a *App) RenameTexture(versionID, textureID, newName string) error {
	return a.textureMgr.RenameTexture(versionID, textureID, newName)
}

// SelectTextureFile 选择要导入的材质文件
func (a *App) SelectTextureFile() (string, error) {
	filename, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择材质文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "材质包文件",
				Pattern:     "*.scbtex",
			},
			{
				DisplayName: "所有文件",
				Pattern:     "*.*",
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}

	if filename == "" {
		return "", nil // 用户取消
	}

	return filename, nil
}

// ImportTexture 导入材质
func (a *App) ImportTexture(versionID, sourcePath string) error {
	// 获取材质文件夹路径
	texturePath, err := a.textureMgr.GetTexturePath(versionID)
	if err != nil {
		return err
	}

	// 如果文件夹不存在，返回错误提示用户先启动游戏
	if texturePath == "" {
		return fmt.Errorf("材质包文件夹不存在，请先启动一次游戏")
	}

	// 检查源文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("源文件不存在: %s", sourcePath)
	}

	// 获取文件名
	fileName := filepath.Base(sourcePath)

	// 目标文件路径
	destPath := filepath.Join(texturePath, fileName)

	// 复制文件
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("成功导入材质包: %s", fileName))
	return nil
}

// DownloadTextureFromURL 从URL下载材质
func (a *App) DownloadTextureFromURL(downloadURL, versionID, fileName string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("开始下载材质: %s -> %s", downloadURL, fileName))

	// 获取材质文件夹路径
	texturePath, err := a.textureMgr.GetTexturePath(versionID)
	if err != nil {
		return err
	}

	// 如果文件夹不存在，返回错误
	if texturePath == "" {
		return fmt.Errorf("材质包文件夹不存在，请先启动一次游戏")
	}

	// 创建临时文件保存下载内容
	tempFile, err := os.CreateTemp("", "sctexture-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // 确保临时文件被删除

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 保存到临时文件
	_, err = io.Copy(tempFile, resp.Body)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("材质下载完成，正在导入到版本: %s，文件名: %s", versionID, fileName))

	// 目标文件路径
	destPath := filepath.Join(texturePath, fileName)

	// 移动临时文件到目标位置
	if err := os.Rename(tempPath, destPath); err != nil {
		// 如果重命名失败，尝试复制
		srcFile, err := os.Open(tempPath)
		if err != nil {
			return fmt.Errorf("打开临时文件失败: %w", err)
		}
		defer srcFile.Close()

		dstFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("创建目标文件失败: %w", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("复制文件失败: %w", err)
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("材质下载并安装成功: %s", fileName))
	return nil
}
