package scbbs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SCLauncher/backend/config"
	"SCLauncher/backend/mod"
	"SCLauncher/backend/scbbsapi"
)

// newTestInstaller 创建指向临时版本目录的安装器
func newTestInstaller(t *testing.T) (*Installer, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{VersionsDir: dir}
	versionID := "test-version"
	if err := os.MkdirAll(filepath.Join(dir, versionID), 0755); err != nil {
		t.Fatal(err)
	}
	return NewInstaller(cfg, mod.NewManager(cfg)), filepath.Join(dir, versionID)
}

// makeScmodBytes 构造一个带 modinfo.json 的 .scmod（本质 zip）
func makeScmodBytes(t *testing.T, packageName, version string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fw, err := w.Create("modinfo.json")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(fw, `{"Name":"模组","Version":%q,"PackageName":%q}`, version, packageName)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// makeWorldBytes 构造一个 .scworld（世界文件夹内容的 zip）
func makeWorldBytes(t *testing.T) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range map[string]string{
		"Project.json":   `{"Subsystems":{"GameInfo":{"WorldName":"我的世界"}}}`,
		"Chunks/0.chunk": "chunk-data",
	} {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// buildPackZip 构造 scbbs ZIP 载体：packinfo.json + 内嵌文件（字节值）
func buildPackZip(t *testing.T, zipPath, manifestContent string, embedded map[string][]byte) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	fw, err := w.Create("packinfo.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(manifestContent)); err != nil {
		t.Fatal(err)
	}
	for name, content := range embedded {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// parseOrFatal 解析并断言成功
func parseOrFatal(t *testing.T, path string) *ParseResult {
	t.Helper()
	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	return result
}

func TestInstallEmbeddedItems(t *testing.T) {
	installer, versionPath := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	manifest := `{"Name":"完整包","PackageName":"full.pack","Version":"1.0","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/测试模组.scmod","PackageName":"com.test","Version":"1.0"},
		{"TargetDirectory":"worlds/","Url":"worlds/我的世界.scworld"},
		{"TargetDirectory":"furniturepacks/","Url":"packs/furn.scfpack"},
		{"TargetDirectory":"Foo/","Url":"config.json","FileName":"config.json"}
	]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/测试模组.scmod":     makeScmodBytes(t, "com.test", "1.0"),
		"worlds/我的世界.scworld": makeWorldBytes(t),
		"packs/furn.scfpack":  []byte("furn-bytes"),
		"config.json":         []byte("{\"k\":1}"),
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v (failures: %v)", err, summary.Failures)
	}
	if summary.Succeeded != 4 || summary.Failed != 0 || summary.Skipped != 0 {
		t.Fatalf("汇总不符: %+v", summary)
	}

	// 模组 → Mods 目录
	modData, err := os.ReadFile(filepath.Join(versionPath, "Mods", "测试模组.scmod"))
	if err != nil || len(modData) == 0 {
		t.Errorf("模组未落地: %v", err)
	}
	// 存档 → Worlds/<去扩展名>/ 文件夹（zip 内容解压）
	if _, err := os.Stat(filepath.Join(versionPath, "Worlds", "我的世界", "Project.json")); err != nil {
		t.Errorf("世界文件未落地: %v", err)
	}
	if _, err := os.Stat(filepath.Join(versionPath, "Worlds", "我的世界", "Chunks", "0.chunk")); err != nil {
		t.Errorf("世界子目录文件未落地: %v", err)
	}
	// 家具包 → FurniturePacks
	if _, err := os.Stat(filepath.Join(versionPath, "FurniturePacks", "furn.scfpack")); err != nil {
		t.Errorf("家具包未落地: %v", err)
	}
	// 自定义数据文件 → Foo/config.json
	data, err := os.ReadFile(filepath.Join(versionPath, "Foo", "config.json"))
	if err != nil || string(data) != `{"k":1}` {
		t.Errorf("自定义文件未落地: %v", err)
	}
}

func TestInstallConflictSkipAndOverwrite(t *testing.T) {
	for _, tc := range []struct {
		name       string
		action     string
		wantSkip   bool
		wantChange bool
	}{
		{"跳过", ConflictSkip, true, false},
		{"覆盖", ConflictOverwrite, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installer, versionPath := newTestInstaller(t)
			packDir := t.TempDir()
			packPath := filepath.Join(packDir, "pack.zip")

			manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
				{"TargetDirectory":"mods/","Url":"mods/a.scmod"}]}`
			buildPackZip(t, packPath, manifest, map[string][]byte{
				"mods/a.scmod": makeScmodBytes(t, "", ""),
			})

			// 预置同名文件
			modsDir := filepath.Join(versionPath, "Mods")
			os.MkdirAll(modsDir, 0755)
			existing := filepath.Join(modsDir, "a.scmod")
			os.WriteFile(existing, []byte("old"), 0644)

			asked := 0
			installer.SetConflictAskFn(func(fileName, itemName string) (string, error) {
				asked++
				return tc.action, nil
			})

			summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
			if err != nil {
				t.Fatalf("Install 失败: %v", err)
			}
			if asked != 1 {
				t.Errorf("冲突询问次数 = %d, 期望 1", asked)
			}
			if tc.wantSkip && summary.Skipped != 1 {
				t.Errorf("期望跳过 1 项: %+v", summary)
			}
			got, _ := os.ReadFile(existing)
			if tc.wantChange && string(got) == "old" {
				t.Error("期望被覆盖")
			}
			if !tc.wantChange && string(got) != "old" {
				t.Error("期望保留原文件")
			}
		})
	}
}

