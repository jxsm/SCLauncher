package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config 应用配置
type Config struct {
	// 清单文件 URL
	ManifestURL string `json:"manifestUrl"`

	// 版本存储目录（相对路径）
	VersionsDir string `json:"versionsDir"`

	// 数据目录（相对路径）
	DataDir string `json:"dataDir"`

	// 下载临时目录（相对路径）
	DownloadsDir string `json:"downloadsDir"`

	// 最大并发下载数
	MaxConcurrent int `json:"maxConcurrent"`

	// 当前选中的版本 ID
	CurrentVersion string `json:"currentVersion"`

	// 主题 (dark/light)
	Theme string `json:"theme"`

	// 语言
	Language string `json:"language"`

	// 自动检查更新
	AutoCheckUpdates bool `json:"autoCheckUpdates"`

	// 背景图片路径（相对路径）
	BackgroundImage string `json:"backgroundImage"`

	// 不再提醒更新的时间戳（Unix时间戳，0表示未设置）
	UpdateRemindDisableUntil int64 `json:"updateRemindDisableUntil"`

	// 配置文件路径
	configPath string

	// 可执行文件所在目录（绝对路径）
	execDir string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	execDir := getExecDir()

	return &Config{
		ManifestURL:      "https://github.com/jxsm/SCVersionList/raw/refs/heads/main/manifest.json",
		VersionsDir:      filepath.Join(".Survivalcraft", "versions"),
		DataDir:          filepath.Join(".Survivalcraft", "data"),
		DownloadsDir:     filepath.Join(".Survivalcraft", "downloads"),
		MaxConcurrent:    3,
		CurrentVersion:   "",
		Theme:            "dark",
		Language:                  "zh-CN",
		AutoCheckUpdates:           true,
		BackgroundImage:            "",
		UpdateRemindDisableUntil:   0,
		execDir:                    execDir,
	}
}

// getExecDir 获取可执行文件所在目录
func getExecDir() string {
	execPath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前目录
		return "."
	}
	return filepath.Dir(execPath)
}

// GetAppDataDir 获取应用数据目录（使用启动器目录）
func GetAppDataDir() string {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前目录
		execPath = "."
	}

	execDir := filepath.Dir(execPath)

	// 在启动器目录下创建 .Survivalcraft 目录
	return filepath.Join(execDir, ".Survivalcraft")
}

// getAppDataDir 获取应用数据目录（内部函数）
func getAppDataDir() string {
	return GetAppDataDir()
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	execDir := getExecDir()

	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		config.configPath = configPath
		config.execDir = execDir
		// 创建配置目录
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
		// 保存默认配置
		if err := config.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 JSON
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.configPath = configPath
	config.execDir = execDir

	// 确保所有目录存在
	if err := config.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}

	return config, nil
}

// toRelativePath 将绝对路径转换为相对路径（内部使用）
func (c *Config) toRelativePath(absolutePath string) string {
	if absolutePath == "" {
		return ""
	}
	// 如果已经是相对路径，直接返回
	if !filepath.IsAbs(absolutePath) {
		return absolutePath
	}
	// 尝试转换为相对于可执行文件目录的路径
	relPath, err := filepath.Rel(c.execDir, absolutePath)
	if err != nil {
		// 转换失败，返回原始路径
		return absolutePath
	}
	return relPath
}

