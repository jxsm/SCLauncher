package scbbs

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SCLauncher/backend/config"
	"SCLauncher/backend/mod"
	"SCLauncher/backend/scbbsapi"
	"SCLauncher/backend/utils"
)

// 安装阶段（走既有 modpack:install:progress 事件通道）
const (
	StagePrepare      = "prepare"       // 准备
	StageProcessItems = "process_items" // 逐项下载/安装
	StageComplete     = "complete"      // 完成
)

// 冲突处理决策（UI 层回传的 action 值）
const (
	ConflictOverwrite    = "overwrite"    // 覆盖
	ConflictSkip         = "skip"         // 跳过
	ConflictOverwriteAll = "overwriteAll" // 全部覆盖（后续不再询问）
	ConflictSkipAll      = "skipAll"      // 全部跳过（后续不再询问）
	ConflictCancel       = "cancel"       // 取消安装
)

// errCancelled 安装被用户取消
var errCancelled = errors.New("安装已取消")

// InstallProgress 安装进度回调（与 modpack.InstallProgress 同签名，
// 复用整合包安装的前端进度弹窗）
type InstallProgress func(stage string, progress float64, message string)

// ConflictAskFn 冲突询问回调：目标文件已存在时调用，由 UI 层实现
// （弹窗询问用户）。fileName/itemName 用于展示；返回错误表示取消安装。
type ConflictAskFn func(fileName, itemName string) (action string, err error)

// InstallSummary 安装结果汇总
type InstallSummary struct {
	Total     int      `json:"total"`              // 参与安装的子项总数
	Succeeded int      `json:"succeeded"`          // 成功
	Skipped   int      `json:"skipped"`            // 冲突跳过
	Failed    int      `json:"failed"`             // 失败
	Failures  []string `json:"failures,omitempty"` // 失败明细
}

// Installer scbbs 整合包安装器
type Installer struct {
	cfg         *config.Config
	paths       *config.Paths
	modMgr      *mod.Manager
	httpClient  *http.Client
	apiClient   *scbbsapi.SCBBSApiClient
	progressCb  InstallProgress
	conflictAsk ConflictAskFn

	mu           sync.Mutex
	cancelChan   chan struct{}
	cancelFunc   context.CancelFunc
	skipAll      bool
	overwriteAll bool
}

// NewInstaller 创建 scbbs 整合包安装器
func NewInstaller(cfg *config.Config, modMgr *mod.Manager) *Installer {
	return &Installer{
		cfg:        cfg,
		paths:      config.NewPaths(cfg),
		modMgr:     modMgr,
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		apiClient:  scbbsapi.NewSCBBSApiClient(),
	}
}

// SetProgressCallback 设置进度回调
func (i *Installer) SetProgressCallback(cb InstallProgress) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.progressCb = cb
}

// SetConflictAskFn 设置冲突询问回调
func (i *Installer) SetConflictAskFn(fn ConflictAskFn) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.conflictAsk = fn
}

// SetAPIClient 替换社区 API 客户端（测试注入用）
func (i *Installer) SetAPIClient(client *scbbsapi.SCBBSApiClient) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.apiClient = client
}

// Cancel 取消安装（同时中断进行中的下载并解锁等待中的冲突询问）
func (i *Installer) Cancel() {
	i.mu.Lock()
	defer i.mu.Unlock()
	select {
	case <-i.cancelChan:
		// 已关闭
	default:
		close(i.cancelChan)
	}
	if i.cancelFunc != nil {
		i.cancelFunc()
	}
}

// IsCancelled 是否已取消
func (i *Installer) IsCancelled() bool {
	i.mu.Lock()
	cancelChan := i.cancelChan
	i.mu.Unlock()
	if cancelChan == nil {
		return false
	}
	select {
	case <-cancelChan:
		return true
	default:
		return false
	}
}

