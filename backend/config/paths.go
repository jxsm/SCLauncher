package config

import (
	"os"
	"path/filepath"
	"strings"
)

// VersionType 版本类型
type VersionType string

const (
	VersionTypeAPI      VersionType = "api"      // 插件版本
	VersionTypeNET      VersionType = "net"      // 联机版本
	VersionTypeOriginal VersionType = "original" // 原版
)

// VersionTypeNames 版本类型显示名称
var VersionTypeNames = map[VersionType]string{
	VersionTypeAPI:      "插件版",
	VersionTypeNET:      "联机版",
	VersionTypeOriginal: "原版",
}

// String 返回版本类型的字符串表示
func (vt VersionType) String() string {
	if name, ok := VersionTypeNames[vt]; ok {
		return name
	}
	return string(vt)
}

// GetDisplayName 获取显示名称
func (vt VersionType) GetDisplayName() string {
	return vt.String()
}

// Paths 路径管理器
type Paths struct {
	config *Config
}

// NewPaths 创建路径管理器
func NewPaths(config *Config) *Paths {
	return &Paths{config: config}
}

// GetVersionsDir 获取版本根目录（绝对路径）
func (p *Paths) GetVersionsDir() string {
	return p.config.GetVersionsDir()
}

// GetVersionPath 获取指定版本的根目录（绝对路径）
func (p *Paths) GetVersionPath(versionID string) string {
	return filepath.Join(p.GetVersionsDir(), versionID)
}

// GetGameExecutablePath 查找游戏可执行文件路径
// 在版本目录中查找包含 "Survivalcraft" 的 .exe 文件
func (p *Paths) GetGameExecutablePath(versionID string) (string, error) {
	versionPath := p.GetVersionPath(versionID)

	// 遍历版本目录查找 .exe 文件
	err := filepath.Walk(versionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 继续遍历
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查是否是 .exe 文件且文件名包含 "Survivalcraft"
		if strings.EqualFold(filepath.Ext(path), ".exe") &&
			strings.Contains(strings.ToLower(filepath.Base(path)), "survivalcraft") {
			return filepath.SkipDir // 找到了，停止遍历
		}

		return nil
	})

	if err != nil && err != filepath.SkipDir {
		return "", err
	}

	// 如果没有找到，返回错误
	return "", os.ErrNotExist
}

// GetModPath 获取模组目录路径（绝对路径）
func (p *Paths) GetModPath(versionID string) string {
	return filepath.Join(p.GetVersionPath(versionID), "mods")
}

// EnsureModDir 确保模组目录存在
func (p *Paths) EnsureModDir(versionID string) error {
	modPath := p.GetModPath(versionID)
	return os.MkdirAll(modPath, 0755)
}

// GetGameDataPath 获取游戏数据目录（绝对路径）
func (p *Paths) GetGameDataPath(versionID string) string {
	return filepath.Join(p.GetVersionPath(versionID), "data")
}

// EnsureGameDataDir 确保游戏数据目录存在
func (p *Paths) EnsureGameDataDir(versionID string) error {
	dataPath := p.GetGameDataPath(versionID)
	return os.MkdirAll(dataPath, 0755)
}

// GetDownloadTempPath 获取下载临时文件路径（绝对路径）
func (p *Paths) GetDownloadTempPath(filename string) string {
	return filepath.Join(p.config.GetDownloadsDir(), ".tmp", filename)
}

// GetDownloadTempDir 获取下载临时目录（绝对路径）
func (p *Paths) GetDownloadTempDir() string {
	return filepath.Join(p.config.GetDownloadsDir(), ".tmp")
}

// EnsureDownloadTempDir 确保下载临时目录存在
func (p *Paths) EnsureDownloadTempDir() error {
	tempDir := p.GetDownloadTempDir()
	return os.MkdirAll(tempDir, 0755)
}

// GetDatabasePath 获取数据库文件路径（绝对路径）
func (p *Paths) GetDatabasePath() string {
	return filepath.Join(p.config.GetDataDir(), "database.db")
}

// GetLogPath 获取日志文件路径（绝对路径）
func (p *Paths) GetLogPath() string {
	return filepath.Join(p.config.GetDataDir(), "launcher.log")
}

// GetConfigPath 获取配置文件路径
func (p *Paths) GetConfigPath() string {
	return p.config.configPath
}

// VersionExists 检查版本是否存在
func (p *Paths) VersionExists(versionID string) bool {
	versionPath := p.GetVersionPath(versionID)
	info, err := os.Stat(versionPath)
	return err == nil && info.IsDir()
}

// GetVersionSize 获取版本目录大小（字节）
func (p *Paths) GetVersionSize(versionID string) (int64, error) {
	var size int64

	versionPath := p.GetVersionPath(versionID)
	err := filepath.Walk(versionPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// DeleteVersion 删除版本目录
func (p *Paths) DeleteVersion(versionID string) error {
	versionPath := p.GetVersionPath(versionID)
	return os.RemoveAll(versionPath)
}

// IsVersionInstalled 检查版本是否已安装
func (p *Paths) IsVersionInstalled(versionID string) bool {
	// 检查版本目录是否存在
	if !p.VersionExists(versionID) {
		return false
	}

	// 检查游戏可执行文件是否存在
	execPath, err := p.GetGameExecutablePath(versionID)
	return err == nil && execPath != ""
}

// GetSkinsDir 获取皮肤存储目录（绝对路径）
func (p *Paths) GetSkinsDir() string {
	return filepath.Join(p.config.GetDataDir(), "Skins")
}

// GetGameCharacterSkinsDir 获取游戏角色皮肤目录
func (p *Paths) GetGameCharacterSkinsDir(versionID string) string {
	return filepath.Join(p.GetVersionPath(versionID), "CharacterSkins")
}

// GetGameDocCharacterSkinsDir 获取游戏doc目录下的角色皮肤目录
func (p *Paths) GetGameDocCharacterSkinsDir(versionID string) string {
	return filepath.Join(p.GetVersionPath(versionID), "doc", "CharacterSkins")
}

// GetBackgroundImageDir 获取背景图片存储目录（绝对路径）
func (p *Paths) GetBackgroundImageDir() string {
	return filepath.Join(p.config.GetDataDir(), "Backgrounds")
}

// GetBackgroundImagePath 获取背景图片的完整路径（绝对路径）
func (p *Paths) GetBackgroundImagePath(filename string) string {
	if filename == "" {
		return ""
	}
	return filepath.Join(p.GetBackgroundImageDir(), filename)
}
