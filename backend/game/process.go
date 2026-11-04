package game

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"SCLauncher/backend/config"
	"SCLauncher/backend/skin"
	"SCLauncher/backend/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Status 游戏状态
type Status string

const (
	StatusStopped Status = "stopped" // 已停止
	StatusRunning Status = "running" // 运行中
	StatusCrashed Status = "crashed" // 崩溃
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID        int       `json:"pid"`        // 进程 ID
	VersionID  string    `json:"versionId"`  // 版本 ID
	StartTime  time.Time `json:"startTime"`  // 启动时间
	Executable string    `json:"executable"` // 可执行文件路径
}

// GameManager 游戏管理器
type GameManager struct {
	ctx           context.Context
	config        *config.Config
	paths         *config.Paths
	repository    *storage.Repository
	process       *exec.Cmd
	processID     int
	status        Status
	statusMu      sync.RWMutex
	processMu     sync.Mutex
	logBuffer     *bytes.Buffer // 日志缓冲区
	killed        bool          // 是否被用户主动终止
	gameWorkDir   string        // 本次启动的游戏工作目录
	gameLogOffset int64         // 启动前 Game.log 的文件大小（用于读取本次运行的日志）
}

// NewGameManager 创建游戏管理器
func NewGameManager(cfg *config.Config, repo *storage.Repository) *GameManager {
	return &GameManager{
		config:     cfg,
		paths:      config.NewPaths(cfg),
		repository: repo,
		status:     StatusStopped,
		logBuffer:  &bytes.Buffer{},
	}
}

// SetContext 设置上下文（用于发送事件）
func (g *GameManager) SetContext(ctx context.Context) {
	g.ctx = ctx
}

