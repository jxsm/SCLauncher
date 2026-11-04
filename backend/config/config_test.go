package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfigAutoCheckRuntime 默认配置应开启运行时自动检查。
func TestDefaultConfigAutoCheckRuntime(t *testing.T) {
	c := DefaultConfig()
	if !c.AutoCheckRuntime {
		t.Error("expected DefaultConfig().AutoCheckRuntime == true")
	}
}

// writeConfig 写一个合法的配置 JSON（含目录字段，避免 EnsureDirs 因空路径失败），
// 用 map + Marshal 自动处理反斜杠转义。
func writeConfig(t *testing.T, path string, extra map[string]any) {
	t.Helper()
	dir := filepath.Dir(path)
	m := map[string]any{
		"versionsDir":  filepath.Join(dir, "v"),
		"dataDir":      filepath.Join(dir, "d"),
		"downloadsDir": filepath.Join(dir, "dl"),
	}
	for k, v := range extra {
		m[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadOldConfigDefaultsRuntimeOn 旧配置（缺少 autoCheckRuntime 字段）应默认置为开启。
func TestLoadOldConfigDefaultsRuntimeOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, map[string]any{"manifestUrl": "x"})
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !c.AutoCheckRuntime {
		t.Error("expected old config to default AutoCheckRuntime=true")
	}
}

// TestLoadRespectsExplicitRuntimeOff 显式写 false 必须被尊重（不会被默认逻辑覆盖）。
func TestLoadRespectsExplicitRuntimeOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, map[string]any{"manifestUrl": "x", "autoCheckRuntime": false})
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.AutoCheckRuntime {
		t.Error("expected explicit autoCheckRuntime:false to be respected")
	}
}

// TestLoadExplicitRuntimeOn 显式写 true 也应保持开启。
func TestLoadExplicitRuntimeOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, map[string]any{"autoCheckRuntime": true})
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !c.AutoCheckRuntime {
		t.Error("expected explicit autoCheckRuntime:true")
	}
}

// TestSetAutoCheckRuntimePersists setter 应写入磁盘并可被 Load 读回。
func TestSetAutoCheckRuntimePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	c := DefaultConfig()
	c.configPath = path
	c.VersionsDir = filepath.Join(dir, "v")
	c.DataDir = filepath.Join(dir, "d")
	c.DownloadsDir = filepath.Join(dir, "dl")

	if err := c.SetAutoCheckRuntime(false); err != nil {
		t.Fatal(err)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2.AutoCheckRuntime {
		t.Error("expected persisted autoCheckRuntime:false")
	}
}

// TestPickFastestLine 最快线路选择：镜像更快时选镜像；全部失败时保持不确定。
func TestPickFastestLine(t *testing.T) {
	cases := []struct {
		name    string
		results []ManifestSpeedResult
		wantURL string
		wantOK  bool
	}{
		{
			name: "mirror faster",
			results: []ManifestSpeedResult{
				{Line: ManifestLineMain, URL: DefaultManifestURLMain, LatencyMs: 1000, OK: true},
				{Line: ManifestLineMirror, URL: DefaultManifestURLMirror, LatencyMs: 300, OK: true},
			},
			wantURL: DefaultManifestURLMirror,
			wantOK:  true,
		},
		{
			name: "main faster",
			results: []ManifestSpeedResult{
				{Line: ManifestLineMain, URL: DefaultManifestURLMain, LatencyMs: 200, OK: true},
				{Line: ManifestLineMirror, URL: DefaultManifestURLMirror, LatencyMs: 900, OK: true},
			},
			wantURL: DefaultManifestURLMain,
			wantOK:  true,
		},
		{
			name: "mirror failed uses main",
			results: []ManifestSpeedResult{
				{Line: ManifestLineMain, URL: DefaultManifestURLMain, LatencyMs: 1000, OK: true},
				{Line: ManifestLineMirror, URL: DefaultManifestURLMirror, LatencyMs: -1, OK: false},
			},
			wantURL: DefaultManifestURLMain,
			wantOK:  true,
		},
		{
			name: "all failed",
			results: []ManifestSpeedResult{
				{Line: ManifestLineMain, URL: DefaultManifestURLMain, LatencyMs: -1, OK: false},
				{Line: ManifestLineMirror, URL: DefaultManifestURLMirror, LatencyMs: -1, OK: false},
			},
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			best, ok := PickFastestLine(tc.results)
			if ok != tc.wantOK {
				t.Fatalf("expected ok=%v, got %v", tc.wantOK, ok)
			}
			if ok && best.URL != tc.wantURL {
				t.Errorf("expected %s, got %s", tc.wantURL, best.URL)
			}
		})
	}
}

// TestLoadOldConfigDefaultsManifestURL 旧配置（无 defaultManifestUrl 字段）应回退为主线地址。
func TestLoadOldConfigDefaultsManifestURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeConfig(t, path, map[string]any{"manifestUrl": "x"})
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultManifestURL != DefaultManifestURLMain {
		t.Errorf("expected DefaultManifestURL=%s, got %s", DefaultManifestURLMain, c.DefaultManifestURL)
	}
}

// TestDefaultManifestSpeedTestPersists 测速结果应写入磁盘并可被 Load 读回。
func TestDefaultManifestSpeedTestPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	writeConfig(t, path, nil)
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// 模拟测速：直接构造结果并走应用逻辑（不发起网络请求）
	c.DefaultManifestURL = DefaultManifestURLMirror
	c.DefaultManifestMainLatencyMs = 1200
	c.DefaultManifestMirrorLatencyMs = 350
	c.DefaultManifestLastTest = 1700000000
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2.DefaultManifestURL != DefaultManifestURLMirror {
		t.Errorf("expected persisted DefaultManifestURL=%s, got %s", DefaultManifestURLMirror, c2.DefaultManifestURL)
	}
	if c2.DefaultManifestMirrorLatencyMs != 350 || c2.DefaultManifestLastTest != 1700000000 {
		t.Error("expected latency/lastTest values to persist")
	}
}
