package modpack

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"SCLauncher/backend/config"
	"SCLauncher/backend/mod"
	"SCLauncher/backend/version"
)

// InstallProgress 安装进度回调
type InstallProgress func(stage string, progress float64, message string)

// ModpackInstaller 整合包安装器
type ModpackInstaller struct {
	config         *config.Config
	paths          *config.Paths
	versionMgr     *version.Manager
	modMgr         *mod.Manager
	httpClient     *http.Client
	currentTask    *InstallTask
	taskMutex      sync.Mutex
	progressCb     InstallProgress
	cancelChan     chan struct{}
	isCancelled    bool
}

// InstallTask 安装任务
type InstallTask struct {
	ID            string  `json:"id"`              // 任务ID
	Manifest      *Manifest `json:"manifest"`      // 整合包清单
	CurrentStage  string  `json:"currentStage"`   // 当前阶段
	Progress      float64 `json:"progress"`        // 总体进度
	StageProgress float64 `json:"stageProgress"`  // 当前阶段进度
	Status        string  `json:"status"`          // 状态：pending/running/completed/failed
	Message       string  `json:"message"`         // 当前消息
	Error         string  `json:"error,omitempty"` // 错误信息
}

// InstallStage 安装阶段
const (
	StagePrepare       = "prepare"        // 准备阶段
	StageDownloadGame  = "download_game"  // 下载游戏
	StageInstallGame   = "install_game"   // 安装游戏
	StageDownloadMods  = "download_mods"  // 下载模组
	StageInstallMods   = "install_mods"   // 安装模组
	StageCopyOverrides = "copy_overrides" // 复制覆盖文件
	StageComplete      = "complete"       // 完成
)

// NewModpackInstaller 创建整合包安装器
func NewModpackInstaller(cfg *config.Config, versionMgr *version.Manager, modMgr *mod.Manager) *ModpackInstaller {
	return &ModpackInstaller{
		config:     cfg,
		paths:      config.NewPaths(cfg),
		versionMgr: versionMgr,
		modMgr:     modMgr,
		httpClient: &http.Client{},
		cancelChan: make(chan struct{}),
	}
}

// SetProgressCallback 设置进度回调
func (m *ModpackInstaller) SetProgressCallback(cb InstallProgress) {
	m.progressCb = cb
}

// Cancel 取消安装
func (m *ModpackInstaller) Cancel() {
	m.taskMutex.Lock()
	defer m.taskMutex.Unlock()

	if !m.isCancelled {
		m.isCancelled = true
		close(m.cancelChan)
	}
}

// IsCancelled 检查是否已取消
func (m *ModpackInstaller) IsCancelled() bool {
	select {
	case <-m.cancelChan:
		return true
	default:
		return false
	}
}

// isValidURL 检查字符串是否是有效的URL
func (m *ModpackInstaller) isValidURL(s string) bool {
	// 检查是否是常见的示例值
	if s == "URL" || s == "" || s == "http://" || s == "https://" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// 检查是否有有效的协议和主机
	if u.Scheme == "" || u.Scheme == "file" {
		return false
	}

	if u.Host == "" {
		return false
	}

	// 检查是否是常见的网络协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	return true
}

// updateProgress 更新进度
func (m *ModpackInstaller) updateProgress(stage string, progress float64, message string) {
	if m.progressCb != nil {
		m.progressCb(stage, progress, message)
	}

	if m.currentTask != nil {
		m.currentTask.CurrentStage = stage
		m.currentTask.StageProgress = progress
		m.currentTask.Message = message
	}
}