// Launch 启动游戏
//
// 启动策略：
//  1. 游戏目录下存在 Survivalcraft.dll（2.4+ 的 .NET 结构）时，优先使用
//     "dotnet Survivalcraft.dll" 启动：可直接捕获 .NET 运行日志，且能规避
//     直接启动 exe 偶发的 2148734214 错误。
//  2. 没有 dll，或 dotnet 方式启动失败（找不到 dotnet、进程在观察期内异常退出）时，
//     回退为直接启动 Survivalcraft.exe。
//  3. 两种方式都无法启动时，返回包含各自失败原因的错误。
func (g *GameManager) Launch(versionID string) error {
	g.processMu.Lock()
	defer g.processMu.Unlock()

	// 检查是否已有游戏在运行
	if g.isRunning() {
		return fmt.Errorf("game is already running")
	}

	// 获取版本信息
	version, err := g.repository.GetVersion(versionID)
	if err != nil {
		return fmt.Errorf("version not found: %w", err)
	}

	if !version.Installed {
		return fmt.Errorf("version is not installed")
	}

	// 查找游戏可执行文件
	exePath, err := g.findGameExecutable(versionID)
	if err != nil {
		return fmt.Errorf("game executable not found: %w", err)
	}

	// 获取工作目录
	workDir := g.getGameWorkDirectory(versionID, exePath)

	// 同步皮肤到游戏目录
	skinManager := skin.NewManager(g.config)
	if err := skinManager.SyncSkinsToGame(versionID); err != nil {
		// 皮肤同步失败不阻塞游戏启动，只记录警告
		if g.ctx != nil {
			runtime.LogWarning(g.ctx, fmt.Sprintf("皮肤同步失败: %v", err))
		}
		fmt.Printf("Warning: failed to sync skins: %v\n", err)
	}

	// 清空日志缓冲区
	g.logBuffer.Reset()
	g.killed = false
	g.gameWorkDir = workDir

	// 记录 Game.log 当前大小，用于崩溃时只读取本次运行的日志
	gameLogPath := filepath.Join(workDir, "Bugs", "Game.log")
	if info, err := os.Stat(gameLogPath); err == nil {
		g.gameLogOffset = info.Size()
	} else {
		g.gameLogOffset = 0
	}

	// 优先尝试 dotnet Survivalcraft.dll
	var dotnetErr error  // dotnet 尝试的失败原因
	var dotnetLog string // dotnet 尝试捕获到的输出（回退 exe 后崩溃时用于诊断）
	if dllPath := g.findGameDll(versionID, exePath); dllPath != "" {
		buf := &syncBuffer{}
		cmd, waitCh, err := g.startViaDotnet(dllPath, workDir, buf)
		if err != nil {
			dotnetErr = err
		} else {
			select {
			case waitErr := <-waitCh:
				if waitErr == nil {
					// 启动后立即正常退出（极短会话），按正常启动处理
					return g.finishLaunch(versionID, cmd, waitCh, buf, "", "dotnet Survivalcraft.dll")
				}
				exitCode := -1
				if exitErr, ok := waitErr.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
				dotnetErr = fmt.Errorf("进程启动后立即退出（退出码 %d）", exitCode)
				dotnetLog = buf.String()
			case <-time.After(launchGracePeriod):
				// 运行超过观察期，dotnet 方式启动成功
				return g.finishLaunch(versionID, cmd, waitCh, buf, "", "dotnet Survivalcraft.dll")
			}
		}
		if g.ctx != nil {
			runtime.LogWarning(g.ctx, fmt.Sprintf("dotnet 启动 Survivalcraft.dll 失败(%v)，回退 Survivalcraft.exe", dotnetErr))
			if dotnetLog != "" {
				runtime.LogWarning(g.ctx, fmt.Sprintf("dotnet 输出: %s", truncateLog(dotnetLog, 1000)))
			}
		}
	}

	// 直接启动 Survivalcraft.exe（无 dll 或 dotnet 方式失败时）
	cmd := exec.Command(exePath)
	cmd.Dir = workDir

	// 不要捕获输出，这对GUI程序会有影响
	// 只记录日志到文件，不重定向stdout/stderr

	// Windows: 不设置 CREATE_NO_WINDOW，让游戏可以正常显示窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false, // ⭐ 不隐藏窗口
		CreationFlags: 0,     // ⭐ 不使用 CREATE_NO_WINDOW
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		// 两种方式都启动失败：错误信息中带上各自的原因
		if dotnetErr != nil {
			return fmt.Errorf("通过 dotnet 启动 Survivalcraft.dll 失败(%v)，通过 Survivalcraft.exe 启动也失败: %w", dotnetErr, err)
		}
		return fmt.Errorf("failed to start game: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	return g.finishLaunch(versionID, cmd, waitCh, nil, dotnetLog, "Survivalcraft.exe")
}

// launchGracePeriod dotnet 启动的观察期：进程在观察期内异常退出视为启动失败，
// 回退到 Survivalcraft.exe。
const launchGracePeriod = 3 * time.Second

// createNoWindow 对应 Windows 的 CREATE_NO_WINDOW 标志（0x08000000）。
// dotnet.exe 是控制台程序，输出已通过管道捕获，无需再弹出控制台窗口。
const createNoWindow = 0x08000000

// finishLaunch 启动成功后的收尾：保存进程信息、记录数据库并启动监控协程。
// stdout 仅在 dotnet 启动方式下非空（直接捕获的 .NET 输出）；
// fallbackLog 非 nil 表示曾尝试 dotnet 启动失败后回退 exe，崩溃时日志会附带该次输出辅助定位。
func (g *GameManager) finishLaunch(versionID string, cmd *exec.Cmd, waitCh <-chan error, stdout *syncBuffer, fallbackLog, method string) error {
	// 记录日志
	if g.ctx != nil {
		runtime.LogInfo(g.ctx, fmt.Sprintf("游戏进程启动成功: PID=%d, 启动方式=%s", cmd.Process.Pid, method))
		runtime.LogInfo(g.ctx, fmt.Sprintf("工作目录: %s", g.paths.GetVersionPath(versionID)))
	}

	// 保存进程信息
	g.process = cmd
	g.processID = cmd.Process.Pid
	g.setStatus(StatusRunning)

	// 记录到数据库
	processRecord := &storage.GameProcessModel{
		ID:        generateProcessID(),
		VersionID: versionID,
		PID:       g.processID,
		StartTime: time.Now(),
	}
	if err := g.repository.CreateGameProcess(processRecord); err != nil {
		// 记录失败不影响启动
		fmt.Printf("Warning: failed to record process: %v\n", err)
	}

	// 启动监控协程（stdout 仅 dotnet 启动方式下传递）
	go g.monitorProcess(versionID, processRecord.ID, stdout, waitCh, fallbackLog)

	return nil
}

// startViaDotnet 使用 "dotnet Survivalcraft.dll" 方式启动游戏，并捕获 stdout/stderr。
// 返回已启动的进程，以及等待其退出的 channel（观察期判断与后续监控复用）。
func (g *GameManager) startViaDotnet(dllPath, workDir string, buf *syncBuffer) (*exec.Cmd, <-chan error, error) {
	dotnetPath, err := resolveDotnet()
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(dotnetPath, dllPath)
	cmd.Dir = workDir
	// 捕获 .NET 运行时输出，崩溃时可直接展示
	cmd.Stdout = buf
	cmd.Stderr = buf
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("进程创建失败: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	return cmd, waitCh, nil
}

// resolveDotnet 定位 dotnet 可执行文件。
// 优先 PATH 中的 dotnet，其次默认安装目录 %ProgramFiles%\dotnet\dotnet.exe
// （与 dotnet 包的运行时检测策略一致）。PATH 中解析到 .bat/.cmd 脚本时
// 无法直接作为子进程启动，视为不可用。
func resolveDotnet() (string, error) {
	if p, err := exec.LookPath("dotnet"); err == nil {
		if ext := strings.ToLower(filepath.Ext(p)); ext != ".bat" && ext != ".cmd" {
			return p, nil
		}
	}
	base := os.Getenv("ProgramFiles")
	if base == "" {
		base = `C:\Program Files`
	}
	fallback := filepath.Join(base, "dotnet", "dotnet.exe")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", fmt.Errorf("未找到 dotnet 运行时命令")
}

// findGameDll 查找游戏目录下的 Survivalcraft.dll（2.4+ 为 .NET 结构，可用 dotnet 直接启动）。
// 返回空字符串表示不存在。
func (g *GameManager) findGameDll(versionID, exePath string) string {
	// 优先取 exe 同目录（正常安装与导入版本均适用）
	sibling := filepath.Join(filepath.Dir(exePath), "Survivalcraft.dll")
	if _, err := os.Stat(sibling); err == nil {
		return sibling
	}

	// 回退：在版本目录中查找
	versionPath := g.paths.GetVersionPath(versionID)
	var dllPath string
	_ = filepath.Walk(versionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if dllPath == "" && strings.EqualFold(info.Name(), "Survivalcraft.dll") {
			dllPath = path
		}
		return nil
	})
	return dllPath
}

// syncBuffer 线程安全的日志缓冲区。
// exec 会为 stdout/stderr 各起一个拷贝协程并发写入，需要加锁。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// truncateLog 截断日志，超出时保留末尾（错误信息通常在末尾）。
func truncateLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max:]
}

