package dotnet

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Installer 负责 winget 安装以及「下载安装包 + 运行」两条路径。
type Installer struct {
	Runner   CommandRunner                                              // 默认 realRunner
	HTTPGet  func(url string) (*http.Response, error)                  // 默认 http.DefaultClient.Get
	Logger   func(format string, args ...any)
	TmpDir   string // 默认 os.TempDir()
}

func (in *Installer) logf(format string, args ...any) {
	if in.Logger != nil {
		in.Logger(format, args...)
	}
}

func (in *Installer) runner() CommandRunner {
	if in.Runner != nil {
		return in.Runner
	}
	return &realRunner{}
}

func (in *Installer) httpGet(url string) (*http.Response, error) {
	if in.HTTPGet != nil {
		return in.HTTPGet(url)
	}
	return http.DefaultClient.Get(url)
}

func (in *Installer) tmpDir() string {
	if in.TmpDir != "" {
		return in.TmpDir
	}
	return os.TempDir()
}

// wingetPackageID 返回某大版本对应的 winget 包 ID。
func wingetPackageID(major int) string {
	return fmt.Sprintf("Microsoft.DotNet.DesktopRuntime.%d", major)
}

// InstallViaWinget 用 winget 安装指定大版本；winget 不可用或失败返回 error。
func (in *Installer) InstallViaWinget(major int) error {
	r := in.runner()
	if _, err := r.LookPath("winget"); err != nil {
		return fmt.Errorf("winget 不可用: %w", err)
	}
	pkgID := wingetPackageID(major)
	args := []string{
		"install", "--id", pkgID, "-e", "--silent",
		"--accept-package-agreements", "--accept-source-agreements",
	}
	in.logf("winget 安装 %s", pkgID)
	stdout, stderr, code, err := r.Run("winget", args...)
	if err != nil {
		return fmt.Errorf("winget 执行失败: %w (stderr=%s)", err, stderr)
	}
	if code != 0 {
		return fmt.Errorf("winget 退出码 %d (stdout=%s stderr=%s)", code, stdout, stderr)
	}
	return nil
}

// DownloadAndRun 下载安装包（带进度回调 + sha512 校验），以 /passive /norestart 运行。
// 安装程序退出码 0 与 3010（成功但需重启）均视为成功。
func (in *Installer) DownloadAndRun(asset InstallerAsset, progress func(downloaded, total int64)) error {
	resp, err := in.httpGet(asset.URL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码 %d", resp.StatusCode)
	}

	if err := os.MkdirAll(in.tmpDir(), 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	dest := filepath.Join(in.tmpDir(), filepath.Base(asset.URL))
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	hasher := sha512.New()
	written := int64(0)
	total := resp.ContentLength
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				_ = os.Remove(dest)
				return fmt.Errorf("写入临时文件失败: %w", werr)
			}
			hasher.Write(buf[:n])
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if rerr != nil {
			f.Close()
			if rerr == io.EOF {
				break
			}
			_ = os.Remove(dest)
			return fmt.Errorf("下载读取失败: %w", rerr)
		}
	}

	// sha512 校验
	if asset.SHA512 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		want := strings.ToLower(strings.TrimSpace(asset.SHA512))
		if got != want {
			_ = os.Remove(dest)
			return fmt.Errorf("sha512 校验失败: 期望 %s 实际 %s", want, got)
		}
	}

	// 运行安装程序 /passive /norestart
	r := in.runner()
	in.logf("运行安装程序 %s /passive", dest)
	stdout, stderr, code, err := r.Run(dest, "/passive", "/norestart")
	if err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("安装程序执行失败: %w (stderr=%s)", err, stderr)
	}
	if code != 0 && code != 3010 {
		_ = os.Remove(dest)
		return fmt.Errorf("安装程序退出码 %d (stdout=%s stderr=%s)", code, stdout, stderr)
	}
	_ = os.Remove(dest)
	return nil
}

// realRunner 是生产环境的 CommandRunner 实现。
type realRunner struct{}

func (realRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (realRunner) Run(name string, args ...string) (string, string, int, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Windows：抑制子进程控制台窗口。dotnet / winget 是控制台程序，不加这个会闪一下黑框。
	// CREATE_NO_WINDOW 只阻止分配控制台，不影响 GUI 子系统程序（如 .NET 安装器的图形进度窗）。
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdout.String(), stderr.String(), exitErr.ExitCode(), nil
		}
		return stdout.String(), stderr.String(), -1, err
	}
	return stdout.String(), stderr.String(), 0, nil
}
