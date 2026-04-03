package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"SCLauncher/backend/skin"
	"SCLauncher/backend/storage"
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
	a.skinMgr = skin.NewManager(cfg)
	a.backgroundMgr = background.NewManager(cfg)

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

// GetAppInfo 获取应用信息
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"version":  appinfo.Version,
		"repoOwner": appinfo.RepoOwner,
		"repoName":  appinfo.RepoName,
	}
}

// GetConfig 获取配置
func (a *App) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"manifestUrl":     a.config.ManifestURL,
		"versionsDir":     a.config.VersionsDir,
		"dataDir":         a.config.DataDir,
		"downloadsDir":    a.config.DownloadsDir,
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
	versionPath := a.paths.GetVersionPath(versionID)
	runtime.BrowserOpenURL(a.ctx, "file:///"+versionPath)
	return nil
}

// OpenVersionModsFolder 打开版本的mods文件夹
func (a *App) OpenVersionModsFolder(versionID string) error {
	modsPath := filepath.Join(a.paths.GetVersionPath(versionID), "mods")
	runtime.BrowserOpenURL(a.ctx, "file:///"+modsPath)
	return nil
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