// Stop 停止游戏
func (g *GameManager) Stop() error {
	g.processMu.Lock()
	defer g.processMu.Unlock()

	if !g.isRunning() {
		return fmt.Errorf("game is not running")
	}

	// 标记为用户主动终止（避免误报崩溃）
	g.killed = true

	// 发送终止信号
	if err := g.process.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop game: %w", err)
	}

	g.process = nil
	g.processID = 0
	g.setStatus(StatusStopped)

	return nil
}

// GetStatus 获取游戏状态
func (g *GameManager) GetStatus() Status {
	g.statusMu.RLock()
	defer g.statusMu.RUnlock()
	return g.status
}

// GetProcessInfo 获取进程信息
func (g *GameManager) GetProcessInfo() (*ProcessInfo, error) {
	g.processMu.Lock()
	defer g.processMu.Unlock()

	if !g.isRunning() {
		return nil, fmt.Errorf("game is not running")
	}

	// 从数据库获取详细信息
	process, err := g.repository.GetRunningGameProcess()
	if err != nil {
		return nil, err
	}

	return &ProcessInfo{
		PID:        process.PID,
		VersionID:  process.VersionID,
		StartTime:  process.StartTime,
		Executable: "", // 可以从版本信息中获取
	}, nil
}

// findGameExecutable 查找游戏可执行文件
func (g *GameManager) findGameExecutable(versionID string) (string, error) {
	versionPath := g.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取exe路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return "", fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "exe_path=") {
				exePath := strings.TrimPrefix(line, "exe_path=")
				// 验证exe文件是否仍然存在
				if _, err := os.Stat(exePath); err != nil {
					return "", fmt.Errorf("imported game executable not found: %s", exePath)
				}
				return exePath, nil
			}
		}

		return "", fmt.Errorf("invalid import metadata file")
	}

	// 正常安装的版本，在版本目录中查找
	var exePath string

	err := filepath.Walk(versionPath, func(path string, info os.FileInfo, err error) error {
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
		return "", fmt.Errorf("game executable not found in version directory")
	}

	return exePath, nil
}