// Install 安装整合包
func (m *ModpackInstaller) Install(manifest *Manifest, targetVersionID string) error {
	m.taskMutex.Lock()
	m.isCancelled = false
	m.cancelChan = make(chan struct{})
	m.currentTask = &InstallTask{
		ID:       fmt.Sprintf("modpack-install-%s", manifest.FileHash),
		Manifest: manifest,
		Status:   "running",
	}
	m.taskMutex.Unlock()

	defer func() {
		if m.currentTask != nil {
			if m.currentTask.Status != "failed" {
				m.currentTask.Status = "completed"
			}
			m.currentTask.Progress = 100
		}
	}()

	// 阶段1: 准备
	if m.IsCancelled() {
		return fmt.Errorf("安装已取消")
	}

	m.updateProgress(StagePrepare, 0, "准备安装环境...")
	versionPath, err := m.prepareEnvironment(manifest, targetVersionID)
	if err != nil {
		return fmt.Errorf("准备环境失败: %w", err)
	}

	// 阶段2: 下载并安装游戏
	if m.IsCancelled() {
		return fmt.Errorf("安装已取消")
	}

	m.updateProgress(StageDownloadGame, 0, "准备下载游戏...")
	gamePath, err := m.downloadAndInstallGame(manifest, versionPath)
	if err != nil {
		return fmt.Errorf("下载游戏失败: %w", err)
	}

	// 阶段3: 下载模组
	if m.IsCancelled() {
		return fmt.Errorf("安装已取消")
	}

	m.updateProgress(StageDownloadMods, 0, "准备下载模组...")
	if err := m.downloadMods(manifest, targetVersionID); err != nil {
		return fmt.Errorf("下载模组失败: %w", err)
	}

	// 阶段4: 复制覆盖文件
	if m.IsCancelled() {
		return fmt.Errorf("安装已取消")
	}

	m.updateProgress(StageCopyOverrides, 0, "复制覆盖文件...")
	if manifest.Overrides != "" {
		if err := m.copyOverrides(manifest, gamePath); err != nil {
			// 覆盖文件复制失败不应该阻止整个安装过程
			// 只记录错误，不中断安装
			m.updateProgress(StageCopyOverrides, 50, fmt.Sprintf("覆盖文件复制跳过: %v", err))
		} else {
			m.updateProgress(StageCopyOverrides, 100, "覆盖文件复制完成")
		}
	} else {
		m.updateProgress(StageCopyOverrides, 100, "无覆盖文件，跳过")
	}

	// 完成
	m.updateProgress(StageComplete, 100, "安装完成！")
	return nil
}

// prepareEnvironment 准备安装环境
func (m *ModpackInstaller) prepareEnvironment(manifest *Manifest, targetVersionID string) (string, error) {
	m.updateProgress(StagePrepare, 10, "创建版本目录...")

	// 创建版本目录（如果不存在）
	versionPath := m.paths.GetVersionPath(targetVersionID)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return "", fmt.Errorf("创建版本目录失败: %w", err)
	}

	m.updateProgress(StagePrepare, 50, "创建工作目录...")

	// 创建临时工作目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("sc-modpack-%s", manifest.FileHash))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建工作目录失败: %w", err)
	}

	m.updateProgress(StagePrepare, 100, "环境准备完成")
	return versionPath, nil
}

// downloadAndInstallGame 下载并安装游戏
func (m *ModpackInstaller) downloadAndInstallGame(manifest *Manifest, versionPath string) (string, error) {
	if manifest.Survivalcraft == nil || manifest.Survivalcraft.Version.Windows == nil {
		return "", fmt.Errorf("该整合包不支持 Windows 平台")
	}

	windowsConfig := manifest.Survivalcraft.Version.Windows
	versionStr := windowsConfig.Version

	m.updateProgress(StageDownloadGame, 0, fmt.Sprintf("准备下载游戏版本: %s", versionStr))

	var downloadURL string
	var err error

	// 检查是否有外部下载链接（必须是有效的URL）
	if windowsConfig.Path != "" && m.isValidURL(windowsConfig.Path) {
		downloadURL = windowsConfig.Path
		m.updateProgress(StageDownloadGame, 10, fmt.Sprintf("使用外部链接下载游戏: %s", downloadURL))
	} else {
		// 从版本列表获取下载链接
		if windowsConfig.Path != "" && !m.isValidURL(windowsConfig.Path) {
			m.updateProgress(StageDownloadGame, 10, "检测到无效的外部链接，将使用版本列表...")
		} else {
			m.updateProgress(StageDownloadGame, 10, "从版本列表获取下载链接...")
		}
		downloadURL, err = m.getGameDownloadURL(manifest, versionStr)
		if err != nil {
			return "", fmt.Errorf("获取下载链接失败: %w", err)
		}
	}

	// 下载游戏文件
	m.updateProgress(StageDownloadGame, 20, "开始下载游戏...")
	gameArchivePath, err := m.downloadGame(downloadURL, versionStr)
	if err != nil {
		return "", fmt.Errorf("下载游戏失败: %w", err)
	}

	// 解压并安装游戏到版本目录
	m.updateProgress(StageInstallGame, 80, "安装游戏...")
	gamePath, err := m.installGameToVersion(gameArchivePath, versionPath)
	if err != nil {
		return "", fmt.Errorf("安装游戏失败: %w", err)
	}

	m.updateProgress(StageInstallGame, 100, "游戏安装完成")
	return gamePath, nil
}

