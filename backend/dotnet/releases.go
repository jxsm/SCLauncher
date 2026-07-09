package dotnet

import (
	"encoding/json"
	"fmt"
	"strings"
)

// releasesBase 是微软 .NET 发布元数据的基础 URL（设为变量便于测试重定向）。
var releasesBase = "https://builds.dotnet.microsoft.com/dotnet/release-metadata"

// InstallerAsset 选中的安装包信息。
type InstallerAsset struct {
	Version string // 例如 "9.0.16"
	URL     string // 下载地址
	SHA512  string // 校验和（小写 hex）
	RID     string // 例如 "win-x64"
}

// releasesMeta 对应 releases.json，只声明关心的字段。
type releasesMeta struct {
	ChannelVersion string `json:"channel-version"`
	LatestRelease  string `json:"latest-release"`
	Releases       []struct {
		ReleaseVersion string `json:"release-version"`
		WindowsDesktop *struct {
			Version string `json:"version"`
			Files   []struct {
				Name string `json:"name"`
				RID  string `json:"rid"`
				URL  string `json:"url"`
				Hash string `json:"hash"`
			} `json:"files"`
		} `json:"windowsdesktop"`
	} `json:"releases"`
}

// ReleasesURL 由主版本号拼出 releases.json 的 URL。
// 例如 major=10 → https://builds.dotnet.microsoft.com/dotnet/release-metadata/10.0/releases.json
func ReleasesURL(major int) string {
	return fmt.Sprintf("%s/%d.0/releases.json", releasesBase, major)
}

// RIDForArch 将 Go 体系结构（runtime.GOARCH）映射为 .NET RID。
func RIDForArch(goarch string) string {
	switch goarch {
	case "386":
		return "win-x86"
	case "arm64":
		return "win-arm64"
	default: // amd64 及未知 → x64（最常见）
		return "win-x64"
	}
}

// PickLatestDesktopInstaller 从 releases.json 字节里挑出指定架构下最新的
// WindowsDesktop Runtime 安装包（.exe）。
//
// releases.json 的 releases 数组按从新到旧排列，因此第一个匹配即为最新。
func PickLatestDesktopInstaller(data []byte, goarch string) (InstallerAsset, error) {
	var meta releasesMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return InstallerAsset{}, fmt.Errorf("解析 releases.json 失败: %w", err)
	}
	if len(meta.Releases) == 0 {
		return InstallerAsset{}, fmt.Errorf("releases.json 中无 release 条目")
	}
	want := RIDForArch(goarch)
	for _, rel := range meta.Releases {
		if rel.WindowsDesktop == nil {
			continue
		}
		for _, f := range rel.WindowsDesktop.Files {
			if f.RID == want && strings.HasSuffix(strings.ToLower(f.Name), ".exe") {
				return InstallerAsset{
					Version: rel.WindowsDesktop.Version,
					URL:     f.URL,
					SHA512:  f.Hash,
					RID:     f.RID,
				}, nil
			}
		}
	}
	return InstallerAsset{}, fmt.Errorf("releases.json 中未找到 RID=%s 的 .exe 安装包", want)
}
