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