func TestInstallConflictOverwriteAllRemembered(t *testing.T) {
	installer, versionPath := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/a.scmod"},
		{"TargetDirectory":"worlds/","Url":"worlds/w.scworld"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/a.scmod":     makeScmodBytes(t, "", ""),
		"worlds/w.scworld": makeWorldBytes(t),
	})

	// 预置两个同名目标
	os.MkdirAll(filepath.Join(versionPath, "Mods"), 0755)
	os.WriteFile(filepath.Join(versionPath, "Mods", "a.scmod"), []byte("old"), 0644)
	os.MkdirAll(filepath.Join(versionPath, "Worlds", "w"), 0755)
	os.WriteFile(filepath.Join(versionPath, "Worlds", "w", "old.txt"), []byte("old"), 0644)

	asked := 0
	installer.SetConflictAskFn(func(fileName, itemName string) (string, error) {
		asked++
		return ConflictOverwriteAll, nil
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v", err)
	}
	if asked != 1 {
		t.Errorf("全部覆盖后不应重复询问，实际询问 %d 次", asked)
	}
	if summary.Skipped != 0 || summary.Succeeded != 2 {
		t.Errorf("汇总不符: %+v", summary)
	}
	// 世界文件夹被删除重建
	if _, err := os.Stat(filepath.Join(versionPath, "Worlds", "w", "old.txt")); err == nil {
		t.Error("覆盖世界时旧文件应被删除")
	}
	if _, err := os.Stat(filepath.Join(versionPath, "Worlds", "w", "Project.json")); err != nil {
		t.Errorf("新世界文件未落地: %v", err)
	}
}

func TestInstallConflictCancel(t *testing.T) {
	installer, versionPath := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/a.scmod"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/a.scmod": makeScmodBytes(t, "", ""),
	})

	os.MkdirAll(filepath.Join(versionPath, "Mods"), 0755)
	existing := filepath.Join(versionPath, "Mods", "a.scmod")
	os.WriteFile(existing, []byte("old"), 0644)

	installer.SetConflictAskFn(func(fileName, itemName string) (string, error) {
		return ConflictCancel, nil
	})

	_, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err == nil || !strings.Contains(err.Error(), "取消") {
		t.Errorf("期望取消错误，实际: %v", err)
	}
	got, _ := os.ReadFile(existing)
	if string(got) != "old" {
		t.Error("取消后原文件不应被改动")
	}
}

func TestInstallModPackageMismatch(t *testing.T) {
	installer, _ := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	// 清单声明 com.other，实际 scmod 是 com.test → 该项失败
	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/a.scmod","PackageName":"com.other","Version":"9.9"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/a.scmod": makeScmodBytes(t, "com.test", "1.0"),
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err == nil {
		t.Fatal("全部条目失败时应返回错误")
	}
	if summary.Failed != 1 || len(summary.Failures) != 1 {
		t.Fatalf("汇总不符: %+v", summary)
	}
	if !strings.Contains(summary.Failures[0], "包名不匹配") {
		t.Errorf("失败原因应包含包名不匹配: %s", summary.Failures[0])
	}
}

func TestInstallDataBlacklist(t *testing.T) {
	installer, versionPath := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"settings/hack.json","Url":"h.json"},
		{"TargetDirectory":"Good/","Url":"g.json","FileName":"g.json"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"h.json": []byte("bad"),
		"g.json": []byte("good"),
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v", err)
	}
	if summary.Failed != 1 || summary.Succeeded != 1 {
		t.Fatalf("汇总不符: %+v", summary)
	}
	if !strings.Contains(summary.Failures[0], "受保护") {
		t.Errorf("失败原因应提到受保护目录: %s", summary.Failures[0])
	}
	if _, err := os.Stat(filepath.Join(versionPath, "settings", "hack.json")); err == nil {
		t.Error("黑名单路径不应被写入")
	}
	if _, err := os.Stat(filepath.Join(versionPath, "Good", "g.json")); err != nil {
		t.Errorf("正常自定义文件未落地: %v", err)
	}
}

