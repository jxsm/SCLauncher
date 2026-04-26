package modpack

import (
	"archive/zip"
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

	fmt.Printf("=== 开始阶段4: 复制覆盖文件 ===\n")
	fmt.Printf("manifest.Overrides 值: '%s'\n", manifest.Overrides)
	fmt.Printf("gamePath 值: '%s'\n", gamePath)

	m.updateProgress(StageCopyOverrides, 0, "复制覆盖文件...")
	if manifest.Overrides != "" {
		fmt.Printf("overrides 字段不为空，开始执行覆盖逻辑...\n")
		if err := m.copyOverrides(manifest, gamePath); err != nil {
			// 覆盖文件复制失败不应该阻止整个安装过程
			// 只记录错误，不中断安装
			fmt.Printf("覆盖文件复制失败: %v\n", err)
			m.updateProgress(StageCopyOverrides, 50, fmt.Sprintf("覆盖文件复制跳过: %v", err))
		} else {
			fmt.Printf("覆盖文件复制成功！\n")
			m.updateProgress(StageCopyOverrides, 100, "覆盖文件复制完成")
		}
	} else {
		fmt.Printf("overrides 字段为空，跳过覆盖逻辑\n")
		m.updateProgress(StageCopyOverrides, 100, "无覆盖文件，跳过")
	}
	fmt.Printf("=== 阶段4结束 ===\n")

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

// downloadMods 下载模组（使用并发控制，最多2个并发）
func (m *ModpackInstaller) downloadMods(manifest *Manifest, targetVersionID string) error {
	if len(manifest.Mods) == 0 {
		m.updateProgress(StageDownloadMods, 100, "无需下载模组")
		return nil
	}

	// 创建semaphore限制并发数为2
	sem := make(chan struct{}, 2)
	var wg sync.WaitGroup

	// 用于收集错误
	errorChan := make(chan error, len(manifest.Mods))
	successCount := 0
	failedMods := make([]string, 0)
	var successMutex sync.Mutex

	totalMods := len(manifest.Mods)
	completedCount := 0

	// 创建SCBBS API客户端
	scbbsClient := NewSCBBSApiClient()

	for i, modInfo := range manifest.Mods {
		// 检查是否已取消
		if m.IsCancelled() {
			return fmt.Errorf("下载已取消")
		}

		wg.Add(1)
		go func(index int, mod ModInfo) {
			defer wg.Done()

			// 获取semaphore槽位
			sem <- struct{}{}
			defer func() { <-sem }()

			// 下载模组
			err := m.downloadMod(mod, targetVersionID, manifest.ModPath, scbbsClient)

			// 线程安全地更新结果
			successMutex.Lock()
			completedCount++
			if err != nil {
				if mod.Required {
					errorChan <- fmt.Errorf("下载必需模组 %s 失败: %w", mod.Name, err)
				} else {
					// 非必需模组，记录失败但继续
					failedMods = append(failedMods, fmt.Sprintf("%s: %v", mod.Name, err))
				}
			} else {
				successCount++
			}

			// 更新进度
			progress := float64(completedCount) / float64(totalMods) * 100
			currentMsg := fmt.Sprintf("下载模组 %d/%d: %s", completedCount, totalMods, mod.Name)
			m.updateProgress(StageDownloadMods, progress, currentMsg)
			successMutex.Unlock()
		}(i, modInfo)
	}

	// 等待所有下载完成
	wg.Wait()
	close(errorChan)

	// 检查是否有必需模组下载失败
	for err := range errorChan {
		return err
	}

	// 显示最终结果
	resultMsg := fmt.Sprintf("成功下载 %d 个模组", successCount)
	if len(failedMods) > 0 {
		resultMsg += fmt.Sprintf("，跳过 %d 个可选模组", len(failedMods))
	}
	m.updateProgress(StageDownloadMods, 100, resultMsg)

	return nil
}

// downloadMod 下载单个模组
func (m *ModpackInstaller) downloadMod(modInfo ModInfo, targetVersionID string, globalModPath string, scbbsClient *SCBBSApiClient) error {
	// 确定模组的安装路径
	modPath := modInfo.ModPath
	if modPath == "" {
		modPath = globalModPath
	}
	if modPath == "" {
		modPath = "/Mods" // 默认路径
	}

	// 确定下载URL和文件名
	var downloadURL string
	var fileName string

	// 优先使用自定义下载链接
	if modInfo.Path != "" && m.isValidURL(modInfo.Path) {
		downloadURL = modInfo.Path
		// 从URL中提取文件名，如果没有则使用模组名称
		if idx := strings.LastIndex(downloadURL, "/"); idx != -1 {
			fileName = downloadURL[idx+1:]
		} else {
			fileName = modInfo.Name + ".scmod"
		}
	} else {
		// 从SCBBS获取下载链接
		postDetail, err := scbbsClient.GetModVersions(modInfo.ProjectID)
		if err != nil {
			return fmt.Errorf("获取模组版本信息失败: %w", err)
		}

		// 查找匹配的版本
		_, modFile, err := scbbsClient.FindMatchingVersion(postDetail.Data.PostVersions, modInfo.Version)
		if err != nil {
			return fmt.Errorf("查找匹配版本失败: %w", err)
		}

		downloadURL = modFile.URL
		fileName = modFile.Filename
	}

	// 下载模组文件
	tempFilePath, err := m.downloadModFile(downloadURL, fileName, modInfo.Name)
	if err != nil {
		return fmt.Errorf("下载模组文件失败: %w", err)
	}
	defer os.Remove(tempFilePath) // 下载完成后删除临时文件

	// 构建目标路径
	versionPath := m.paths.GetVersionPath(targetVersionID)
	destPath := filepath.Join(versionPath, modPath, fileName)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 移动文件到目标位置
	if err := os.Rename(tempFilePath, destPath); err != nil {
		// 如果跨设备移动失败，尝试复制
		if err := copyFile(tempFilePath, destPath); err != nil {
			return fmt.Errorf("移动模组文件失败: %w", err)
		}
	}

	return nil
}

// downloadModFile 下载模组文件到临时目录
func (m *ModpackInstaller) downloadModFile(downloadURL, fileName, _ string) (string, error) {
	// 创建临时文件（使用fileName作为临时文件名的一部分，方便调试）
	tempFile, err := os.CreateTemp("", fmt.Sprintf("scmod-%s-*.tmp", fileName))
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tempFile.Close()

	tempFilePath := tempFile.Name()

	// 发起HTTP请求
	resp, err := m.httpClient.Get(downloadURL)
	if err != nil {
		os.Remove(tempFilePath)
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tempFilePath)
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// 下载文件
	buffer := make([]byte, 32*1024)
	for {
		if m.IsCancelled() {
			os.Remove(tempFilePath)
			return "", fmt.Errorf("下载已取消")
		}

		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := tempFile.Write(buffer[:n]); err != nil {
				os.Remove(tempFilePath)
				return "", fmt.Errorf("写入文件失败: %w", err)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tempFilePath)
			return "", fmt.Errorf("下载失败: %w", err)
		}
	}

	return tempFilePath, nil
}

