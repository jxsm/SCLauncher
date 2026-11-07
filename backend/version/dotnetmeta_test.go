package version

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// apiVersionDir 返回本地测试用的真实版本目录（不存在则跳过测试）
func apiVersionDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "build", "bin", ".Survivalcraft", "versions", name)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("本地测试版本目录不存在: %s", dir)
	}
	return dir
}

func TestDetectLocalVersionAPI195(t *testing.T) {
	dir := apiVersionDir(t, "api-2.4-api-1.9.5.1-1793865131607975900")
	info, err := DetectLocalVersion(dir)
	if err != nil {
		t.Fatalf("DetectLocalVersion 失败: %v", err)
	}
	if info.GameVersion != "2.4" {
		t.Errorf("游戏版本 = %q, 期望 2.4 (full=%q)", info.GameVersion, info.GameVersionFull)
	}
	if info.GameVersionFull != "2.4.0.0" {
		t.Errorf("完整版本 = %q, 期望 2.4.0.0", info.GameVersionFull)
	}
	if !info.IsAPI || info.SubVersion != "1.9.5.1" {
		t.Errorf("API 版本 = %q (IsAPI=%v), 期望 1.9.5.1 (IsAPI=true)", info.SubVersion, info.IsAPI)
	}
}

func TestDetectLocalVersionAPI18(t *testing.T) {
	dir := apiVersionDir(t, "api-2.4-api-1.8-1793977442451387200")
	info, err := DetectLocalVersion(dir)
	if err != nil {
		t.Fatalf("DetectLocalVersion 失败: %v", err)
	}
	if info.GameVersion != "2.4" {
		t.Errorf("游戏版本 = %q, 期望 2.4", info.GameVersion)
	}
	// 1.8 版 DLL 内字符串写法是 "1.80"，原样保留
	if !info.IsAPI || info.SubVersion != "1.80" {
		t.Errorf("API 版本 = %q (IsAPI=%v), 期望 1.80 (IsAPI=true)", info.SubVersion, info.IsAPI)
	}
}

// TestDetectLocalVersionVanillaExe 只有 exe 的原版（2.3-）：走 PE 版本资源，无 API 版本
func TestDetectLocalVersionVanillaExe(t *testing.T) {
	dir := t.TempDir()
	// 构造含 VS_FIXEDFILEINFO 的最小 exe（版本 2.3.0.7）
	buf := make([]byte, 64)
	copy(buf, vsSignature)
	binary.LittleEndian.PutUint32(buf[8:], (2<<16)|3)  // FileVersionMS = 2.3
	binary.LittleEndian.PutUint32(buf[12:], (0<<16)|7) // FileVersionLS = 0.7
	if err := os.WriteFile(filepath.Join(dir, "Survivalcraft.exe"), buf, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectLocalVersion(dir)
	if err != nil {
		t.Fatalf("DetectLocalVersion 失败: %v", err)
	}
	if info.GameVersion != "2.3" || info.GameVersionFull != "2.3.0.7" {
		t.Errorf("游戏版本 = %q (full=%q), 期望 2.3 (2.3.0.7)", info.GameVersion, info.GameVersionFull)
	}
	if info.IsAPI || info.SubVersion != "" {
		t.Errorf("原版不应有 API 版本, 得到 %q (IsAPI=%v)", info.SubVersion, info.IsAPI)
	}
}

// TestDetectLocalVersionDotnetWithoutModsManager .NET 程序集名不是 Survivalcraft（如 Engine.dll）
// 不应被当作游戏本体
func TestDetectLocalVersionDotnetWithoutModsManager(t *testing.T) {
	src := filepath.Join(apiVersionDir(t, "api-2.4-api-1.9.5.1-1793865131607975900"), "Engine.dll")
	dir := t.TempDir()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Survivalcraft.dll"), data, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := DetectLocalVersion(dir)
	if err != nil {
		t.Fatalf("DetectLocalVersion 失败: %v", err)
	}
	// Engine.dll 的 PE 版本资源是 1.0.0.0，且不含 ModsManager → 不是插件版
	if info.IsAPI {
		t.Errorf("Engine.dll 不应被识别为插件版 (SubVersion=%q)", info.SubVersion)
	}
	if info.GameVersion != "1.0" {
		t.Errorf("游戏版本 = %q, 期望 1.0（来自 PE 资源回退）", info.GameVersion)
	}
}

func TestParseVersionTuple(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"1.9.5.1", []int{1, 9, 5, 1}},
		{"1.80", []int{1, 80}},
		{"2.4", []int{2, 4}},
		{"1", nil},
		{"x.y", nil},
		{"", nil},
		{"1.9.5.1a", nil},
	}
	for _, c := range cases {
		got := parseVersionTuple(c.in)
		if len(got) != len(c.want) {
			t.Errorf("parseVersionTuple(%q) = %v, 期望 %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseVersionTuple(%q) = %v, 期望 %v", c.in, got, c.want)
				break
			}
		}
	}
}

func TestMajorVersionOf(t *testing.T) {
	cases := map[string]string{
		"2.4.0.0": "2.4",
		"2.3.0.7": "2.3",
		"2.31":    "2.31",
		"3":       "3",
	}
	for in, want := range cases {
		if got := majorVersionOf(in); got != want {
			t.Errorf("majorVersionOf(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
