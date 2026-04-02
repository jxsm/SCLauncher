package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	// 清单文件 URL
	ManifestURL string `json:"manifestUrl"`

	// 版本存储目录
	VersionsDir string `json:"versionsDir"`

	// 数据目录
	DataDir string `json:"dataDir"`

	// 下载临时目录
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

	// 配置文件路径
	configPath string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	appDataDir := GetAppDataDir()

	return &Config{
		ManifestURL:      "https://github.com/jxsm/SCVersionList/raw/refs/heads/main/manifest.json",
		VersionsDir:      filepath.Join(appDataDir, "versions"),
		DataDir:          filepath.Join(appDataDir, "data"),
		DownloadsDir:     filepath.Join(appDataDir, "downloads"),
		MaxConcurrent:    3,
		CurrentVersion:   "",
		Theme:            "dark",
		Language:         "zh-CN",
		AutoCheckUpdates: true,
	}
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
	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		config.configPath = configPath
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

	// 确保所有目录存在
	if err := config.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}

	return config, nil
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

	// 序列化为 JSON（带缩进）
	data, err := json.MarshalIndent(c, "", "  ")
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
		c.VersionsDir,
		c.DataDir,
		c.DownloadsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetVersionPath 获取指定版本的路径
func (c *Config) GetVersionPath(versionID string) string {
	return filepath.Join(c.VersionsDir, versionID)
}

// GetModPath 获取指定版本的模组路径
func (c *Config) GetModPath(versionID string) string {
	return filepath.Join(c.GetVersionPath(versionID), "mods")
}

// GetDownloadTempPath 获取下载临时文件路径
func (c *Config) GetDownloadTempPath(filename string) string {
	return filepath.Join(c.DownloadsDir, ".tmp", filename)
}

// SetCurrentVersion 设置当前选中的版本
func (c *Config) SetCurrentVersion(versionID string) error {
	c.CurrentVersion = versionID
	return c.Save()
}
