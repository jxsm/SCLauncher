package dotnet

import (
	"strings"
	"testing"
)

const releasesFixture = `{
  "channel-version": "10.0",
  "latest-release": "10.0.0",
  "releases": [
    {
      "release-version": "10.0.0",
      "windowsdesktop": {
        "version": "10.0.0",
        "files": [
          {"name":"windowsdesktop-runtime-win-x64.exe","rid":"win-x64","url":"http://x64","hash":"aaa"},
          {"name":"windowsdesktop-runtime-win-x86.exe","rid":"win-x86","url":"http://x86","hash":"bbb"},
          {"name":"windowsdesktop-runtime-win-arm64.exe","rid":"win-arm64","url":"http://arm64","hash":"ccc"},
          {"name":"windowsdesktop-runtime-win-x64.zip","rid":"win-x64","url":"http://x64zip","hash":"ddd"}
        ]
      }
    }
  ]
}`

func TestReleasesURL(t *testing.T) {
	if got := ReleasesURL(10); got != "https://builds.dotnet.microsoft.com/dotnet/release-metadata/10.0/releases.json" {
		t.Errorf("unexpected url: %s", got)
	}
	if got := ReleasesURL(9); !strings.Contains(got, "/9.0/releases.json") {
		t.Errorf("unexpected url: %s", got)
	}
}

func TestRIDForArch(t *testing.T) {
	cases := map[string]string{
		"amd64": "win-x64",
		"386":   "win-x86",
		"arm64": "win-arm64",
		"":      "win-x64",
		"mips":  "win-x64",
	}
	for arch, want := range cases {
		if got := RIDForArch(arch); got != want {
			t.Errorf("RIDForArch(%q)=%q want %q", arch, got, want)
		}
	}
}

func TestPickLatestDesktopInstaller(t *testing.T) {
	t.Run("x64 exe skips zip", func(t *testing.T) {
		a, err := PickLatestDesktopInstaller([]byte(releasesFixture), "amd64")
		if err != nil {
			t.Fatal(err)
		}
		if a.URL != "http://x64" || a.Version != "10.0.0" || a.RID != "win-x64" || a.SHA512 != "aaa" {
			t.Errorf("unexpected asset: %+v", a)
		}
	})
	t.Run("x86", func(t *testing.T) {
		a, err := PickLatestDesktopInstaller([]byte(releasesFixture), "386")
		if err != nil || a.URL != "http://x86" {
			t.Fatalf("err=%v asset=%+v", err, a)
		}
	})
	t.Run("arm64", func(t *testing.T) {
		a, err := PickLatestDesktopInstaller([]byte(releasesFixture), "arm64")
		if err != nil || a.URL != "http://arm64" {
			t.Fatalf("err=%v asset=%+v", err, a)
		}
	})
	t.Run("skip nil windowsdesktop then match", func(t *testing.T) {
		data := []byte(`{"releases":[{"release-version":"10.0.0","windowsdesktop":null},{"release-version":"10.0.0","windowsdesktop":{"version":"10.0.0","files":[{"name":"x.exe","rid":"win-x64","url":"http://late","hash":"z"}]}}]}`)
		a, err := PickLatestDesktopInstaller(data, "amd64")
		if err != nil || a.URL != "http://late" {
			t.Fatalf("err=%v asset=%+v", err, a)
		}
	})
	t.Run("empty releases", func(t *testing.T) {
		if _, err := PickLatestDesktopInstaller([]byte(`{"releases":[]}`), "amd64"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("corrupt json", func(t *testing.T) {
		if _, err := PickLatestDesktopInstaller([]byte(`{bad`), "amd64"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("no matching rid (zip only)", func(t *testing.T) {
		data := []byte(`{"releases":[{"release-version":"10.0.0","windowsdesktop":{"version":"10.0.0","files":[{"name":"x.zip","rid":"win-x64","url":"http://zip"}]}}]}`)
		if _, err := PickLatestDesktopInstaller(data, "amd64"); err == nil {
			t.Fatal("expected error (only zip, no exe)")
		}
	})
}