// emit 发送进度
func (i *Installer) emit(stage string, progress float64, message string) {
	i.mu.Lock()
	cb := i.progressCb
	i.mu.Unlock()
	if cb != nil {
		cb(stage, progress, message)
	}
}

// askConflict 目标文件已存在时询问用户决策。
// 返回 (skip, err)：skip=true 跳过该项；err 非 nil 表示取消整个安装。
// 已选择过"全部覆盖/全部跳过"后不再询问。
func (i *Installer) askConflict(fileName, itemName string) (bool, error) {
	i.mu.Lock()
	skipAll, overwriteAll, conflictAsk := i.skipAll, i.overwriteAll, i.conflictAsk
	i.mu.Unlock()

	if skipAll {
		return true, nil
	}
	if overwriteAll {
		return false, nil
	}
	if conflictAsk == nil {
		// 无询问渠道时默认覆盖（与 v0.1 安装器行为一致）
		return false, nil
	}

	action, err := conflictAsk(fileName, itemName)
	if err != nil {
		return false, err
	}
	switch action {
	case ConflictSkip:
		return true, nil
	case ConflictSkipAll:
		i.mu.Lock()
		i.skipAll = true
		i.mu.Unlock()
		return true, nil
	case ConflictOverwriteAll:
		i.mu.Lock()
		i.overwriteAll = true
		i.mu.Unlock()
		return false, nil
	case ConflictCancel:
		return false, errCancelled
	default: // ConflictOverwrite 及未知值一律覆盖
		return false, nil
	}
}

