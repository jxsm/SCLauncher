package scbbs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExportSaveGamePackRoundTrip 导出 → 解析 → 安装全链路：
// 世界文件夹 + 两个模组导出为 scbbs ZIP，重新解析并安装到临时版本目录，
// 断言世界以文件夹形式还原、模组落到 Mods。
func TestExportSaveGamePackRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// 世界文件夹（模拟存档目录）
	worldPath := filepath.Join(dir, "我的世界")
	if err := os.MkdirAll(filepath.Join(worldPath, "Chunks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldPath, "Project.json"), []byte(`{"Subsystems":{"GameInfo":{"WorldName":"我的世界"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldPath, "Chunks", "0.chunk"), []byte("chunk-data"), 0644); err != nil {
		t.Fatal(err)
	}

	// 模组文件
	modA := filepath.Join(dir, "a.scmod")
	if err := os.WriteFile(modA, makeScmodBytes(t, "com.a", "1.0"), 0644); err != nil {
		t.Fatal(err)
	}
	modB := filepath.Join(dir, "b.scmod")
	if err := os.WriteFile(modB, makeScmodBytes(t, "com.b", "2.0"), 0644); err != nil {
		t.Fatal(err)
	}

	packPath := filepath.Join(dir, "pack.zip")
	err := ExportSaveGamePack(worldPath, "我的世界", "", []ExportMod{
		{SourcePath: modA, FileName: "a.scmod", PackageName: "com.a", Version: "1.0"},
		{SourcePath: modB, FileName: "b.scmod", PackageName: "com.b", Version: "2.0"},
	}, packPath)
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}

	// --- 解析断言 ---
	result, err := Parse(packPath)
	if err != nil {
		t.Fatalf("导出的包应能被本启动器解析: %v", err)
	}
	m := result.Manifest
	if m.Name != "我的世界" || m.PackageName != "我的世界" || m.Version != "1.0.0" {
		t.Errorf("清单基本信息错误: %+v", m)
	}
	if m.Description != "我的世界" {
		t.Errorf("空描述应回退存档名，实际 %q", m.Description)
	}
	if len(m.Items) != 3 {
		t.Fatalf("条目应为 3（1 世界 + 2 模组），实际 %d", len(m.Items))
	}
	if len(result.MissingEmbedded) != 0 {
		t.Errorf("不应有缺失内嵌项: %d", len(result.MissingEmbedded))
	}
	if m.Items[0].ResolveType() != ItemTypeWorld {
		t.Errorf("首条目应为 world，实际 %s", m.Items[0].ResolveType())
	}
	for _, item := range m.Items[1:] {
		if item.ResolveType() != ItemTypeMod {
			t.Errorf("模组条目类型错误: %s", item.ResolveType())
		}
		if item.PackageName == "" || item.Version == "" {
			t.Errorf("模组条目应带校验字段: %+v", item)
		}
	}

	// --- 安装 round-trip ---
	inst, versionPath := newTestInstaller(t)
	summary, err := inst.Install(result, "test-version")
	if err != nil {
		t.Fatalf("安装失败: %v", err)
	}
	if summary.Succeeded != 3 || summary.Failed != 0 {
		t.Errorf("安装汇总错误: %+v", summary)
	}

	// 世界以文件夹形式还原
	worldJson := filepath.Join(versionPath, "Worlds", "我的世界", "Project.json")
	if _, err := os.Stat(worldJson); err != nil {
		t.Errorf("世界未还原: %v", err)
	}
	chunk := filepath.Join(versionPath, "Worlds", "我的世界", "Chunks", "0.chunk")
	if data, err := os.ReadFile(chunk); err != nil || string(data) != "chunk-data" {
		t.Errorf("世界区块文件内容错误: %v %q", err, data)
	}
	// 模组落地
	for _, name := range []string{"a.scmod", "b.scmod"} {
		if _, err := os.Stat(filepath.Join(versionPath, "Mods", name)); err != nil {
			t.Errorf("模组 %s 未落地: %v", name, err)
		}
	}
}

// TestExportSaveGamePackPureWorld 无模组的纯存档包
func TestExportSaveGamePackPureWorld(t *testing.T) {
	dir := t.TempDir()
	worldPath := filepath.Join(dir, "W")
	if err := os.MkdirAll(worldPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldPath, "Project.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	packPath := filepath.Join(dir, "pure.zip")
	if err := ExportSaveGamePack(worldPath, "纯存档", "描述", nil, packPath); err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	result, err := Parse(packPath)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(result.Manifest.Items) != 1 || result.Manifest.Items[0].ResolveType() != ItemTypeWorld {
		t.Errorf("应只有 1 个 world 条目: %+v", result.Manifest.Items)
	}
	if result.Manifest.Description != "描述" {
		t.Errorf("描述应保留: %q", result.Manifest.Description)
	}
}

// TestSanitizeFileBaseName 文件名净化
func TestSanitizeFileBaseName(t *testing.T) {
	cases := map[string]string{
		"正常名字":        "正常名字",
		"a/b":         "a_b",
		`a\b`:         "a_b",
		`a:b*.zip`:    "a_b_.zip",
		`前?后`:         "前_后",
		`名字.`:         "名字",
		"  trimmed  ": "trimmed",
		"..":          "",
		".":           "",
		"":            "",
	}
	for input, want := range cases {
		if got := sanitizeFileBaseName(input); got != want {
			t.Errorf("sanitizeFileBaseName(%q) = %q, 期望 %q", input, got, want)
		}
	}
}