// getGameWorkDirectory 获取游戏工作目录
func (g *GameManager) getGameWorkDirectory(versionID string, exePath string) string {
	versionPath := g.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	importedMetaFile := filepath.Join(versionPath, ".imported")
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，使用exe文件所在的目录
		return filepath.Dir(exePath)
	}

	// 正常安装的版本，使用版本目录
	return versionPath
}

// WorkDirectory 返回指定版本的游戏工作目录（供运行时环境检查等复用）。
// 逻辑等价于 Launch 流程中 findGameExecutable + getGameWorkDirectory 的组合。
func (g *GameManager) WorkDirectory(versionID string) (string, error) {
	exePath, err := g.findGameExecutable(versionID)
	if err != nil {
		return "", err
	}
	return g.getGameWorkDirectory(versionID, exePath), nil
}

// monitorProcess 监控进程
// stdout 在 dotnet 启动方式下非空，为直接捕获的 .NET 输出；
// fallbackLog 非 nil 表示 dotnet 启动失败后回退 exe，崩溃日志会附带该次输出辅助定位。
func (g *GameManager) monitorProcess(versionID, processRecordID string, stdout *syncBuffer, waitCh <-chan error, fallbackLog string) {
	// 等待进程结束（Wait 已在启动流程中开始，此处接收其结果）
	err := <-waitCh

	endTime := time.Now()
	exitCode := 0
	isCrashed := false

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// 记录日志
	if g.ctx != nil {
		runtime.LogInfo(g.ctx, fmt.Sprintf("游戏进程退出: PID=%d, 退出码=%d, killed=%v", g.processID, exitCode, g.killed))
	}

	// 被用户主动终止 → 不算崩溃
	if g.killed {
		g.setStatus(StatusStopped)
		g.killed = false
	} else if exitCode != 0 {
		// 非零退出码 + 非主动终止 = 崩溃
		g.setStatus(StatusCrashed)
		isCrashed = true
		if g.ctx != nil {
			runtime.LogError(g.ctx, fmt.Sprintf("游戏崩溃: PID=%d, 退出码=%d", g.processID, exitCode))
		}
	} else {
		g.setStatus(StatusStopped)
	}

	// 清理进程信息
	g.processMu.Lock()
	g.process = nil
	g.processID = 0
	g.processMu.Unlock()

	// 更新数据库记录
	if err := g.repository.UpdateGameProcessEnded(processRecordID, endTime, exitCode); err != nil {
		fmt.Printf("Warning: failed to update process record: %v\n", err)
	}

	// 如果是崩溃，发送通知到前端
	if isCrashed && g.ctx != nil {
		version, _ := g.repository.GetVersion(versionID)
		versionName := versionID
		if version != nil {
			versionName = version.Name
		}

		// 获取崩溃日志：优先用 dotnet 捕获的输出，为空则读 Game.log
		logContent := ""
		if stdout != nil && stdout.Len() > 0 {
			logContent = stdout.String()
		} else {
			logContent = g.readGameLog(versionID)
		}
		if logContent == "" {
			logContent = "游戏崩溃退出（日志未捕获）"
		}

		// 若曾用 dotnet 启动失败后回退 exe，附带该次输出辅助定位
		// （exe 直接崩溃往往只有一个退出码，如 2148734214，dotnet 的输出才有具体原因）
		if fallbackLog != "" {
			logContent = "===== dotnet Survivalcraft.dll 启动失败，已回退 Survivalcraft.exe =====\n" +
				"【dotnet 输出】\n" + fallbackLog + "\n\n" +
				"【Survivalcraft.exe 运行日志】\n" + logContent
		}

		runtime.EventsEmit(g.ctx, "game:crashed", map[string]interface{}{
			"versionId":   versionID,
			"versionName": versionName,
			"exitCode":    exitCode,
			"log":         logContent,
			"crashTime":   endTime.Format("2006-01-02 15:04:05"),
		})
	}
}