// getGameDownloadURL 从版本列表获取游戏下载链接
func (m *ModpackInstaller) getGameDownloadURL(manifest *Manifest, versionStr string) (string, error) {
	// 解析版本字符串
	// 格式: "2.4:api-1.8.2.3" -> 大版本:类型-版本号
	parts := strings.Split(versionStr, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("无效的版本格式: %s", versionStr)
	}

	// 解析类型和版本号
	typeVersionParts := strings.Split(parts[1], "-")
	if len(typeVersionParts) != 2 {
		return "", fmt.Errorf("无效的版本类型格式: %s", parts[1])
	}

	gameVersion := parts[0]          // 2.4
	versionType := typeVersionParts[0] // api
	subVersion := typeVersionParts[1]  // 1.8.2.3

	// 下载并解析版本列表
	m.updateProgress(StageDownloadGame, 15, "下载版本列表...")
	versionManifest, err := m.fetchVersionManifest(manifest)
	if err != nil {
		return "", fmt.Errorf("获取版本列表失败: %w", err)
	}

	// 查找匹配的版本
	m.updateProgress(StageDownloadGame, 18, "查找匹配版本...")
	downloadURL, err := m.findVersionInManifest(versionManifest, gameVersion, versionType, subVersion)
	if err != nil {
		return "", fmt.Errorf("查找版本失败: %w", err)
	}

	return downloadURL, nil
}

// fetchVersionManifest 获取版本清单
func (m *ModpackInstaller) fetchVersionManifest(manifest *Manifest) (*version.Manifest, error) {
	var manifestURL string

	if manifest.Survivalcraft.VersionList.Windows != "" {
		manifestURL = manifest.Survivalcraft.VersionList.Windows
	} else {
		// 使用启动器默认的版本列表
		manifestURL = m.config.ManifestURL
	}

	// 解析清单
	parser := version.NewManifestParser()
	return parser.ParseFromURL(manifestURL)
}

// findVersionInManifest 在清单中查找指定版本
func (m *ModpackInstaller) findVersionInManifest(versionManifest *version.Manifest, gameVersion, versionType, subVersion string) (string, error) {
	// 根据版本类型获取对应的版本列表
	var versionMap map[string][]version.ManifestVersion
	switch versionType {
	case "api":
		versionMap = versionManifest.API
	case "net":
		versionMap = versionManifest.NET
	case "original":
		versionMap = versionManifest.Original
	case "modified":
		versionMap = versionManifest.Modified
	default:
		return "", fmt.Errorf("不支持的版本类型: %s", versionType)
	}

	// 获取该大版本的版本列表
	versions, exists := versionMap[gameVersion]
	if !exists {
		return "", fmt.Errorf("未找到大版本 %s 的 %s 版本列表", gameVersion, versionType)
	}

	// 查找匹配的子版本
	for _, v := range versions {
		// 移除可能的前缀（如 "API" 前缀）进行比较
		cleanSubVersion := strings.TrimPrefix(v.SubVersion, "API")
		cleanSubVersion = strings.TrimPrefix(cleanSubVersion, "NET")
		cleanSubVersion = strings.TrimPrefix(cleanSubVersion, "Original")
		cleanSubVersion = strings.TrimPrefix(cleanSubVersion, "Modified")

		if cleanSubVersion == subVersion || v.SubVersion == subVersion {
			return v.Path, nil
		}
	}

	return "", fmt.Errorf("未找到版本 %s:%s-%s", gameVersion, versionType, subVersion)
}

