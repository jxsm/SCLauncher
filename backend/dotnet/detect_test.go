package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTfmToMajor(t *testing.T) {
	cases := []struct {
		tfm  string
		want int
		ok   bool
	}{
		{"net10.0", 10, true},
		{"net9.0", 9, true},
		{"net8.0", 8, true},
		{"net6.0", 6, true},
		{"net48", 0, false}, // .NET Framework：无小数点形式
		{"netcoreapp3.1", 0, false},
		{"", 0, false},
		{"net", 0, false},
		{"net.", 0, false},
		{"net0.0", 0, false}, // 主版本 0
		{"net5", 0, false},   // 无小数点
		{"netabc.0", 0, false},
	}
	for _, c := range cases {
		got, ok := tfmToMajor(c.tfm)
		if ok != c.ok || got != c.want {
			t.Errorf("tfmToMajor(%q) = (%d,%v), want (%d,%v)", c.tfm, got, ok, c.want, c.ok)
		}
	}
}

func TestParseRuntimeTargetName(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{".NETCoreApp,Version=v10.0/win-x64", 10, true},
		{".NETCoreApp,Version=v9.0", 9, true},
		{"v10.0", 10, true},
		{"no version here", 0, false},
		{"v0.1", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := parseRuntimeTargetName(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseRuntimeTargetName(%q) = (%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseRuntimeConfig(t *testing.T) {
	t.Run("framework dependent", func(t *testing.T) {
		data := []byte(`{"runtimeOptions":{"tfm":"net10.0","frameworks":[{"name":"Microsoft.WindowsDesktop.App","version":"10.0.0"}]}}`)
		tfm, dep, err := parseRuntimeConfig(data)
		if err != nil || tfm != "net10.0" || !dep {
			t.Fatalf("got tfm=%q dep=%v err=%v", tfm, dep, err)
		}
	})
	t.Run("self contained", func(t *testing.T) {
		data := []byte(`{"runtimeOptions":{"tfm":"net10.0"}}`)
		tfm, dep, err := parseRuntimeConfig(data)
		if err != nil || tfm != "net10.0" || dep {
			t.Fatalf("got tfm=%q dep=%v err=%v", tfm, dep, err)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		if _, _, err := parseRuntimeConfig([]byte(`{bad`)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParseDepsRuntimeTarget(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		data := []byte(`{"runtimeTarget":{"name":".NETCoreApp,Version=v9.0/win-x64"}}`)
		m, ok := parseDepsRuntimeTarget(data)
		if !ok || m != 9 {
			t.Fatalf("got (%d,%v)", m, ok)
		}
	})
	t.Run("no version", func(t *testing.T) {
		_, ok := parseDepsRuntimeTarget([]byte(`{"runtimeTarget":{"name":"nothing"}}`))
		if ok {
			t.Fatal("expected not ok")
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		if _, ok := parseDepsRuntimeTarget([]byte(`{bad`)); ok {
			t.Fatal("expected not ok")
		}
	})
}

func TestDetectRequired(t *testing.T) {
	t.Run("runtimeconfig framework dependent", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.runtimeconfig.json",
			`{"runtimeOptions":{"tfm":"net10.0","frameworks":[{"name":"Microsoft.WindowsDesktop.App","version":"10.0.0"}]}}`)
		r, err := DetectRequired(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !r.Needed || r.MajorVersion != 10 || r.Source != SourceRuntimeConfig || r.TFM != "net10.0" {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("self contained", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.runtimeconfig.json", `{"runtimeOptions":{"tfm":"net10.0"}}`)
		r, _ := DetectRequired(dir)
		if r.Needed || r.Source != SourceSelfContained || r.TFM != "net10.0" {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("deps fallback", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.deps.json", `{"runtimeTarget":{"name":".NETCoreApp,Version=v9.0/win-x64"}}`)
		r, _ := DetectRequired(dir)
		if !r.Needed || r.MajorVersion != 9 || r.Source != SourceDeps {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("runtimeconfig invalid -> deps fallback", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.runtimeconfig.json", `{bad json`)
		writeFile(t, dir, "Game.deps.json", `{"runtimeTarget":{"name":".NETCoreApp,Version=v8.0"}}`)
		r, _ := DetectRequired(dir)
		if !r.Needed || r.MajorVersion != 8 || r.Source != SourceDeps {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("runtimeconfig non-net tfm -> deps fallback", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.runtimeconfig.json",
			`{"runtimeOptions":{"tfm":"netcoreapp3.1","frameworks":[{"name":"Microsoft.NETCore.App"}]}}`)
		writeFile(t, dir, "Game.deps.json", `{"runtimeTarget":{"name":".NETCoreApp,Version=v8.0"}}`)
		r, _ := DetectRequired(dir)
		if !r.Needed || r.MajorVersion != 8 {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("runtimeconfig no frameworks and valid tfm -> self contained", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Game.runtimeconfig.json", `{"runtimeOptions":{"tfm":"net6.0"}}`)
		r, _ := DetectRequired(dir)
		if r.Needed || r.Source != SourceSelfContained {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("nested runtimeconfig found via walk", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, sub, "Game.runtimeconfig.json",
			`{"runtimeOptions":{"tfm":"net10.0","frameworks":[{"name":"Microsoft.WindowsDesktop.App","version":"10.0.0"}]}}`)
		r, _ := DetectRequired(dir)
		if !r.Needed || r.MajorVersion != 10 {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("nothing found", func(t *testing.T) {
		r, _ := DetectRequired(t.TempDir())
		if r.Needed || r.Source != SourceNone {
			t.Errorf("unexpected: %+v", r)
		}
	})
	t.Run("nonexistent dir", func(t *testing.T) {
		r, _ := DetectRequired(filepath.Join(t.TempDir(), "does-not-exist"))
		if r.Needed || r.Source != SourceNone {
			t.Errorf("unexpected: %+v", r)
		}
	})
}
