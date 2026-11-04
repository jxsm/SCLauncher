package scbbs

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// writeJSONFile 写临时 JSON 文件
func writeJSONFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("写入 %s 失败: %v", name, err)
	}
	return path
}

// buildScbbsZip 构造 scbbs ZIP 载体测试包：packinfo.json + 内嵌文件
func buildScbbsZip(t *testing.T, zipPath, manifestContent string, embedded map[string]string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("创建 zip 失败: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	if manifestContent != "" {
		writeZipEntry(t, w, "packinfo.json", manifestContent)
	}
	for name, content := range embedded {
		writeZipEntry(t, w, name, content)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("关闭 zip 失败: %v", err)
	}
}

// writeZipEntry 向 zip 写入一个文件条目
func writeZipEntry(t *testing.T, w *zip.Writer, name, content string) {
	t.Helper()
	fw, err := w.Create(name)
	if err != nil {
		t.Fatalf("写入条目 %q 失败: %v", name, err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("写入条目内容 %q 失败: %v", name, err)
	}
}

// --- JSON 载体 ---

func TestParseJSONCarrierLenient(t *testing.T) {
	dir := t.TempDir()
	// 小写键名 + 行注释 + 块注释 + 尾逗号 + BOM + FileId 写成数字
	manifest := "\uFEFF{\n" +
		"  // scbbs 整合包\n" +
		"  \"name\": \"测试整合包\", /* 名称 */\n" +
		"  \"packageName\": \"test.pack\",\n" +
		"  \"version\": \"1.0.0\",\n" +
		"  \"description\": \"用于单元测试\",\n" +
		"  \"items\": [\n" +
		"    {\"targetDirectory\": \"mods/\", \"url\": \"https://example.com/a.scmod\", \"packageName\": \"com.a\", \"version\": \"1.0\",},\n" +
		"    {\"TargetDirectory\": \"worlds/\", \"Url\": \"https://example.com/b.scworld\"},\n" +
		"    {\"targetDirectory\": \"mymod/config/\", \"fileId\": 12345,},\n" +
		"  ],\n" +
		"}"
	path := writeJSONFile(t, dir, "pack.json", []byte(manifest))

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if result.IsZipCarrier {
		t.Error("应为 JSON 载体")
	}
	m := result.Manifest
	if m.Name != "测试整合包" || m.PackageName != "test.pack" || m.Version != "1.0.0" {
		t.Errorf("基本信息解析错误: %+v", m)
	}
	if len(m.Items) != 3 {
		t.Fatalf("条目数应为 3，实际 %d", len(m.Items))
	}
	if m.Items[0].FileID != "" || m.Items[2].FileID != "12345" {
		t.Errorf("FileId 解析错误: %q / %q", m.Items[0].FileID, m.Items[2].FileID)
	}
	if m.Items[0].ResolveType() != ItemTypeMod ||
		m.Items[1].ResolveType() != ItemTypeWorld ||
		m.Items[2].ResolveType() != ItemTypeData {
		t.Errorf("类型解析错误: %s / %s / %s",
			m.Items[0].ResolveType(), m.Items[1].ResolveType(), m.Items[2].ResolveType())
	}
}

func TestParseJSONCarrierNotScbbs(t *testing.T) {
	dir := t.TempDir()
	// 顶层没有 Items 数组 → 不是 scbbs 格式
	path := writeJSONFile(t, dir, "other.json", []byte(`{"name": "配置文件", "list": [1, 2]}`))

	if _, err := Parse(path); !errors.Is(err, ErrNotScbbsPack) {
		t.Errorf("期望 ErrNotScbbsPack，实际: %v", err)
	}
}

func TestParseJSONCarrierMissingRequired(t *testing.T) {
	dir := t.TempDir()
	// 是 scbbs 格式但缺 Description → 校验错误
	path := writeJSONFile(t, dir, "bad.json", []byte(
		`{"name": "n", "packageName": "p", "version": "1", "items": [{"targetDirectory": "mods/", "url": "https://e.com/a.scmod"}]}`))

	_, err := Parse(path)
	if err == nil || errors.Is(err, ErrNotScbbsPack) {
		t.Errorf("期望校验错误，实际: %v", err)
	}
	if !strings.Contains(err.Error(), "Description") {
		t.Errorf("错误信息应提到缺失字段: %v", err)
	}
}

func TestParseJSONCarrierRejectsRelativeURL(t *testing.T) {
	dir := t.TempDir()
	path := writeJSONFile(t, dir, "rel.json", []byte(
		`{"name": "n", "packageName": "p", "version": "1", "description": "d",
		  "items": [{"targetDirectory": "mods/", "url": "mods/a.scmod"}]}`))

	_, err := Parse(path)
	if err == nil || errors.Is(err, ErrNotScbbsPack) {
		t.Errorf("JSON 载体的相对路径 Url 应被拒绝，实际: %v", err)
	}
}

func TestParseIconValidation(t *testing.T) {
	dir := t.TempDir()
	base := `"name": "n", "packageName": "p", "version": "1", "description": "d",
		"items": [{"targetDirectory": "mods/", "url": "https://e.com/a.scmod"}]`

	// JSON 载体：Icon 必须是 http(s) 链接
	path := writeJSONFile(t, dir, "icon.json", []byte(`{`+base+`, "icon": "icon.png"}`))
	if _, err := Parse(path); err == nil || !strings.Contains(err.Error(), "http") {
		t.Errorf("JSON 载体的包内 Icon 应被拒绝，实际: %v", err)
	}

	// 扩展名不合法
	path2 := writeJSONFile(t, dir, "icon2.json", []byte(`{`+base+`, "icon": "https://e.com/icon.jpg"}`))
	if _, err := Parse(path2); err == nil || !strings.Contains(err.Error(), "png") {
		t.Errorf("非 png/webp 图标应被拒绝，实际: %v", err)
	}
}

func TestParseTargetPathNormalization(t *testing.T) {
	dir := t.TempDir()
	// 旧字段 TargetPath 反推 TargetDirectory 与文件名
	path := writeJSONFile(t, dir, "legacy.json", []byte(
		`{"name": "n", "packageName": "p", "version": "1", "description": "d",
		  "items": [{"targetPath": "mods/OldMod.scmod", "url": "https://e.com/dl"}]}`))

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	item := result.Manifest.Items[0]
	if item.TargetDirectory != "mods" {
		t.Errorf("反推 TargetDirectory = %q, 期望 %q", item.TargetDirectory, "mods")
	}
	// 归一化会把 TargetPath 尾段填入 FileName 字段
	// （InferFileName 仍按优先级取 Url 尾段，与 ScbbsPlus 一致）
	if item.FileName != "OldMod.scmod" {
		t.Errorf("反推 FileName = %q, 期望 %q", item.FileName, "OldMod.scmod")
	}
	if item.ResolveType() != ItemTypeMod {
		t.Errorf("类型应为 mod，实际 %s", item.ResolveType())
	}
}

func TestResolveTypeByCategoryFallback(t *testing.T) {
	cases := map[string]struct {
		category ItemType
		item     Item
	}{
		"mod":    {ItemTypeMod, Item{TargetDirectory: "Mods", Category: "world"}},
		"world":  {ItemTypeWorld, Item{Category: "scworld"}},
		"furn":   {ItemTypeFurniture, Item{Category: "furniture"}},
		"tex":    {ItemTypeTexture, Item{Category: "png"}},
		"skin":   {ItemTypeSkin, Item{Category: "skin"}},
		"data":   {ItemTypeData, Item{Category: "whatever"}},
		"custom": {ItemTypeData, Item{TargetDirectory: "MyData/config"}},
	}
	for name, c := range cases {
		if got := c.item.ResolveType(); got != c.category {
			t.Errorf("%s: ResolveType = %s, 期望 %s", name, got, c.category)
		}
	}
}

// --- ZIP 载体 ---

const zipManifest = `{
	"Name": "打包测试", "PackageName": "zip.pack", "Version": "2.0", "Description": "d",
	"Items": [
		{"TargetDirectory": "mods/", "Url": "mods/测试模组.scmod", "PackageName": "com.test", "Version": "1.0"},
		{"TargetDirectory": "worlds/", "Url": "worlds/我的世界.scworld"},
		{"TargetDirectory": "furniturepacks/", "Url": "packs/furn.scfpack"}
	]
}`

func TestParseZIPCarrier(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pack.zip")
	buildScbbsZip(t, zipPath, zipManifest, map[string]string{
		"mods/测试模组.scmod":     "scmod-bytes",
		"worlds/我的世界.scworld": "world-bytes",
		"packs/furn.scfpack":  "furn-bytes",
		"Icon.png":            "png-bytes",
	})

	result, err := Parse(zipPath)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if !result.IsZipCarrier {
		t.Error("应为 ZIP 载体")
	}
	if len(result.Manifest.Items) != 3 {
		t.Fatalf("条目数应为 3，实际 %d", len(result.Manifest.Items))
	}
	if len(result.MissingEmbedded) != 0 {
		t.Errorf("不应有缺失内嵌项，实际 %d", len(result.MissingEmbedded))
	}
	// Icon 未填 → 探测包内 Icon.png
	if string(result.EmbeddedIconBytes) != "png-bytes" || result.EmbeddedIconExt != "png" {
		t.Errorf("默认图标探测错误: %q .%s", result.EmbeddedIconBytes, result.EmbeddedIconExt)
	}
}

func TestParseZIPCarrierNotScbbs(t *testing.T) {
	dir := t.TempDir()
	// 没有 packinfo.json（模拟 v0.1 scmodpack）
	zipPath := filepath.Join(dir, "v01.zip")
	buildScbbsZip(t, zipPath, "", map[string]string{"manifest.jsonc": "{}"})

	if _, err := Parse(zipPath); !errors.Is(err, ErrNotScbbsPack) {
		t.Errorf("期望 ErrNotScbbsPack，实际: %v", err)
	}
}

func TestParseZIPCarrierGBKEntryNames(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "gbk.zip")

	// 条目名用 GBK 字节写入（中文压缩工具行为），清单 Url 用 UTF-8 引用
	gbkMod, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("mods/中文模组.scmod"))
	if err != nil {
		t.Fatalf("GBK 编码失败: %v", err)
	}
	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d",
		"Items":[{"TargetDirectory":"mods/","Url":"mods/中文模组.scmod"}]}`

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	writeZipEntry(t, w, "packinfo.json", manifest)
	writeZipEntry(t, w, string(gbkMod), "data")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(zipPath)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if len(result.MissingEmbedded) != 0 {
		t.Errorf("GBK 条目名应能被 UTF-8 Url 匹配到，实际缺失 %d 项", len(result.MissingEmbedded))
	}
}

func TestParseZIPCarrierTraversalEntryExcluded(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	writeZipEntry(t, w, "packinfo.json", `{"Name":"n","PackageName":"p","Version":"1","Description":"d",
		"Items":[{"TargetDirectory":"mods/","Url":"mods/a.scmod"}]}`)
	writeZipEntry(t, w, "../evil.txt", "malicious")
	writeZipEntry(t, w, "mods/a.scmod", "ok")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(zipPath)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if len(result.MissingEmbedded) != 0 {
		t.Error("正常条目不应缺失")
	}
}

func TestParseZIPCarrierMissingEmbedded(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "missing.zip")
	// 引用了包内不存在的文件且无 FileId → 记入 MissingEmbedded（不报错）
	buildScbbsZip(t, zipPath, `{"Name":"n","PackageName":"p","Version":"1","Description":"d",
		"Items":[{"TargetDirectory":"mods/","Url":"mods/不存在.scmod"}]}`, nil)

	result, err := Parse(zipPath)
	if err != nil {
		t.Fatalf("缺失内嵌项不应导致解析失败: %v", err)
	}
	if len(result.MissingEmbedded) != 1 {
		t.Fatalf("MissingEmbedded 应为 1，实际 %d", len(result.MissingEmbedded))
	}
}

func TestPreprocessManifestJSON(t *testing.T) {
	input := []byte("\uFEFF{\n// 行注释\n\"a\": \"http://x//y\", /* 块\n注释 */ \"b\": 1,\n\"c\": [1,2,],\n}")
	got := string(preprocessManifestJSON(input))
	// 字符串内的 // 保留；块注释剥离；} / ] 前的尾逗号去除；普通逗号保留
	if !strings.Contains(got, `"http://x//y"`) {
		t.Errorf("字符串内的 // 应保留: %s", got)
	}
	if strings.Contains(got, "块") || strings.Contains(got, "行注释") {
		t.Errorf("注释应被剥离: %s", got)
	}
	if !strings.Contains(got, `[1,2]`) {
		t.Errorf("数组尾逗号应被去除: %s", got)
	}
	if !strings.Contains(got, `"b": 1,`) {
		t.Errorf("普通逗号应保留: %s", got)
	}
}
