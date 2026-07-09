package dotnet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewManager(t *testing.T) {
	if m := NewManager(); m == nil {
		t.Fatal("nil manager")
	}
}

// TestManagerDefaults 覆盖默认实现分支：defaultFetchReleases / httpGet(HTTPClient) /
// sharedDir / runner，全部不触网（重定向 releasesBase 到本地 httptest）。
func TestManagerDefaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"releases":[{"release-version":"9.0.0","windowsdesktop":{"version":"9.0.0","files":[]}}]}`))
	}))
	defer srv.Close()

	prev := releasesBase
	releasesBase = srv.URL
	defer func() { releasesBase = prev }()

	t.Run("fetch with explicit client", func(t *testing.T) {
		m := &Manager{HTTPClient: srv.Client()}
		data, err := m.defaultFetchReleases(9)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			t.Fatal("empty data")
		}
	})
	t.Run("fetch with default client (nil)", func(t *testing.T) {
		m := &Manager{} // HTTPClient nil → 走 DefaultClient
		if _, err := m.defaultFetchReleases(9); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fetch non-200", func(t *testing.T) {
		badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer badSrv.Close()
		saved := releasesBase
		releasesBase = badSrv.URL
		defer func() { releasesBase = saved }()
		if _, err := (&Manager{}).defaultFetchReleases(9); err == nil {
			t.Fatal("expected error on 500")
		}
	})
	t.Run("default sharedDir and runner", func(t *testing.T) {
		m := &Manager{}
		if m.sharedDir() == "" {
			t.Fatal("default sharedDir empty")
		}
		if m.runner() == nil {
			t.Fatal("default runner nil")
		}
	})
}

func TestManagerStatus(t *testing.T) {
	t.Run("needed not installed", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "g.runtimeconfig.json",
			`{"runtimeOptions":{"tfm":"net10.0","frameworks":[{"name":"Microsoft.WindowsDesktop.App","version":"10.0.0"}]}}`)
		m := &Manager{
			Runner:   &fakeRunner{runFn: func(string, []string) (string, string, int, error) { return "Microsoft.NETCore.App 9.0.0 [p]", "", 0, nil }},
			SharedDir: t.TempDir(),
		}
		st, err := m.Status(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !st.Needed || st.MajorVersion != 10 || st.Installed {
			t.Errorf("unexpected: %+v", st)
		}
	})
	t.Run("needed and installed via dotnet", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "g.runtimeconfig.json",
			`{"runtimeOptions":{"tfm":"net10.0","frameworks":[{"name":"Microsoft.WindowsDesktop.App","version":"10.0.0"}]}}`)
		m := &Manager{
			Runner:   &fakeRunner{runFn: func(string, []string) (string, string, int, error) { return "Microsoft.WindowsDesktop.App 10.0.0 [p]", "", 0, nil }},
			SharedDir: t.TempDir(),
		}
		st, _ := m.Status(dir)
		if !st.Installed {
			t.Errorf("expected installed: %+v", st)
		}
	})
	t.Run("not needed", func(t *testing.T) {
		m := &Manager{Runner: &fakeRunner{}, SharedDir: t.TempDir()}
		st, _ := m.Status(t.TempDir())
		if st.Needed {
			t.Errorf("expected not needed: %+v", st)
		}
	})
}

func TestManagerIsInstalled(t *testing.T) {
	t.Run("via dotnet", func(t *testing.T) {
		m := &Manager{
			Runner:   &fakeRunner{runFn: func(string, []string) (string, string, int, error) { return "Microsoft.WindowsDesktop.App 9.0.0 [p]", "", 0, nil }},
			SharedDir: t.TempDir(),
		}
		if !m.IsInstalled(9) {
			t.Error("expected installed")
		}
		if m.IsInstalled(10) {
			t.Error("expected not installed")
		}
	})
	t.Run("via dir fallback", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.MkdirAll(dir+"/9.0.16", 0755)
		m := &Manager{
			Runner:   &fakeRunner{runFn: func(string, []string) (string, string, int, error) { return "Microsoft.NETCore.App 8.0.0 [p]", "", 0, nil }},
			SharedDir: dir,
		}
		if !m.IsInstalled(9) {
			t.Error("expected installed via dir")
		}
	})
	t.Run("dotnet missing and not in dir", func(t *testing.T) {
		m := &Manager{
			Runner:    &fakeRunner{lookPathFn: func(string) error { return os.ErrNotExist }},
			SharedDir: t.TempDir(),
		}
		if m.IsInstalled(9) {
			t.Error("expected not installed")
		}
	})
}

func TestManagerInstall(t *testing.T) {
	t.Run("download success", func(t *testing.T) {
		payload := []byte("installer")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		fixture := fmt.Sprintf(
			`{"releases":[{"release-version":"10.0.0","windowsdesktop":{"version":"10.0.0","files":[{"name":"x.exe","rid":"win-x64","url":%q,"hash":%q}]}}]}`,
			srv.URL+"/x.exe", sha512Hex(payload),
		)

		var progressed bool
		var installerRan bool
		r := &fakeRunner{runFn: func(name string, args []string) (string, string, int, error) {
			installerRan = true
			return "", "", 0, nil
		}}
		prev := currentArch
		currentArch = func() string { return "amd64" }
		defer func() { currentArch = prev }()

		m := &Manager{
			Runner:        r,
			SharedDir:     t.TempDir(),
			FetchReleases: func(int) ([]byte, error) { return []byte(fixture), nil },
			Logger:        func(string, ...any) {},
		}
		if err := m.Install(10, func(int64, int64) { progressed = true }); err != nil {
			t.Fatal(err)
		}
		if !installerRan {
			t.Error("installer not run")
		}
		if !progressed {
			t.Error("no progress reported")
		}
	})

	t.Run("installer run fails", func(t *testing.T) {
		payload := []byte("installer")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
		defer srv.Close()
		fixture := fmt.Sprintf(
			`{"releases":[{"release-version":"10.0.0","windowsdesktop":{"version":"10.0.0","files":[{"name":"x.exe","rid":"win-x64","url":%q,"hash":%q}]}}]}`,
			srv.URL+"/x.exe", sha512Hex(payload),
		)
		r := &fakeRunner{runFn: func(name string, args []string) (string, string, int, error) {
			return "", "fail", 5, nil // 安装程序退出码非 0/3010
		}}
		prev := currentArch
		currentArch = func() string { return "amd64" }
		defer func() { currentArch = prev }()
		m := &Manager{
			Runner:        r,
			SharedDir:     t.TempDir(),
			FetchReleases: func(int) ([]byte, error) { return []byte(fixture), nil },
		}
		if err := m.Install(10, nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		m := &Manager{
			Runner:        &fakeRunner{},
			SharedDir:     t.TempDir(),
			FetchReleases: func(int) ([]byte, error) { return nil, os.ErrNotExist },
		}
		if err := m.Install(10, nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no matching asset", func(t *testing.T) {
		fixture := `{"releases":[{"release-version":"10.0.0","windowsdesktop":{"version":"10.0.0","files":[{"name":"x.zip","rid":"win-x64","url":"http://x","hash":"z"}]}}]}`
		prev := currentArch
		currentArch = func() string { return "amd64" }
		defer func() { currentArch = prev }()
		m := &Manager{
			Runner:        &fakeRunner{},
			SharedDir:     t.TempDir(),
			FetchReleases: func(int) ([]byte, error) { return []byte(fixture), nil },
		}
		if err := m.Install(10, nil); err == nil {
			t.Fatal("expected error")
		}
	})
}
