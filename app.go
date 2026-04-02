package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
	"SCLauncher/backend/config"
	"SCLauncher/backend/game"
	"SCLauncher/backend/mod"
	"SCLauncher/backend/storage"
	"SCLauncher/backend/version"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用结构体
type App struct {
	ctx        context.Context
	config     *config.Config
	paths      *config.Paths
	db         *storage.Database
	repository *storage.Repository
	versionMgr *version.Manager
	gameMgr    *game.GameManager
	modMgr     *mod.Manager
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

	runtime.LogInfo(a.ctx, fmt.Sprintf("版本目录: %s", cfg.VersionsDir))
	runtime.LogInfo(a.ctx, fmt.Sprintf("下载目录: %s", cfg.DownloadsDir))

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

	// 自动设置主要版本（如果没有的话）
	if err := a.versionMgr.AutoSetPrimaryVersion(); err != nil {
		runtime.LogInfo(a.ctx, fmt.Sprintf("自动设置主要版本: %v", err))
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

// GetConfig 获取配置
func (a *App) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"manifestUrl":    a.config.ManifestURL,
		"versionsDir":    a.config.VersionsDir,
		"dataDir":        a.config.DataDir,
		"downloadsDir":   a.config.DownloadsDir,
		"maxConcurrent":  a.config.MaxConcurrent,
		"currentVersion": a.config.CurrentVersion,
		"theme":          a.config.Theme,
		"language":       a.config.Language,
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

// SetCurrentVersion 设置当前选中的版本
func (a *App) SetCurrentVersion(versionID string) error {
	return a.config.SetCurrentVersion(versionID)
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

// ========== 游戏管理 API ==========

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
