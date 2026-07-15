package mod

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeZip 写一个临时 zip 文件，包含给定的 文件名->内容 映射，返回路径
func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	tmp, err := os.CreateTemp("", "modinfo-test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	t.Cleanup(func() { os.Remove(tmp.Name()) })
	return tmp.Name()
}

func TestParseModInfo_ObjectMapDeps(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "示例模组",
			"Version": "1.0",
			"ApiVersion": "1.9.1",
			"PackageName": "xfdz.Template",
			"Dependencies": { "scbbsplus": "1.0", "othermod": "[1.0,2.0)" }
		}`,
		"data.dll": "binary",
	})
	info := parseModInfoFromModFile(path)
	if info == nil {
		t.Fatal("expected non-nil ModInfo")
	}
	if info.Name != "示例模组" || info.PackageName != "xfdz.Template" || info.ApiVersion != "1.9.1" {
		t.Errorf("unexpected basic fields: %+v", info)
	}
	if len(info.Dependencies) != 2 {
		t.Fatalf("expected 2 deps, got %d (%+v)", len(info.Dependencies), info.Dependencies)
	}
	got := map[string]string{}
	for _, d := range info.Dependencies {
		got[d.PackageName] = d.VersionRange
	}
	if got["scbbsplus"] != "1.0" || got["othermod"] != "[1.0,2.0)" {
		t.Errorf("unexpected deps: %+v", got)
	}
}

func TestParseModInfo_StringArrayDeps(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.8", "PackageName": "x",
			"Dependencies": ["pkgA:1.2", "pkgB"]
		}`,
	})
	info := parseModInfoFromModFile(path)
	if len(info.Dependencies) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(info.Dependencies))
	}
	got := map[string]string{}
	for _, d := range info.Dependencies {
		got[d.PackageName] = d.VersionRange
	}
	if got["pkgA"] != "1.2" || got["pkgB"] != "" {
		t.Errorf("unexpected deps: %+v", got)
	}
}

func TestParseModInfo_ObjectArrayDeps(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.9.1", "PackageName": "x",
			"Dependencies": [ { "PackageName": "dep", "Version": ">=2.0" } ]
		}`,
	})
	info := parseModInfoFromModFile(path)
	if len(info.Dependencies) != 1 || info.Dependencies[0].PackageName != "dep" || info.Dependencies[0].VersionRange != ">=2.0" {
		t.Errorf("unexpected deps: %+v", info.Dependencies)
	}
}

// 回归：单个依赖写成裸对象（非数组）不应被误当作 pkg->version 映射
func TestParseModInfo_SingleDependencyObject(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.9.1", "PackageName": "x",
			"Dependencies": { "PackageName": "dep", "Version": ">=2.0" }
		}`,
	})
	info := parseModInfoFromModFile(path)
	if len(info.Dependencies) != 1 || info.Dependencies[0].PackageName != "dep" || info.Dependencies[0].VersionRange != ">=2.0" {
		t.Errorf("expected single dep {dep, >=2.0}, got %+v", info.Dependencies)
	}
}

// 回归：数字版本值（对象映射）应被保留为字符串
func TestParseModInfo_NumericVersionDep(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.9.1", "PackageName": "x",
			"Dependencies": { "pkgA": 1.0 }
		}`,
	})
	info := parseModInfoFromModFile(path)
	if len(info.Dependencies) != 1 || info.Dependencies[0].PackageName != "pkgA" {
		t.Errorf("expected pkgA dep, got %+v", info.Dependencies)
	}
	if info.Dependencies[0].VersionRange == "" {
		t.Errorf("expected non-empty version for numeric value, got %+v", info.Dependencies)
	}
}

// 回归：裸字符串依赖列表中包含括号范围（带逗号）不应被错误切分
func TestParseModInfo_BareStringWithRangeComma(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.9.1", "PackageName": "x",
			"Dependencies": "depA:[1.0,2.0),depB:1.0"
		}`,
	})
	info := parseModInfoFromModFile(path)
	if len(info.Dependencies) != 2 {
		t.Fatalf("expected 2 deps, got %d (%+v)", len(info.Dependencies), info.Dependencies)
	}
	got := map[string]string{}
	for _, d := range info.Dependencies {
		got[d.PackageName] = d.VersionRange
	}
	if got["depA"] != "[1.0,2.0)" || got["depB"] != "1.0" {
		t.Errorf("range with comma corrupted: %+v", got)
	}
}

