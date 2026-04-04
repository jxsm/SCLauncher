package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Manifest 清单文件结构
type Manifest struct {
	API      map[string][]ManifestVersion `json:"api"`      // 插件版
	NET      map[string][]ManifestVersion `json:"net"`      // 联机版
	Original map[string][]ManifestVersion `json:"original"` // 原版
}

// ManifestVersion 清单中的版本信息
type ManifestVersion struct {
	SubVersion string `json:"sub_version"` // 子版本号（如 API1.60）
	Size       int64  `json:"size"`        // 文件大小
	Path       string `json:"path"`        // 下载地址
	FileFormat string `json:"file_format"` // 文件格式（zip）
	Illustrate string `json:"illustrate"`  // 说明
	SHA256     string `json:"sha256"`      // SHA256 校验和
}

// VersionType 版本类型
type VersionType string

const (
	VersionTypeAPI      VersionType = "api"      // 插件版
	VersionTypeNET      VersionType = "net"      // 联机版
	VersionTypeOriginal VersionType = "original" // 原版
)

// Version 完整的版本信息
type Version struct {
	ID          string     `json:"id"`          // 唯一 ID (如 api-2.31-API1.60)
	VersionType VersionType `json:"versionType"` // 版本类型
	GameVersion string     `json:"gameVersion"` // 游戏版本（如 2.31）
	SubVersion  string     `json:"subVersion"`  // 子版本（如 API1.60）
	Name        string     `json:"name"`        // 显示名称
	Size        int64      `json:"size"`        // 文件大小
	DownloadURL string     `json:"downloadUrl"` // 下载地址
	Checksum    string     `json:"checksum"`    // SHA256 校验和
	FileFormat  string     `json:"fileFormat"`  // 文件格式
	Illustrate  string     `json:"illustrate"`  // 说明
	ReleaseDate time.Time  `json:"releaseDate"` // 发布日期
	Installed   bool       `json:"installed"`   // 是否已安装（运行时计算）
	LocalPath   string     `json:"localPath,omitempty"` // 本地路径（运行时计算）
	PathExists  bool       `json:"pathExists"` // 路径是否存在（用于检测手动删除）
}

// ManifestParser 清单解析器
type ManifestParser struct {
	client *http.Client
}

// NewManifestParser 创建清单解析器
func NewManifestParser() *ManifestParser {
	return &ManifestParser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ParseFromURL 从 URL 解析清单文件
func (p *ManifestParser) ParseFromURL(url string) (*Manifest, error) {
	// 下载清单文件
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: status %d", resp.StatusCode)
	}

	// 读取内容
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// 解析 JSON
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// ParseFromBytes 从字节数组解析清单文件
func (p *ManifestParser) ParseFromBytes(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}
	return &manifest, nil
}

// ToVersions 将清单转换为版本列表
func (m *Manifest) ToVersions() []Version {
	var versions []Version

	// 解析插件版
	for gameVersion, manifestVersions := range m.API {
		for _, mv := range manifestVersions {
			versions = append(versions, Version{
				ID:          generateVersionID(VersionTypeAPI, gameVersion, mv.SubVersion),
				VersionType: VersionTypeAPI,
				GameVersion: gameVersion,
				SubVersion:  mv.SubVersion,
				Name:        fmt.Sprintf("插件版 %s %s", gameVersion, mv.SubVersion),
				Size:        mv.Size,
				DownloadURL: mv.Path,
				Checksum:    mv.SHA256,
				FileFormat:  mv.FileFormat,
				Illustrate:  mv.Illustrate,
				ReleaseDate: time.Now(), // 清单中没有发布日期，使用当前时间
			})
		}
	}

	// 解析联机版
	for gameVersion, manifestVersions := range m.NET {
		for _, mv := range manifestVersions {
			versions = append(versions, Version{
				ID:          generateVersionID(VersionTypeNET, gameVersion, mv.SubVersion),
				VersionType: VersionTypeNET,
				GameVersion: gameVersion,
				SubVersion:  mv.SubVersion,
				Name:        fmt.Sprintf("联机版 %s %s", gameVersion, mv.SubVersion),
				Size:        mv.Size,
				DownloadURL: mv.Path,
				Checksum:    mv.SHA256,
				FileFormat:  mv.FileFormat,
				Illustrate:  mv.Illustrate,
				ReleaseDate: time.Now(),
			})
		}
	}

	// 解析原版
	for gameVersion, manifestVersions := range m.Original {
		for _, mv := range manifestVersions {
			versions = append(versions, Version{
				ID:          generateVersionID(VersionTypeOriginal, gameVersion, mv.SubVersion),
				VersionType: VersionTypeOriginal,
				GameVersion: gameVersion,
				SubVersion:  mv.SubVersion,
				Name:        fmt.Sprintf("原版 %s %s", gameVersion, mv.SubVersion),
				Size:        mv.Size,
				DownloadURL: mv.Path,
				Checksum:    mv.SHA256,
				FileFormat:  mv.FileFormat,
				Illustrate:  mv.Illustrate,
				ReleaseDate: time.Now(),
			})
		}
	}

	return versions
}

// generateVersionID 生成版本唯一 ID
func generateVersionID(vtype VersionType, gameVersion, subVersion string) string {
	return fmt.Sprintf("%s-%s-%s", vtype, gameVersion, subVersion)
}

// GetDisplayName 获取版本类型的显示名称
func (vt VersionType) GetDisplayName() string {
	switch vt {
	case VersionTypeAPI:
		return "插件版"
	case VersionTypeNET:
		return "联机版"
	case VersionTypeOriginal:
		return "原版"
	default:
		return string(vt)
	}
}
