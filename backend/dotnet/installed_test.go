package dotnet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionMajor(t *testing.T) {
	cases := []struct {
		v    string
		want int
		ok   bool
	}{
		{"9.0.16", 9, true},
		{"10.0.5", 10, true},
		{"3.1.32", 3, true},
		{"", 0, false},
		{"abc", 0, false},
		{"0.1", 0, false},
	}
	for _, c := range cases {
		got, ok := versionMajor(c.v)
		if ok != c.ok || got != c.want {
			t.Errorf("versionMajor(%q)=(%d,%v) want (%d,%v)", c.v, got, ok, c.want, c.ok)
		}
	}
}

func TestParseListRuntimes(t *testing.T) {
	out := "Microsoft.WindowsDesktop.App 9.0.16 [C:\\dotnet\\9.0.16]\n" +
		"Microsoft.WindowsDesktop.App 10.0.5 [C:\\dotnet\\10.0.5]\n" +
		"Microsoft.NETCore.App 9.0.16 [C:\\dotnet\\netcore]\n" +
		"Microsoft.AspNetCore.App 9.0.16 [C:\\dotnet\\aspnet]\n" +
		"\n" +
		"garbage line\n" +
		"Microsoft.WindowsDesktop.App preview.0 [path]\n"
	m := ParseListRuntimes(out)
	if !m[9] || !m[10] || len(m) != 2 {
		t.Errorf("expected {9,10}, got %v", m)
	}
}

func TestParseListRuntimesEmpty(t *testing.T) {
	if m := ParseListRuntimes(""); len(m) != 0 {
		t.Errorf("expected empty, got %v", m)
	}
}

func TestMajorsFromInstallDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "9.0.16"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "10.0.5"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "notadir.txt"), []byte("x"), 0644)

	m, err := MajorsFromInstallDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !m[9] || !m[10] || len(m) != 2 {
		t.Errorf("expected {9,10}, got %v", m)
	}
}

func TestMajorsFromInstallDirMissing(t *testing.T) {
	m, err := MajorsFromInstallDir(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty, got %v", m)
	}
}

func TestListInstalledDesktopMajors(t *testing.T) {
	t.Run("dotnet present", func(t *testing.T) {
		r := &fakeRunner{runFn: func(name string, args []string) (string, string, int, error) {
			return "Microsoft.WindowsDesktop.App 9.0.16 [p]\nMicrosoft.WindowsDesktop.App 10.0.5 [p]", "", 0, nil
		}}
		m, err := ListInstalledDesktopMajors(r)
		if err != nil {
			t.Fatal(err)
		}
		if !m[9] || !m[10] {
			t.Errorf("expected {9,10}, got %v", m)
		}
	})
	t.Run("dotnet missing", func(t *testing.T) {
		r := &fakeRunner{lookPathFn: func(string) error { return os.ErrNotExist }}
		m, err := ListInstalledDesktopMajors(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 0 {
			t.Errorf("expected empty, got %v", m)
		}
	})
	t.Run("run error", func(t *testing.T) {
		r := &fakeRunner{runFn: func(string, []string) (string, string, int, error) {
			return "", "boom", 1, os.ErrInvalid
		}}
		m, err := ListInstalledDesktopMajors(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 0 {
			t.Errorf("expected empty, got %v", m)
		}
	})
	t.Run("nil runner", func(t *testing.T) {
		if _, err := ListInstalledDesktopMajors(nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDefaultDesktopSharedPath(t *testing.T) {
	if p := DefaultDesktopSharedPath(); p == "" {
		t.Fatal("empty path")
	}
}