// Save 保存配置到文件
func (c *Config) Save() error {
	if c.configPath == "" {
		return fmt.Errorf("config path is not set")
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(c.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 创建临时配置对象用于序列化（只保存相对路径）
	tempConfig := struct {
		ManifestURL               string `json:"manifestUrl"`
		VersionsDir               string `json:"versionsDir"`
		DataDir                   string `json:"dataDir"`
		DownloadsDir              string `json:"downloadsDir"`
		MaxConcurrent             int    `json:"maxConcurrent"`
		CurrentVersion            string `json:"currentVersion"`
		Theme                     string `json:"theme"`
		Language                  string `json:"language"`
		AutoCheckUpdates          bool   `json:"autoCheckUpdates"`
		BackgroundImage           string `json:"backgroundImage"`
		UpdateRemindDisableUntil  int64  `json:"updateRemindDisableUntil"`
	}{
		ManifestURL:               c.ManifestURL,
		VersionsDir:               c.toRelativePath(c.VersionsDir),
		DataDir:                   c.toRelativePath(c.DataDir),
		DownloadsDir:              c.toRelativePath(c.DownloadsDir),
		MaxConcurrent:             c.MaxConcurrent,
		CurrentVersion:            c.CurrentVersion,
		Theme:                     c.Theme,
		Language:                  c.Language,
		AutoCheckUpdates:          c.AutoCheckUpdates,
		BackgroundImage:           c.toRelativePath(c.BackgroundImage),
		UpdateRemindDisableUntil:  c.UpdateRemindDisableUntil,
	}

	// 序列化为 JSON（带缩进）
	data, err := json.MarshalIndent(tempConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsureDirs 确保所有必要的目录存在
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.GetAbsolutePath(c.VersionsDir),
		c.GetAbsolutePath(c.DataDir),
		c.GetAbsolutePath(c.DownloadsDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetAbsolutePath 将相对路径转换为绝对路径
func (c *Config) GetAbsolutePath(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	// 如果已经是绝对路径，直接返回
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	// 转换为绝对路径
	return filepath.Join(c.execDir, relativePath)
}

// GetRelativePathForDisplay 获取用于显示的相对路径
func (c *Config) GetRelativePathForDisplay(absolutePath string) string {
	if absolutePath == "" {
		return ""
	}
	// 转换为相对路径
	relPath := c.toRelativePath(absolutePath)
	// 添加 ./ 前缀使其更清晰
	if relPath != "" && !filepath.IsAbs(relPath) && relPath[0] != '.' && relPath[0] != '/' {
		return "./" + filepath.ToSlash(relPath)
	}
	return filepath.ToSlash(relPath)
}

// GetVersionsDir 获取版本目录的绝对路径
func (c *Config) GetVersionsDir() string {
	return c.GetAbsolutePath(c.VersionsDir)
}

// GetDataDir 获取数据目录的绝对路径
func (c *Config) GetDataDir() string {
	return c.GetAbsolutePath(c.DataDir)
}

// GetDownloadsDir 获取下载目录的绝对路径
func (c *Config) GetDownloadsDir() string {
	return c.GetAbsolutePath(c.DownloadsDir)
}

// GetBackgroundImagePath 获取背景图片的绝对路径
func (c *Config) GetBackgroundImagePath() string {
	if c.BackgroundImage == "" {
		return ""
	}
	return c.GetAbsolutePath(c.BackgroundImage)
}

// GetVersionPath 获取指定版本的路径
func (c *Config) GetVersionPath(versionID string) string {
	return filepath.Join(c.GetVersionsDir(), versionID)
}

// GetModPath 获取指定版本的模组路径
func (c *Config) GetModPath(versionID string) string {
	return filepath.Join(c.GetVersionPath(versionID), "mods")
}

// GetDownloadTempPath 获取下载临时文件路径
func (c *Config) GetDownloadTempPath(filename string) string {
	return filepath.Join(c.GetDownloadsDir(), ".tmp", filename)
}

// SetCurrentVersion 设置当前选中的版本
func (c *Config) SetCurrentVersion(versionID string) error {
	c.CurrentVersion = versionID
	return c.Save()
}

// SetManifestURL 设置清单文件 URL
func (c *Config) SetManifestURL(url string) error {
	c.ManifestURL = url
	return c.Save()
}

// SetMaxConcurrent 设置最大并发下载数
func (c *Config) SetMaxConcurrent(max int) error {
	c.MaxConcurrent = max
	return c.Save()
}

// SetLanguage 设置语言
func (c *Config) SetLanguage(lang string) error {
	c.Language = lang
	return c.Save()
}

// SetBackgroundImage 设置背景图片路径
func (c *Config) SetBackgroundImage(relativePath string) error {
	c.BackgroundImage = relativePath
	return c.Save()
}

// SetUpdateRemindDisableUntil 设置不再提醒更新的截止时间戳
func (c *Config) SetUpdateRemindDisableUntil(timestamp int64) error {
	c.UpdateRemindDisableUntil = timestamp
	return c.Save()
}

// ShouldCheckUpdate 是否应该检查更新（考虑不再提醒期限）
func (c *Config) ShouldCheckUpdate() bool {
	// 如果未设置不再提醒，则应该检查
	if c.UpdateRemindDisableUntil == 0 {
		return true
	}
	// 检查当前时间是否已超过截止时间
	currentTime := time.Now().Unix()
	return currentTime > c.UpdateRemindDisableUntil
}
