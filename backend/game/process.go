package game

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"SCLauncher/backend/config"
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
	ctx        context.Context
	config     *config.Config
	paths      *config.Paths
	repository *storage.Repository
	process    *exec.Cmd
	processID  int
	status     Status
	statusMu   sync.RWMutex
	processMu  sync.Mutex
	logBuffer  *bytes.Buffer // 日志缓冲区
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

	// 清空日志缓冲区
	g.logBuffer.Reset()

	// 创建进程
	cmd := exec.Command(exePath)
	cmd.Dir = g.paths.GetVersionPath(versionID)

	// 不要捕获输出，这对GUI程序会有影响
	// 只记录日志到文件，不重定向stdout/stderr

	// Windows: 不设置 CREATE_NO_WINDOW，让游戏可以正常显示窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false, // ⭐ 不隐藏窗口
		CreationFlags: 0,      // ⭐ 不使用 CREATE_NO_WINDOW
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start game: %w", err)
	}

	// 记录日志
	if g.ctx != nil {
		runtime.LogInfo(g.ctx, fmt.Sprintf("游戏进程启动成功: PID=%d, 可执行文件=%s", cmd.Process.Pid, exePath))
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

	// 启动监控协程（不传递输出缓冲区）
	go g.monitorProcess(versionID, processRecord.ID, nil, nil)

	return nil
}

// Stop 停止游戏
func (g *GameManager) Stop() error {
	g.processMu.Lock()
	defer g.processMu.Unlock()

	if !g.isRunning() {
		return fmt.Errorf("game is not running")
	}

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

// monitorProcess 监控进程
func (g *GameManager) monitorProcess(versionID, processRecordID string, stdout, stderr *bytes.Buffer) {
	// 等待进程结束
	err := g.process.Wait()

	endTime := time.Now()
	exitCode := 0
	isCrashed := false

	// 记录日志
	if g.ctx != nil {
		runtime.LogInfo(g.ctx, fmt.Sprintf("游戏进程退出: PID=%d, 退出码=%d", g.processID, exitCode))
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			g.setStatus(StatusCrashed)
			isCrashed = true

			// 记录崩溃日志
			if g.ctx != nil {
				runtime.LogError(g.ctx, fmt.Sprintf("游戏崩溃: PID=%d, 退出码=%d", g.processID, exitCode))
			}
		} else {
			g.setStatus(StatusStopped)
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

	// 如果是崩溃，发送通知到前端（无日志内容，因为我们没有捕获）
	if isCrashed && g.ctx != nil {
		// 获取版本信息
		version, _ := g.repository.GetVersion(versionID)
		versionName := versionID
		if version != nil {
			versionName = version.Name
		}

		runtime.EventsEmit(g.ctx, "game:crashed", map[string]interface{}{
			"versionId":   versionID,
			"versionName": versionName,
			"exitCode":    exitCode,
			"log":         "游戏崩溃退出（日志未捕获）",
			"crashTime":   endTime.Format("2006-01-02 15:04:05"),
		})
	}
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