// readGameLog 读取游戏目录下的 Game.log（从启动前的偏移量开始，只取本次运行的日志）
// 高版本路径：{gameDir}/Bugs/Game.log
// 低版本路径：{gameDir}/Logs/Game.log
func (g *GameManager) readGameLog(versionID string) string {
	if g.gameWorkDir == "" {
		runtime.LogWarning(g.ctx, "[GameLog] gameWorkDir is empty")
		return ""
	}

	// 尝试两个可能的路径
	candidates := []string{
		filepath.Join(g.gameWorkDir, "Bugs", "Game.log"),
		filepath.Join(g.gameWorkDir, "Logs", "Game.log"),
	}

	for _, gameLogPath := range candidates {
		runtime.LogInfo(g.ctx, fmt.Sprintf("[GameLog] Trying: %s (offset=%d)", gameLogPath, g.gameLogOffset))

		file, err := os.Open(gameLogPath)
		if err != nil {
			runtime.LogInfo(g.ctx, fmt.Sprintf("[GameLog] Not found: %v", err))
			continue
		}

		// 从启动前的偏移量开始读取
		if g.gameLogOffset > 0 {
			if _, err := file.Seek(g.gameLogOffset, 0); err != nil {
				file.Close()
				continue
			}
		}

		// 获取文件大小，只读取末尾部分（Game.log 可能很大）
		stat, _ := file.Stat()
		if stat == nil {
			file.Close()
			continue
		}

		// 计算本次运行的日志大小
		logSize := stat.Size() - g.gameLogOffset
		if logSize <= 0 {
			file.Close()
			continue
		}

		// 只读取最后 10KB（避免日志过大撑爆弹窗）
		const maxLogSize int64 = 10 * 1024
		if logSize > maxLogSize {
			// 跳到末尾前 50KB 的位置
			file.Seek(-maxLogSize, 2)
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		content := string(data)
		if content != "" {
			return content
		}
	}

	return ""
}

// isRunning 检查游戏是否正在运行
func (g *GameManager) isRunning() bool {
	g.statusMu.RLock()
	defer g.statusMu.RUnlock()
	return g.status == StatusRunning
}

// setStatus 设置状态
func (g *GameManager) setStatus(status Status) {
	g.statusMu.Lock()
	defer g.statusMu.Unlock()
	g.status = status
}

// generateProcessID 生成进程记录 ID
func generateProcessID() string {
	return fmt.Sprintf("proc-%d", time.Now().UnixNano())
}
