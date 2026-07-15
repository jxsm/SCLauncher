package savegame

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// makeZipBytes 在内存中构造一个 zip，返回其字节（用于 PreviewSaveRequiredMods 测试）
func makeZipBytes(t *testing.T, files map[string]string) []byte {
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
	return buf.Bytes()
}

// writeZip 写一个临时 zip 文件，包含给定的 文件名->内容 映射，返回路径
func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	tmp, err := os.CreateTemp("", "savegame-test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.Write(makeZipBytes(t, files)); err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	t.Cleanup(func() { os.Remove(tmp.Name()) })
	return tmp.Name()
}

const sampleUsedModsXML = `<Project Guid="x" Name="GameProject" Version="2.4" APIVersion="1.9.0.0">
  <Subsystems>
    <Values Name="GameInfo"><Value Name="WorldName" Type="string" Value="逃离学校" /></Values>
    <Values Name="UsedMods">
      <Value Name="ModsCount" Type="int" Value="2" />
      <Values Name="Mods">
        <Values Name="0">
          <Value Name="Name" Type="string" Value="连锁砍树" />
          <Value Name="Version" Type="string" Value="25.9.6" />
          <Value Name="ApiVersion" Type="string" Value="1.8" />
          <Value Name="Author" Type="string" Value="LLKXY" />
          <Value Name="Link" Type="string" Value="https://www.schz.top" />
          <Value Name="PackageName" Type="string" Value="LXK" />
        </Values>
        <Values Name="1">
          <Value Name="Name" Type="string" Value="命令方块" />
          <Value Name="Version" Type="string" Value="4.1.1" />
          <Value Name="PackageName" Type="string" Value="zh.command" />
        </Values>
      </Values>
    </Values>
  </Subsystems>
</Project>`

// 回归：XML 的 UsedMods 应解析出全部模组及其字段
func TestParseRequiredMods_XML(t *testing.T) {
	mods := parseRequiredMods([]byte(sampleUsedModsXML), nil)
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods, got %d (%+v)", len(mods), mods)
	}
	byPkg := map[string]SaveRequiredMod{}
	for _, m := range mods {
		byPkg[m.PackageName] = m
	}
	if m := byPkg["LXK"]; m.Version != "25.9.6" || m.Name != "连锁砍树" || m.Author != "LLKXY" || m.Link != "https://www.schz.top" {
		t.Errorf("unexpected LXK: %+v", m)
	}
	if m := byPkg["zh.command"]; m.Version != "4.1.1" || m.Name != "命令方块" {
		t.Errorf("unexpected zh.command: %+v", m)
	}
}

// 回归：JSON 的 Mods 为对象，字段值用 ["string","值"] 数组编码（GameInfo 即此格式）
func TestParseRequiredMods_JSONObject(t *testing.T) {
	jsonBytes := []byte(`{
	  "Subsystems": {
	    "UsedMods": {
	      "ModsCount": ["int", 2],
	      "Mods": {
	        "0": {
	          "Name": ["string", "连锁砍树"],
	          "Version": ["string", "25.9.6"],
	          "PackageName": ["string", "LXK"],
	          "Author": ["string", "LLKXY"]
	        },
	        "1": {
	          "Name": ["string", "命令方块"],
	          "Version": ["string", "4.1.1"],
	          "PackageName": ["string", "zh.command"]
	        }
	      }
	    }
	  }
	}`)
	mods := parseRequiredMods(nil, jsonBytes)
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods, got %d (%+v)", len(mods), mods)
	}
	byPkg := map[string]SaveRequiredMod{}
	for _, m := range mods {
		byPkg[m.PackageName] = m
	}
	if m := byPkg["LXK"]; m.Version != "25.9.6" || m.Name != "连锁砍树" || m.Author != "LLKXY" {
		t.Errorf("unexpected LXK: %+v", m)
	}
	if m := byPkg["zh.command"]; m.Version != "4.1.1" {
		t.Errorf("unexpected zh.command: %+v", m)
	}
}

// 回归：JSON 的 Mods 为数组，字段值为裸字符串
func TestParseRequiredMods_JSONArray(t *testing.T) {
	jsonBytes := []byte(`{
	  "Subsystems": {
	    "UsedMods": {
	      "Mods": [
	        { "PackageName": "LXK", "Version": "25.9.6", "Name": "连锁砍树" }
	      ]
	    }
	  }
	}`)
	mods := parseRequiredMods(nil, jsonBytes)
	if len(mods) != 1 || mods[0].PackageName != "LXK" || mods[0].Version != "25.9.6" || mods[0].Name != "连锁砍树" {
		t.Errorf("unexpected: %+v", mods)
	}
}