// Install 安装整合包到指定版本（逐项处理，单项失败不中断整体）
func (i *Installer) Install(result *ParseResult, versionID string) (*InstallSummary, error) {
	if result == nil || result.Manifest == nil {
		return nil, fmt.Errorf("无效的解析结果")
	}

	// 重置取消状态与冲突记忆
	i.mu.Lock()
	i.cancelChan = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	i.cancelFunc = cancel
	i.skipAll, i.overwriteAll = false, false
	i.mu.Unlock()
	defer cancel()

	items := result.Manifest.EnabledItems()
	if len(items) == 0 {
		return nil, fmt.Errorf("整合包没有启用的子项")
	}

	versionPath := i.paths.GetVersionPath(versionID)
	if info, err := os.Stat(versionPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("目标版本目录不存在: %s", versionID)
	}

	i.emit(StagePrepare, 0, "正在准备安装...")

	// ZIP 载体：打开并建立内嵌文件索引
	var embedded map[string]*zip.File
	if result.IsZipCarrier {
		reader, err := zip.OpenReader(result.FilePath)
		if err != nil {
			return nil, fmt.Errorf("打开整合包文件失败: %w", err)
		}
		defer reader.Close()
		embedded = buildEmbeddedIndex(reader.File)
	}

	tempDir, err := os.MkdirTemp("", "sc-scbbs-install-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	i.emit(StagePrepare, 100, "准备完成")

	summary := &InstallSummary{Total: len(items)}
	for idx, item := range items {
		if i.IsCancelled() {
			return nil, errCancelled
		}
		i.emit(StageProcessItems, float64(idx)/float64(len(items))*100,
			fmt.Sprintf("正在处理子项 %d/%d：%s", idx+1, len(items), item.DisplayName()))

		skipped, err := i.processItem(ctx, result, item, versionID, versionPath, embedded, tempDir, idx)
		switch {
		case errors.Is(err, errCancelled):
			return nil, errCancelled
		case err != nil:
			summary.Failed++
			summary.Failures = append(summary.Failures, fmt.Sprintf("%s: %v", item.DisplayName(), err))
		case skipped:
			summary.Skipped++
		default:
			summary.Succeeded++
		}
	}

	if summary.Succeeded == 0 && summary.Skipped == 0 {
		return summary, fmt.Errorf("所有子项均安装失败（共 %d 项）", summary.Failed)
	}

	i.emit(StageComplete, 100,
		fmt.Sprintf("安装完成：成功 %d，跳过 %d，失败 %d", summary.Succeeded, summary.Skipped, summary.Failed))
	return summary, nil
}

// processItem 处理单个子项：取数据 → 定文件名 → 校验 → 落地。
// 返回 (是否跳过, 错误)。
func (i *Installer) processItem(ctx context.Context, result *ParseResult, item *Item,
	versionID, versionPath string, embedded map[string]*zip.File, tempDir string, idx int) (bool, error) {

	itemType := item.ResolveType()

	// 1. 解析来源并把数据取到临时文件
	tmpPath, fileName, communityInfo, err := i.fetchItem(ctx, result, item, embedded, tempDir, idx)
	if err != nil {
		return false, err
	}
	defer os.Remove(tmpPath)

	// 2. 补全扩展名（模组扩展名按联机/单机决定）
	fileName = ensureExtension(fileName, itemType, i.isOnlineVersion(versionID))
	if strings.TrimSpace(fileName) == "" {
		return false, fmt.Errorf("无法确定落地文件名")
	}

	// 3. 模组子项：与 modinfo.json 的包名/版本比对（防链接调包，
	//    对齐手机版 _validateModItem / ScbbsPlus ValidateModPackItem）
	if itemType == ItemTypeMod {
		if err := i.validateModItem(tmpPath, item, communityInfo); err != nil {
			return false, err
		}
	}

	// 4. 按类型落地
	switch itemType {
	case ItemTypeWorld:
		return i.installWorldItem(tmpPath, fileName, item, versionID)
	case ItemTypeData:
		return i.installDataItem(tmpPath, fileName, item, versionPath)
	default:
		return i.installStandardItem(tmpPath, fileName, item, itemType, versionID)
	}
}

// fetchItem 解析子项来源并把数据取到临时文件。
// 来源优先级（对齐手机版 _processItem）：http(s) 直链 > ZIP 包内路径 > FileId 社区反查。
// 返回 (临时文件路径, 推断的文件名, 社区文件元数据或 nil, 错误)。
func (i *Installer) fetchItem(ctx context.Context, result *ParseResult, item *Item,
	embedded map[string]*zip.File, tempDir string, idx int) (string, string, *scbbsapi.UploadFileInfo, error) {

	url := strings.TrimSpace(item.URL)
	fileID := strings.TrimSpace(item.FileID)
	tmpPath := filepath.Join(tempDir, fmt.Sprintf("item-%d.bin", idx))
	var communityInfo *scbbsapi.UploadFileInfo

	switch {
	case url != "" && isHTTPURL(url):
		// http(s) 直链下载
		if err := i.downloadToFile(ctx, url, tmpPath); err != nil {
			return "", "", nil, fmt.Errorf("下载失败: %w", err)
		}

	case url != "" && result.IsZipCarrier:
		// ZIP 包内相对路径
		key := strings.ToLower(NormalizeArchivePath(url))
		entry, ok := embedded[key]
		if !ok {
			// 包内缺失 → FileId 兜底
			if fileID == "" {
				return "", "", nil, fmt.Errorf("整合包内缺少文件 %q", url)
			}
			info, err := i.apiClient.FetchUploadInfo(fileID)
			if err != nil {
				return "", "", nil, fmt.Errorf("查询社区文件 %s 失败: %w", fileID, err)
			}
			communityInfo = info
			if err := i.downloadToFile(ctx, info.URL, tmpPath); err != nil {
				return "", "", nil, fmt.Errorf("下载失败: %w", err)
			}
		} else {
			if err := extractZipEntryToFile(entry, tmpPath); err != nil {
				return "", "", nil, fmt.Errorf("提取包内文件 %q 失败: %w", url, err)
			}
		}

	case fileID != "":
		// 社区文件 ID 反查下载链接
		info, err := i.apiClient.FetchUploadInfo(fileID)
		if err != nil {
			return "", "", nil, fmt.Errorf("查询社区文件 %s 失败: %w", fileID, err)
		}
		communityInfo = info
		if err := i.downloadToFile(ctx, info.URL, tmpPath); err != nil {
			return "", "", nil, fmt.Errorf("下载失败: %w", err)
		}

	default:
		return "", "", nil, fmt.Errorf("子项既没有 Url 也没有 FileId")
	}

	// 文件名：显式 FileName > 社区文件元数据名 > Url 尾段 > TargetPath 文件名
	fileName := item.InferFileName()
	if strings.TrimSpace(fileName) == "" && communityInfo != nil {
		fileName = strings.TrimSpace(communityInfo.Filename)
	}
	return tmpPath, fileName, communityInfo, nil
}

// downloadToFile 下载 URL 到文件（支持取消）
func (i *Installer) downloadToFile(ctx context.Context, rawURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SCLauncher/1.0")

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回状态码 %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// validateModItem 模组包名/版本校验：打开 .scmod 读 modinfo.json，
// 与清单（或社区元数据）声明的期望值比对，大小写不敏感。
func (i *Installer) validateModItem(modPath string, item *Item, communityInfo *scbbsapi.UploadFileInfo) error {
	expectedPkg := strings.TrimSpace(item.PackageName)
	expectedVer := strings.TrimSpace(item.Version)
	if communityInfo != nil {
		if expectedPkg == "" {
			expectedPkg = strings.TrimSpace(communityInfo.PackageName)
		}
		if expectedVer == "" {
			expectedVer = strings.TrimSpace(communityInfo.PackageVersion)
		}
	}
	if expectedPkg == "" && expectedVer == "" {
		return nil
	}

	info := mod.ParseModInfoFromModFile(modPath)
	if info == nil {
		return fmt.Errorf("不是有效的模组文件（无法解析 modinfo.json）")
	}
	if expectedPkg != "" && !strings.EqualFold(strings.TrimSpace(info.PackageName), expectedPkg) {
		actual := info.PackageName
		if actual == "" {
			actual = "(无)"
		}
		return fmt.Errorf("模组包名不匹配：期望 %s，实际 %s", expectedPkg, actual)
	}
	if expectedVer != "" && !strings.EqualFold(strings.TrimSpace(info.Version), expectedVer) {
		actual := info.Version
		if actual == "" {
			actual = "(无)"
		}
		return fmt.Errorf("模组版本不匹配：期望 %s，实际 %s", expectedVer, actual)
	}
	return nil
}

// itemTypeDirName 标准资源类型 → 游戏数据目录名
var itemTypeDirName = map[ItemType]string{
	ItemTypeTexture:   "BlocksTextures",
	ItemTypeSkin:      "CharacterSkins",
	ItemTypeFurniture: "FurniturePacks",
}

// installStandardItem 安装标准资源子项（模组/材质/皮肤/家具包，单文件落地）
func (i *Installer) installStandardItem(srcPath, fileName string, item *Item, itemType ItemType, versionID string) (bool, error) {
	var targetDir string
	if itemType == ItemTypeMod {
		// GetModPath 已处理导入版本与联机版（NetMods）
		targetDir = i.modMgr.GetModPath(versionID)
	} else {
		targetDir = i.resolveGameDir(versionID, itemTypeDirName[itemType])
	}

	safeName := utils.SanitizeFilename(fileName)
	if safeName == "" {
		return false, fmt.Errorf("文件名 %q 无效", fileName)
	}
	destPath := filepath.Join(targetDir, safeName)

	if _, err := os.Stat(destPath); err == nil {
		skip, err := i.askConflict(safeName, item.DisplayName())
		if err != nil {
			return false, err
		}
		if skip {
			return true, nil
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return false, err
	}
	if err := copyFile(srcPath, destPath); err != nil {
		return false, err
	}
	return false, nil
}

// installWorldItem 安装存档子项：把 .scworld（zip 归档流）解压为
// Worlds 目录下以文件名（去最后一段扩展名）命名的世界文件夹。
func (i *Installer) installWorldItem(srcPath, fileName string, item *Item, versionID string) (bool, error) {
	worldName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	safeName := utils.SanitizeFilename(worldName)
	if safeName == "" {
		return false, fmt.Errorf("无法确定世界文件夹名")
	}
	worldsDir := i.resolveGameDir(versionID, "Worlds")
	targetDir := filepath.Join(worldsDir, safeName)

	reader, err := zip.OpenReader(srcPath)
	if err != nil {
		return false, fmt.Errorf("存档文件不是有效的 .scworld: %w", err)
	}
	defer reader.Close()

	// 世界文件夹已存在：冲突询问（覆盖 = 删除重建）
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		skip, err := i.askConflict(safeName, item.DisplayName())
		if err != nil {
			return false, err
		}
		if skip {
			return true, nil
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return false, err
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return false, err
	}
	for _, f := range reader.File {
		name := utils.DecodeArchiveEntryName(f.Name)
		if name == ".DS_Store" || strings.HasPrefix(name, "__MACOSX") {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(filepath.Join(targetDir, filepath.FromSlash(name)), 0755)
			continue
		}
		// 防路径穿越（zip slip）：不安全条目直接跳过
		destPath, err := utils.ValidateZipEntry(targetDir, name)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return false, err
		}
		if err := extractZipEntryToFile(f, destPath); err != nil {
			return false, err
		}
	}
	return false, nil
}

// protectedDataRoots 自定义数据文件首段黑名单
// （对齐手机版 _protectedDataRootNames，桌面版额外保护 NetMods）
var protectedDataRoots = map[string]struct{}{
	"project.xml":    {},
	"saves":          {},
	"mods":           {},
	"netmods":        {},
	"settings":       {},
	"modpacks":       {},
	"worlds":         {},
	"texturepacks":   {},
	"characterskins": {},
	"furniturepacks": {},
}

// installDataItem 安装自定义数据文件（净化后写入版本目录任意非保护路径）
func (i *Installer) installDataItem(srcPath, fileName string, item *Item, versionPath string) (bool, error) {
	relPath := resolveDataRelPath(item, fileName)
	if relPath == "" {
		return false, fmt.Errorf("无法确定目标路径")
	}
	sanitized, err := sanitizeDataRelPath(relPath)
	if err != nil {
		return false, err
	}
	destPath, err := utils.ValidatePath(versionPath, filepath.Join(versionPath, filepath.FromSlash(sanitized)))
	if err != nil {
		return false, fmt.Errorf("目标路径不安全: %w", err)
	}

	if _, err := os.Stat(destPath); err == nil {
		skip, err := i.askConflict(filepath.Base(destPath), item.DisplayName())
		if err != nil {
			return false, err
		}
		if skip {
			return true, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return false, err
	}
	if err := copyFile(srcPath, destPath); err != nil {
		return false, err
	}
	return false, nil
}

// resolveDataRelPath 解析自定义文件的目标相对路径：
// TargetPath 优先；否则 TargetDirectory 以 "/" 结尾视为目录（拼接源文件名），
// 不以 "/" 结尾则本身就是完整文件路径（对齐手机版 _resolveDataFileTargetPath）。
func resolveDataRelPath(item *Item, fileName string) string {
	if p := strings.TrimSpace(item.TargetPath); p != "" {
		return p
	}
	dir := strings.TrimSpace(item.TargetDirectory)
	if dir == "" {
		return ""
	}
	normalized := strings.ReplaceAll(dir, "\\", "/")
	if strings.HasSuffix(normalized, "/") {
		if strings.TrimSpace(fileName) == "" {
			return ""
		}
		return normalized + fileName
	}
	return normalized
}

// sanitizeDataRelPath 净化自定义文件相对路径：去盘符、剔除 ".."/"."/空段、
// 非法字符替换为 "_"、剥离段尾点与空格；首段命中保护目录黑名单则报错。
func sanitizeDataRelPath(relPath string) (string, error) {
	p := strings.ReplaceAll(strings.TrimSpace(relPath), "\\", "/")
	// 去掉 Windows 盘符前缀（如 "C:"）
	if len(p) >= 2 && p[1] == ':' {
		p = p[2:]
	}

	var segments []string
	for _, raw := range strings.Split(p, "/") {
		segment := strings.TrimSpace(raw)
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		var b strings.Builder
		for _, r := range segment {
			if strings.ContainsRune(`\/:*?"<>|`, r) || r < 32 || r == 127 {
				b.WriteRune('_')
			} else {
				b.WriteRune(r)
			}
		}
		segment = strings.Trim(b.String(), " .")
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return "", fmt.Errorf("目标路径 %q 无效", relPath)
	}
	if _, blocked := protectedDataRoots[strings.ToLower(segments[0])]; blocked {
		return "", fmt.Errorf("不允许写入受保护的目录: %s", segments[0])
	}
	return strings.Join(segments, "/"), nil
}

// resolveGameDir 解析版本的数据目录（Worlds/BlocksTextures/CharacterSkins/
// FurniturePacks 等）：依次探测 版本目录/<dir>、版本目录/doc/<dir>、导入版本
// 原路径同两条；全部不存在时：doc 目录存在则默认建在 doc 下，否则建在版本目录根下。
func (i *Installer) resolveGameDir(versionID, dirName string) string {
	versionPath := i.paths.GetVersionPath(versionID)
	candidates := []string{
		filepath.Join(versionPath, dirName),
		filepath.Join(versionPath, "doc", dirName),
	}
	if originalPath, err := mod.GetImportedVersionOriginalPath(versionPath); err == nil && originalPath != "" {
		candidates = append(candidates,
			filepath.Join(originalPath, dirName),
			filepath.Join(originalPath, "doc", dirName),
		)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if info, err := os.Stat(filepath.Join(versionPath, "doc")); err == nil && info.IsDir() {
		return filepath.Join(versionPath, "doc", dirName)
	}
	return filepath.Join(versionPath, dirName)
}

// isOnlineVersion 判断目标版本是否联机版（存在 NetMods 目录）
func (i *Installer) isOnlineVersion(versionID string) bool {
	versionPath := i.paths.GetVersionPath(versionID)
	candidates := []string{filepath.Join(versionPath, "NetMods")}
	if originalPath, err := mod.GetImportedVersionOriginalPath(versionPath); err == nil && originalPath != "" {
		candidates = append(candidates, filepath.Join(originalPath, "NetMods"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// ensureExtension 文件名无扩展名时按类型补全（模组扩展名按联机/单机决定，
// 对齐手机版 _ensureExtension）
func ensureExtension(fileName string, itemType ItemType, online bool) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return ""
	}
	if hasExtension(name) {
		return name
	}
	switch itemType {
	case ItemTypeMod:
		if online {
			return name + ".netmod"
		}
		return name + ".scmod"
	case ItemTypeWorld:
		return name + ".scworld"
	case ItemTypeTexture:
		return name + ".scbtex"
	case ItemTypeSkin:
		return name + ".scskin"
	case ItemTypeFurniture:
		return name + ".scfpack"
	default:
		return name
	}
}

// hasExtension 文件名末段是否带扩展名（点号不在首位也不在末位）
func hasExtension(fileName string) bool {
	base := strings.ReplaceAll(fileName, "\\", "/")
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	dot := strings.LastIndex(base, ".")
	return dot > 0 && dot < len(base)-1
}

// copyFile 复制文件内容
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// extractZipEntryToFile 把 zip 条目内容写出到文件
func extractZipEntryToFile(entry *zip.File, destPath string) error {
	rc, err := entry.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, rc)
	return err
}
