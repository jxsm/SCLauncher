package dotnet

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CommandRunner 抽象命令执行，便于单测注入。
type CommandRunner interface {
	// LookPath 等价于 exec.LookPath。
	LookPath(file string) (string, error)
	// Run 执行命令并返回 stdout、stderr、退出码与执行错误。
	Run(name string, args ...string) (stdout, stderr string, exitCode int, err error)
}

// ListInstalledDesktopMajors 通过 `dotnet --list-runtimes` 列出本机已安装的
// WindowsDesktop 运行时主版本集合。
//
// dotnet 不在 PATH 或命令失败时返回空集（视为未安装），不向上抛错以免阻塞启动。
func ListInstalledDesktopMajors(runner CommandRunner) (map[int]bool, error) {
	if runner == nil {
		return nil, fmt.Errorf("runner is nil")
	}
	if _, err := runner.LookPath("dotnet"); err != nil {
		return map[int]bool{}, nil
	}
	stdout, _, _, err := runner.Run("dotnet", "--list-runtimes")
	if err != nil {
		return map[int]bool{}, nil
	}
	return ParseListRuntimes(stdout), nil
}

// ParseListRuntimes 从 `dotnet --list-runtimes` 输出中提取 WindowsDesktop 主版本。
func ParseListRuntimes(stdout string) map[int]bool {
	majors := map[int]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 形如 "Microsoft.WindowsDesktop.App 9.0.16 [C:\Program Files\dotnet\shared\...]"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "Microsoft.WindowsDesktop.App" {
			continue
		}
		if m, ok := versionMajor(fields[1]); ok {
			majors[m] = true
		}
	}
	return majors
}

// versionMajor 将 "9.0.16" 解析为主版本 9。
func versionMajor(v string) (int, bool) {
	dot := strings.IndexByte(v, '.')
	major := v
	if dot >= 0 {
		major = v[:dot]
	}
	n, err := strconv.Atoi(major)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// MajorsFromInstallDir 扫描 dotnet 安装目录下以版本号命名的子目录，
// 作为 `dotnet --list-runtimes` 不可用时的回退。
//
// dir 通常是 ...\dotnet\shared\Microsoft.WindowsDesktop.App，其下每个子目录形如 9.0.16。
func MajorsFromInstallDir(dir string) (map[int]bool, error) {
	majors := map[int]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return majors, nil // 目录不存在视为未安装，不报错
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if m, ok := versionMajor(e.Name()); ok {
			majors[m] = true
		}
	}
	return majors, nil
}

// DefaultDesktopSharedPath 返回默认的 Desktop 运行时安装路径（用于回退）。
// 兼容 64 位系统：在 64 位进程里 %ProgramFiles% 指向 C:\Program Files。
func DefaultDesktopSharedPath() string {
	base := os.Getenv("ProgramFiles")
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, "dotnet", "shared", "Microsoft.WindowsDesktop.App")
}