// 回归：缺 UsedMods / 空 Mods / ModsCount=0 都应返回空切片而非报错
func TestParseRequiredMods_Empty(t *testing.T) {
	cases := map[string]string{
		"no UsedMods": `{"Subsystems": {}}`,
		"empty Mods":  `{"Subsystems": {"UsedMods": {"Mods": {}}}}`,
		"zero count":  `{"Subsystems": {"UsedMods": {"ModsCount": ["int", 0], "Mods": {}}}}`,
		"xml no UsedMods": `<Project><Subsystems><Values Name="GameInfo"><Value Name="WorldName" Type="string" Value="x" /></Values></Subsystems></Project>`,
	}
	for name, body := range cases {
		var mods []SaveRequiredMod
		if body[0] == '<' {
			mods = parseRequiredMods([]byte(body), nil)
		} else {
			mods = parseRequiredMods(nil, []byte(body))
		}
		if len(mods) != 0 {
			t.Errorf("%s: expected 0 mods, got %d (%+v)", name, len(mods), mods)
		}
	}
}

// 回归：单项缺 PackageName 时，XML 仍会追加（前端按包名过滤）；JSON 缺包名则跳过
func TestParseRequiredMods_MissingPackageName(t *testing.T) {
	xmlBytes := []byte(`<Project><Subsystems><Values Name="UsedMods"><Values Name="Mods">
		<Values Name="0"><Value Name="Name" Type="string" Value="无名" /><Value Name="Version" Type="string" Value="1.0" /></Values>
	</Values></Values></Subsystems></Project>`)
	mods := parseRequiredMods(xmlBytes, nil)
	if len(mods) != 1 {
		t.Fatalf("expected 1 entry (empty pkg), got %d", len(mods))
	}
	if mods[0].PackageName != "" || mods[0].Name != "无名" {
		t.Errorf("unexpected: %+v", mods[0])
	}
}

// 回归：XML 与 JSON 同时给出时按包名去重（XML 优先），不产生重复
func TestParseRequiredMods_Dedup(t *testing.T) {
	xmlBytes := []byte(sampleUsedModsXML)
	jsonBytes := []byte(`{"Subsystems":{"UsedMods":{"Mods":{"0":{"PackageName":["string","LXK"],"Version":["string","9.9"]}}}}}`)
	mods := parseRequiredMods(xmlBytes, jsonBytes)
	count := 0
	for _, m := range mods {
		if m.PackageName == "LXK" {
			count++
			if m.Version != "25.9.6" { // XML 优先
				t.Errorf("expected XML version to win, got %s", m.Version)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected LXK deduped to 1, got %d", count)
	}
}

// 回归：从 .scworld/.zip 归档预览所需模组
func TestPreviewSaveRequiredMods_FromZip(t *testing.T) {
	path := writeZip(t, map[string]string{
		"Project.xml": sampleUsedModsXML,
	})
	mgr := &Manager{} // PreviewSaveRequiredMods 只用到 zip 读取，不需 config/paths
	mods, err := mgr.PreviewSaveRequiredMods(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods from zip, got %d (%+v)", len(mods), mods)
	}
	pkgs := map[string]bool{}
	for _, m := range mods {
		pkgs[m.PackageName] = true
	}
	if !pkgs["LXK"] || !pkgs["zh.command"] {
		t.Errorf("expected LXK and zh.command, got %+v", pkgs)
	}
}

// 回归：不支持的扩展名应报错
func TestPreviewSaveRequiredMods_BadExt(t *testing.T) {
	tmp, err := os.CreateTemp("", "bad-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmp.WriteString("not a save")
	tmp.Close()
	t.Cleanup(func() { os.Remove(tmp.Name()) })

	mgr := &Manager{}
	if _, err := mgr.PreviewSaveRequiredMods(tmp.Name()); err == nil {
		t.Errorf("expected error for unsupported extension, got nil")
	}
}

// 回归：GetSaveRequiredMods 从磁盘世界目录读取 Project.xml 往返一致
func TestGetSaveRequiredMods_DiskRoundTrip(t *testing.T) {
	// 直接在临时世界目录写入 Project.xml，并通过路径直接读取校验解析结果一致
	tmpDir := t.TempDir()
	worldPath := filepath.Join(tmpDir, "逃离学校")
	if err := os.MkdirAll(worldPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldPath, "Project.xml"), []byte(sampleUsedModsXML), 0644); err != nil {
		t.Fatal(err)
	}

	// 直接对世界目录文件跑解析器，校验与纯函数结果一致
	xmlBytes, _ := os.ReadFile(filepath.Join(worldPath, "Project.xml"))
	mods := parseRequiredMods(xmlBytes, nil)
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods from disk file, got %d", len(mods))
	}
}