// 回归：字符串编码的标量字段（LoadOrder/NonPersistentMod）应正确解析
func TestParseModInfo_StringEncodedScalars(t *testing.T) {
	path := writeZip(t, map[string]string{
		"modinfo.json": `{
			"Name": "X", "Version": "1", "ApiVersion": "1.9.1", "PackageName": "x",
			"LoadOrder": "10000", "NonPersistentMod": "true"
		}`,
	})
	info := parseModInfoFromModFile(path)
	if info.LoadOrder != 10000 {
		t.Errorf("expected LoadOrder 10000, got %d", info.LoadOrder)
	}
	if !info.NonPersistentMod {
		t.Errorf("expected NonPersistentMod true, got %v", info.NonPersistentMod)
	}
}

func TestParseModInfo_EmptyDeps(t *testing.T) {
	for _, raw := range []string{`{}`, `[]`} {
		path := writeZip(t, map[string]string{
			"modinfo.json": `{"Name":"X","Version":"1","ApiVersion":"1.9.1","PackageName":"x","Dependencies":` + raw + `}`,
		})
		info := parseModInfoFromModFile(path)
		if info == nil || info.Dependencies == nil || len(info.Dependencies) != 0 {
			t.Errorf("expected empty non-nil deps for %s, got %+v", raw, info)
		}
	}
}

func TestParseModInfo_NestedAndCaseInsensitive(t *testing.T) {
	// modinfo.json 嵌套一层 + 字段名大小写混合
	path := writeZip(t, map[string]string{
		"MyMod/modinfo.json": `{
			"name": "Nested", "version": "0.2", "apiversion": "1.7.0", "packagename": "nested",
			"gameplayimpactlevel": "Break", "loadorder": 10000, "nonpersistentmod": true
		}`,
	})
	info := parseModInfoFromModFile(path)
	if info == nil {
		t.Fatal("expected non-nil")
	}
	if info.Name != "Nested" || info.ApiVersion != "1.7.0" || info.GameplayImpactLevel != "Break" {
		t.Errorf("unexpected: %+v", info)
	}
	if info.LoadOrder != 10000 || !info.NonPersistentMod {
		t.Errorf("unexpected numeric/bool: %+v", info)
	}
}

func TestParseModInfo_CorruptAndMissing(t *testing.T) {
	// 非 zip 文件
	tmp, _ := os.CreateTemp("", "notzip-*.bin")
	tmp.WriteString("this is not a zip")
	tmp.Close()
	t.Cleanup(func() { os.Remove(tmp.Name()) })
	if info := parseModInfoFromModFile(tmp.Name()); info != nil {
		t.Errorf("expected nil for non-zip, got %+v", info)
	}

	// zip 但无 modinfo.json
	path := writeZip(t, map[string]string{"readme.txt": "hello"})
	if info := parseModInfoFromModFile(path); info != nil {
		t.Errorf("expected nil when modinfo.json absent, got %+v", info)
	}

	// 非法 JSON
	path2 := writeZip(t, map[string]string{"modinfo.json": "{ not json"})
	if info := parseModInfoFromModFile(path2); info != nil {
		t.Errorf("expected nil for bad json, got %+v", info)
	}
}

func TestParseModInfo_RealSampleIfPresent(t *testing.T) {
	// 如果存在真实样本则解析它，否则跳过
	sample := filepath.Join("..", "..", "build", "bin", ".Survivalcraft", "versions",
		"api-2.4-1.9.2.1-1784108996755677100", "mods", "[1.9.1.3] 原始工艺 v0.1.7.scmod")
	if _, err := os.Stat(sample); err != nil {
		t.Skip("sample mod not present")
	}
	info := parseModInfoFromModFile(sample)
	if info == nil {
		t.Fatal("expected non-nil for real sample")
	}
	if info.Name != "原始工艺" || info.PackageName != "old make" || info.ApiVersion != "1.9.2.1" {
		t.Errorf("unexpected real sample info: %+v", info)
	}
}