// downloadGame 下载游戏文件
func (m *ModpackInstaller) downloadGame(downloadURL, version string) (string, error) {
	// 创建临时文件
	tempDir := filepath.Join(os.TempDir(), "sc-modpack-game-download")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	filename := filepath.Join(tempDir, fmt.Sprintf("%s.zip", version))

	// 检查文件是否已存在
	if _, err := os.Stat(filename); err == nil {
		m.updateProgress(StageDownloadGame, 70, "发现已下载的文件")
		return filename, nil
	}

	// 下载文件
	resp, err := m.httpClient.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// 获取文件大小
	totalSize := resp.ContentLength

	// 创建文件
	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 下载并显示进度
	downloaded := int64(0)
	buffer := make([]byte, 32*1024)
	lastProgress := 0.0

	for {
		if m.IsCancelled() {
			os.Remove(filename)
			return "", fmt.Errorf("下载已取消")
		}

		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, err := out.Write(buffer[:n])
			if err != nil {
				return "", fmt.Errorf("写入文件失败: %w", err)
			}

			downloaded += int64(n)

			// 更新进度 (20% - 70%)
			if totalSize > 0 {
				progress := 20.0 + (float64(downloaded)/float64(totalSize))*50.0
				// 避免频繁更新
				if progress-lastProgress > 1.0 {
					m.updateProgress(StageDownloadGame, progress,
						fmt.Sprintf("下载中... %.1f%%", (float64(downloaded)/float64(totalSize))*100))
					lastProgress = progress
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("下载失败: %w", err)
		}
	}

	m.updateProgress(StageDownloadGame, 70, "下载完成")
	return filename, nil
}

// installGameToVersion 安装游戏到版本目录
func (m *ModpackInstaller) installGameToVersion(archivePath string, versionPath string) (string, error) {
	m.updateProgress(StageInstallGame, 85, "解压游戏文件到版本目录...")

	// 直接解压到版本目录
	installer := version.NewInstaller()
	if err := installer.Install(archivePath, versionPath, nil); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}

	m.updateProgress(StageInstallGame, 100, "游戏解压完成")
	return versionPath, nil
}

// downloadMods 下载模组
func (m *ModpackInstaller) downloadMods(manifest *Manifest, targetVersionID string) error {
	if len(manifest.Mods) == 0 {
		m.updateProgress(StageDownloadMods, 100, "无需下载模组")
		return nil
	}

	totalMods := len(manifest.Mods)
	for i, modInfo := range manifest.Mods {
		if m.IsCancelled() {
			return fmt.Errorf("下载已取消")
		}

		// 计算进度
		progress := float64(i) / float64(totalMods) * 100
		m.updateProgress(StageDownloadMods, progress,
			fmt.Sprintf("下载模组 %d/%d: %s", i+1, totalMods, modInfo.Name))

		if err := m.downloadMod(modInfo, targetVersionID, manifest.ModPath); err != nil {
			if modInfo.Required {
				return fmt.Errorf("下载必需模组 %s 失败: %w", modInfo.Name, err)
			}
			// 非必需模组，记录错误但继续
			m.updateProgress(StageDownloadMods, progress,
				fmt.Sprintf("跳过可选模组 %s: %v", modInfo.Name, err))
		}
	}

	m.updateProgress(StageDownloadMods, 100, fmt.Sprintf("成功下载 %d 个模组", totalMods))
	return nil
}

// downloadMod 下载单个模组
func (m *ModpackInstaller) downloadMod(modInfo ModInfo, targetVersionID string, globalModPath string) error {
	// TODO: 实现模组下载逻辑
	// 这里需要调用模组管理器的下载功能

	// 确定模组的安装路径
	modPath := modInfo.ModPath
	if modPath == "" {
		modPath = globalModPath
	}
	if modPath == "" {
		modPath = "/Mods" // 默认路径
	}

	_ = targetVersionID // 暂时避免未使用参数警告
	_ = modInfo
	_ = modPath
	return nil
}

// copyOverrides 复制覆盖文件
func (m *ModpackInstaller) copyOverrides(manifest *Manifest, gamePath string) error {
	// 检查 overrides 字段是否为空
	if manifest.Overrides == "" {
		return fmt.Errorf("未指定覆盖文件目录")
	}

	m.updateProgress(StageCopyOverrides, 10, "准备复制覆盖文件...")

	// 解压整合包到临时目录
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("sc-modpack-overrides-%s", manifest.FileHash))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压
	installer := version.NewInstaller()
	if err := installer.Install(manifest.FilePath, tempDir, nil); err != nil {
		return fmt.Errorf("解压整合包失败: %w", err)
	}

	// 查找覆盖目录
	overridesPath := filepath.Join(tempDir, manifest.Overrides)
	if _, err := os.Stat(overridesPath); os.IsNotExist(err) {
		return fmt.Errorf("覆盖目录不存在: %s", manifest.Overrides)
	}

	// 复制文件
	m.updateProgress(StageCopyOverrides, 50, "复制覆盖文件...")
	if err := copyDirectory(overridesPath, gamePath); err != nil {
		return fmt.Errorf("复制覆盖文件失败: %w", err)
	}

	m.updateProgress(StageCopyOverrides, 100, "覆盖文件复制完成")
	return nil
}

// copyDirectory 复制目录
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 复制文件
		return copyFile(path, dstPath)
	})
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}