func TestInstallHTTPDownload(t *testing.T) {
	installer, versionPath := newTestInstaller(t)

	// http 直链下载家具包
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "downloaded-furn")
	}))
	defer srv.Close()

	packDir := t.TempDir()
	jsonPath := filepath.Join(packDir, "net.json")
	manifest := fmt.Sprintf(`{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"furniturepacks/","Url":"%s/furn"}]}`, srv.URL)
	if err := os.WriteFile(jsonPath, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := installer.Install(parseOrFatal(t, jsonPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v (%v)", err, summary.Failures)
	}
	// Url 尾段无扩展名 → 按类型补 .scfpack
	data, err := os.ReadFile(filepath.Join(versionPath, "FurniturePacks", "furn.scfpack"))
	if err != nil || string(data) != "downloaded-furn" {
		t.Errorf("下载文件未落地或内容不符: %v", err)
	}
}

func TestInstallFileIdLookup(t *testing.T) {
	installer, versionPath := newTestInstaller(t)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upload/info" {
			// {code,data} 包装 + PascalCase 字段（两种容错同时覆盖）
			fmt.Fprint(w, `{"code":200,"msg":"ok","data":{"Id":77,"Filename":"社区家具.scfpack","Url":"`+srv.URL+`/files/community.scfpack","PackageName":"","PackageVersion":""}}`)
			return
		}
		fmt.Fprint(w, "community-furn-bytes")
	}))
	defer srv.Close()
	installer.SetAPIClient(scbbsapi.NewSCBBSApiClientWithBaseURL(srv.URL))

	packDir := t.TempDir()
	jsonPath := filepath.Join(packDir, "community.json")
	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"furniturepacks/","FileId":77}]}`
	if err := os.WriteFile(jsonPath, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := installer.Install(parseOrFatal(t, jsonPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v (%v)", err, summary.Failures)
	}
	// 文件名取社区元数据的 Filename
	if _, err := os.Stat(filepath.Join(versionPath, "FurniturePacks", "社区家具.scfpack")); err != nil {
		t.Errorf("社区文件未落地（应使用元数据文件名）: %v", err)
	}
}

func TestInstallNetModsForOnlineVersion(t *testing.T) {
	installer, versionPath := newTestInstaller(t)

	// 联机版标记：NetMods 目录存在
	if err := os.MkdirAll(filepath.Join(versionPath, "NetMods"), 0755); err != nil {
		t.Fatal(err)
	}

	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")
	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/netmod"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/netmod": makeScmodBytes(t, "", ""),
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v (%v)", err, summary.Failures)
	}
	// 联机版 → NetMods 目录 + .netmod 扩展名
	if _, err := os.Stat(filepath.Join(versionPath, "NetMods", "netmod.netmod")); err != nil {
		t.Errorf("联机版模组未落地到 NetMods（含 .netmod 扩展名）: %v", err)
	}
	if _, err := os.Stat(filepath.Join(versionPath, "Mods", "netmod.netmod")); err == nil {
		t.Error("联机版不应落到 Mods 目录")
	}
}

func TestInstallDisabledItemsExcluded(t *testing.T) {
	installer, versionPath := newTestInstaller(t)
	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")

	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"mods/","Url":"mods/a.scmod"},
		{"TargetDirectory":"mods/","Url":"mods/b.scmod","IsEnabled":false}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"mods/a.scmod": makeScmodBytes(t, "", ""),
		"mods/b.scmod": makeScmodBytes(t, "", ""),
	})

	summary, err := installer.Install(parseOrFatal(t, packPath), "test-version")
	if err != nil {
		t.Fatalf("Install 失败: %v", err)
	}
	if summary.Total != 1 || summary.Succeeded != 1 {
		t.Fatalf("禁用项不应参与安装: %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(versionPath, "Mods", "b.scmod")); err == nil {
		t.Error("禁用项不应被安装")
	}
}

func TestInstallWorldUsesExistingDocWorldsDir(t *testing.T) {
	installer, versionPath := newTestInstaller(t)

	// 预置 doc/Worlds 目录 → 存档应落到 doc 下
	docWorlds := filepath.Join(versionPath, "doc", "Worlds")
	if err := os.MkdirAll(docWorlds, 0755); err != nil {
		t.Fatal(err)
	}

	packDir := t.TempDir()
	packPath := filepath.Join(packDir, "pack.zip")
	manifest := `{"Name":"n","PackageName":"p","Version":"1","Description":"d","Items":[
		{"TargetDirectory":"worlds/","Url":"worlds/w.scworld"}]}`
	buildPackZip(t, packPath, manifest, map[string][]byte{
		"worlds/w.scworld": makeWorldBytes(t),
	})

	if _, err := installer.Install(parseOrFatal(t, packPath), "test-version"); err != nil {
		t.Fatalf("Install 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docWorlds, "w", "Project.json")); err != nil {
		t.Errorf("存档应落到已存在的 doc/Worlds: %v", err)
	}
}
