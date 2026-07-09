package dotnet

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
)

// currentArch 返回当前进程的 GOARCH，便于测试覆盖。
var currentArch = func() string { return runtime.GOARCH }

// Status 一次「检查 + 查询」的结果。
type Status struct {
	Needed       bool   // 游戏是否依赖系统运行时
	MajorVersion int    // 所需大版本
	Installed    bool   // 本机是否已安装该大版本
	Source       string // 探测来源，同 RequiredRuntime.Source
	TFM          string // 原始 tfm
}

// Manager 编排 detect → installed → install 的整体流程。
type Manager struct {
	Runner        CommandRunner
	HTTPClient    *http.Client
	FetchReleases func(major int) ([]byte, error) // 默认 defaultFetchReleases
	Logger        func(format string, args ...any)
	SharedDir     string // 桌面运行时安装目录，默认 DefaultDesktopSharedPath()；便于测试注入
}

// NewManager 返回使用真实实现（realRunner + 默认 HTTP）的管理器。
func NewManager() *Manager {
	return &Manager{Runner: &realRunner{}}
}

// sharedDir 返回桌面运行时安装目录（测试可覆盖）。
func (m *Manager) sharedDir() string {
	if m.SharedDir != "" {
		return m.SharedDir
	}
	return DefaultDesktopSharedPath()
}

func (m *Manager) runner() CommandRunner {
	if m.Runner != nil {
		return m.Runner
	}
	return &realRunner{}
}

// httpGet 返回供 Installer 使用的下载函数。
func (m *Manager) httpGet() func(string) (*http.Response, error) {
	if m.HTTPClient != nil {
		return m.HTTPClient.Get
	}
	return http.DefaultClient.Get
}

// defaultFetchReleases 从微软官方抓取某大版本的 releases.json。
func (m *Manager) defaultFetchReleases(major int) ([]byte, error) {
	client := m.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(ReleasesURL(major))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("releases.json 状态码 %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Status 检查游戏目录所需运行时，并比对本机是否已安装。
func (m *Manager) Status(gameDir string) (Status, error) {
	req, err := DetectRequired(gameDir)
	if err != nil {
		return Status{}, err
	}
	st := Status{
		Needed:       req.Needed,
		MajorVersion: req.MajorVersion,
		Source:       req.Source,
		TFM:          req.TFM,
	}
	if !req.Needed {
		return st, nil
	}
	if m.IsInstalled(req.MajorVersion) {
		st.Installed = true
	}
	return st, nil
}

// IsInstalled 判断指定大版本是否已安装（先 dotnet CLI，再目录扫描回退）。
func (m *Manager) IsInstalled(major int) bool {
	majors, err := ListInstalledDesktopMajors(m.runner())
	if err == nil && majors[major] {
		return true
	}
	if dirMajors, err := MajorsFromInstallDir(m.sharedDir()); err == nil && dirMajors[major] {
		return true
	}
	return false
}

// Install 安装指定大版本：直接下载微软官方安装包并以 /passive 模式运行。
//
// 默认不使用 winget —— 原因：winget 没有可对接的下载进度，且会弹出 CMD 黑框，
// 体验差且让用户产生安全顾虑。下载路径能上报真实进度（progress 回调），/passive 由
// 安装程序自身的图形进度窗口承接，无控制台窗口。InstallViaWinget 仍作为公开方法保留备用。
func (m *Manager) Install(major int, progress func(downloaded, total int64)) error {
	inst := &Installer{
		Runner:  m.Runner,
		HTTPGet: m.httpGet(),
		Logger:  m.Logger,
	}

	fetch := m.FetchReleases
	if fetch == nil {
		fetch = m.defaultFetchReleases
	}
	data, err := fetch(major)
	if err != nil {
		return fmt.Errorf("获取 releases.json 失败: %w", err)
	}
	asset, err := PickLatestDesktopInstaller(data, currentArch())
	if err != nil {
		return err
	}
	return inst.DownloadAndRun(asset, progress)
}
