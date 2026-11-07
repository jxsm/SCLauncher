package main

import (
	"regexp"
	"testing"

	"SCLauncher/backend/appinfo"
)

// TestWailsConfigProvidesVersion 确保版本号唯一来源 wails.json 的
// info.productVersion 配置正确且能注入 appinfo.Version，
// 防止发版时漏改版本号或 JSON 手误（缺字段/格式错误）
func TestWailsConfigProvidesVersion(t *testing.T) {
	appinfo.SetWailsConfig(wailsConfig)

	if appinfo.Version == "0.0.0" {
		t.Fatal("wails.json 缺少 info.productVersion 字段，或 JSON 无法解析，appinfo.Version 未被注入")
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(appinfo.Version) {
		t.Fatalf("版本号格式应为 x.y.z，当前为 %q", appinfo.Version)
	}
}

// TestCompareVersions 覆盖更新检测的版本比较逻辑
func TestCompareVersions(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"相同版本", "1.0.8", "1.0.8", false},
		{"最新版更大-主版本", "1.0.8", "2.0.0", true},
		{"最新版更大-次版本", "1.0.8", "1.1.0", true},
		{"最新版更大-修订号", "1.0.8", "1.0.9", true},
		{"最新版更小", "1.0.8", "1.0.7", false},
		{"最新版更旧-主版本", "2.0.0", "1.9.9", false},
		{"最新版位数更少", "1.0.8", "1.1", true},   // 1.1 → 1.1.0
		{"当前版位数更少", "1.1", "1.0.8", false}, // 1.1 → 1.1.0
		{"跨大版本", "1.9.9", "2.0.0", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := compareVersions(c.current, c.latest); got != c.want {
				t.Errorf("compareVersions(%q, %q) = %v, 期望 %v", c.current, c.latest, got, c.want)
			}
		})
	}
}
