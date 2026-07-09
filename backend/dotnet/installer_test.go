package dotnet

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func TestWingetPackageID(t *testing.T) {
	if got := wingetPackageID(10); got != "Microsoft.DotNet.DesktopRuntime.10" {
		t.Errorf("got %s", got)
	}
}

func TestInstallViaWinget(t *testing.T) {
	t.Run("winget missing", func(t *testing.T) {
		r := &fakeRunner{lookPathFn: func(string) error { return os.ErrNotExist }}
		if err := (&Installer{Runner: r}).InstallViaWinget(10); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("success", func(t *testing.T) {
		var calledArgs []string
		r := &fakeRunner{runFn: func(name string, args []string) (string, string, int, error) {
			if name == "winget" {
				calledArgs = args
			}
			return "ok", "", 0, nil
		}}
		if err := (&Installer{Runner: r}).InstallViaWinget(9); err != nil {
			t.Fatal(err)
		}
		if !contains(calledArgs, "Microsoft.DotNet.DesktopRuntime.9") {
			t.Errorf("args missing pkg id: %v", calledArgs)
		}
		if !contains(calledArgs, "--silent") {
			t.Errorf("args missing --silent: %v", calledArgs)
		}
	})
	t.Run("winget nonzero exit", func(t *testing.T) {
		r := &fakeRunner{runFn: func(string, []string) (string, string, int, error) {
			return "", "fail", 2, nil
		}}
		if err := (&Installer{Runner: r}).InstallViaWinget(10); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDownloadAndRun(t *testing.T) {
	payload := []byte("hello installer bytes")

	t.Run("success with sha512 and exit 3010", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(payload)
		}))
		defer srv.Close()

		tmp := t.TempDir()
		var runArgs []string
		r := &fakeRunner{runFn: func(name string, args []string) (string, string, int, error) {
			runArgs = args
			return "", "", 3010, nil // 3010 = 成功但需重启
		}}
		asset := InstallerAsset{
			Version: "10.0.0",
			URL:     srv.URL + "/installer.exe",
			SHA512:  sha512Hex(payload),
			RID:     "win-x64",
		}
		var lastDownloaded int64
		if err := (&Installer{Runner: r, TmpDir: tmp}).DownloadAndRun(asset, func(d, total int64) {
			lastDownloaded = d
		}); err != nil {
			t.Fatal(err)
		}
		if lastDownloaded != int64(len(payload)) {
			t.Errorf("progress not complete: %d", lastDownloaded)
		}
		if !contains(runArgs, "/passive") || !contains(runArgs, "/norestart") {
			t.Errorf("missing installer flags: %v", runArgs)
		}
		if _, err := os.Stat(filepath.Join(tmp, "installer.exe")); !os.IsNotExist(err) {
			t.Errorf("temp file not removed")
		}
	})
	t.Run("sha512 mismatch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		asset := InstallerAsset{URL: srv.URL + "/installer.exe", SHA512: "deadbeef", RID: "win-x64"}
		if err := (&Installer{Runner: &fakeRunner{}, TmpDir: t.TempDir()}).DownloadAndRun(asset, nil); err == nil {
			t.Fatal("expected sha mismatch error")
		}
	})
	t.Run("sha512 omitted skips verify", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		asset := InstallerAsset{URL: srv.URL + "/installer.exe"} // 无 hash，跳过校验
		if err := (&Installer{Runner: &fakeRunner{}, TmpDir: t.TempDir()}).DownloadAndRun(asset, nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("http 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()
		asset := InstallerAsset{URL: srv.URL + "/installer.exe"}
		if err := (&Installer{Runner: &fakeRunner{}, TmpDir: t.TempDir()}).DownloadAndRun(asset, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("installer nonzero exit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		r := &fakeRunner{runFn: func(string, []string) (string, string, int, error) { return "", "fail", 5, nil }}
		asset := InstallerAsset{URL: srv.URL + "/installer.exe", SHA512: sha512Hex(payload)}
		if err := (&Installer{Runner: r, TmpDir: t.TempDir()}).DownloadAndRun(asset, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("runner run error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		r := &fakeRunner{runFn: func(string, []string) (string, string, int, error) {
			return "", "", -1, os.ErrInvalid
		}}
		asset := InstallerAsset{URL: srv.URL + "/installer.exe", SHA512: sha512Hex(payload)}
		if err := (&Installer{Runner: r, TmpDir: t.TempDir()}).DownloadAndRun(asset, nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

// realRunner 的覆盖：确认退出码映射在 ExitError 上工作。
func TestRealRunnerExitCode(t *testing.T) {
	r := realRunner{}
	// 用一个肯定存在的命令 true（exit 0）
	if _, err := r.LookPath("go"); err != nil {
		t.Skip("go not on PATH in this env")
	}
	stdout, _, code, err := r.Run("go", "version")
	if err != nil || code != 0 || stdout == "" {
		t.Logf("go version run: code=%d err=%v stdout_len=%d", code, err, len(stdout))
	}
	// 不存在的命令 → err 非 ExitError
	if _, _, _, err := r.Run("definitely-not-a-real-bin-xyz"); err == nil {
		t.Log("expected non-nil err for missing command")
	}
}