// copyOverrides 复制覆盖文件（直接从压缩包中提取 overrides 目录）
func (m *ModpackInstaller) copyOverrides(manifest *Manifest, gamePath string) error {
	// 检查 overrides 字段是否为空
	if manifest.Overrides == "" {
		return fmt.Errorf("未指定覆盖文件目录")
	}

	m.updateProgress(StageCopyOverrides, 10, "准备复制覆盖文件...")

	// 打开整合包压缩包（.scmodpack 实际上是 zip 文件）
	zipReader, err := zip.OpenReader(manifest.FilePath)
	if err != nil {
		return fmt.Errorf("打开整合包失败: %w", err)
	}
	defer zipReader.Close()

	// 构建 overrides 目录前缀（zip 文件中路径总是使用 /）
	overridesPrefix := strings.TrimSuffix(manifest.Overrides, "/") + "/"

	// 添加调试信息：列出所有文件
	fmt.Printf("=== 覆盖文件调试信息 ===\n")
	fmt.Printf("Overrides 字段: %s\n", manifest.Overrides)
	fmt.Printf("查找前缀: %s\n", overridesPrefix)
	fmt.Printf("游戏根目录: %s\n", gamePath)
	fmt.Printf("压缩包中的所有文件:\n")

	// 统计需要复制的文件数量，用于进度显示
	totalFiles := 0
	var filesToCopy []*zip.File

	// 第一遍扫描：收集所有需要复制的文件
	for _, file := range zipReader.File {
		// zip 文件中的路径总是使用正斜杠
		filePath := file.Name
		fmt.Printf("  - %s (目录: %v)\n", filePath, file.FileInfo().IsDir())

		// 检查文件是否在 overrides 目录中
		if strings.HasPrefix(filePath, overridesPrefix) {
			filesToCopy = append(filesToCopy, file)
			if !file.FileInfo().IsDir() {
				totalFiles++
			}
			fmt.Printf("    ✓ 匹配！将复制到: %s\n", strings.TrimPrefix(filePath, overridesPrefix))
		}
	}

	fmt.Printf("匹配的文件总数: %d\n", totalFiles)
	fmt.Printf("=== 调试信息结束 ===\n")

	if totalFiles == 0 {
		return fmt.Errorf("覆盖目录中没有文件: %s (查找前缀: %s)", manifest.Overrides, overridesPrefix)
	}

	m.updateProgress(StageCopyOverrides, 20, fmt.Sprintf("发现 %d 个文件需要覆盖", totalFiles))

	// 第二遍扫描：复制文件
	copiedCount := 0
	for _, file := range filesToCopy {
		if m.IsCancelled() {
			return fmt.Errorf("复制已取消")
		}

		// 计算相对路径（去掉 overrides 前缀）
		relPath := strings.TrimPrefix(file.Name, overridesPrefix)

		// 构建目标路径
		destPath := filepath.Join(gamePath, relPath)

		fmt.Printf("复制: %s -> %s\n", file.Name, destPath)

		if file.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(destPath, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("创建目录失败 %s: %w", relPath, err)
			}
		} else {
			// 复制文件
			if err := extractFileFromFile(file, destPath); err != nil {
				return fmt.Errorf("复制文件失败 %s: %w", relPath, err)
			}
			copiedCount++

			// 更新进度
			progress := 20.0 + (float64(copiedCount)/float64(totalFiles))*70.0
			if copiedCount%10 == 0 || copiedCount == totalFiles {
				m.updateProgress(StageCopyOverrides, progress,
					fmt.Sprintf("复制覆盖文件 %d/%d", copiedCount, totalFiles))
			}
		}
	}

	m.updateProgress(StageCopyOverrides, 100, fmt.Sprintf("覆盖文件复制完成，共 %d 个文件", copiedCount))
	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// extractFileFromFile 从 zip 文件中提取单个文件
func extractFileFromFile(file *zip.File, destPath string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// 打开 zip 中的文件
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制文件内容
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	return nil
}
