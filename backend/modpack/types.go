package modpack

// Manifest 整合包清单
type Manifest struct {
	// 基本信息
	ManifestType    string  `json:"manifestType"`    // 固定值：SurvivalcraftModpack
	ManifestVersion float64 `json:"manifestVersion"` // 清单版本号
	Name            string  `json:"name"`            // 整合包名称
	Version         string  `json:"version"`         // 整合包版本
	Author          string  `json:"author"`          // 作者
	Description     string  `json:"description"`     // 描述
	Icon            string  `json:"icon"`            // 图标文件路径
	Created         string  `json:"created"`         // 创建时间
	Changelog       string  `json:"changelog"`       // 更新日志

	// 游戏核心配置
	Survivalcraft *SurvivalcraftConfig `json:"survivalcraft"` // 生存战争配置

	// 模组列表
	Mods    []ModInfo `json:"mods"`    // 模组列表
	ModPath string    `json:"modPath"` // 模组存放路径（默认为/Mods，联机版为/NetMods）

	// 自定义覆盖文件
	Overrides string `json:"overrides"` // 覆盖文件目录名

	// 校验
	Checksum string `json:"checksum"` // 校验方式（sha256）

	// 解析后的额外信息
	FilePath        string `json:"-"` // 整合包文件路径
	FileHash        string `json:"-"` // 文件哈希值
	HasExternalLinks bool   `json:"hasExternalLinks"` // 是否包含外部链接
	IsCarryFormat   bool   `json:"isCarryFormat"` // 是否为carry格式（自带游戏）
}

// SurvivalcraftConfig 生存战争配置
type SurvivalcraftConfig struct {
	Version    VersionConfig      `json:"version"`    // 版本配置
	VersionList VersionListConfig `json:"versionList"` // 版本列表配置
}

// VersionConfig 版本配置
type VersionConfig struct {
	Manual  bool             `json:"manual"`  // 是否手动选择版本（默认false）
	Android *PlatformVersion `json:"android"` // Android 平台配置
	Windows *PlatformVersion `json:"windows"` // Windows 平台配置
}

// PlatformVersion 平台版本信息
type PlatformVersion struct {
	Version        string `json:"version"`         // 版本号（如：2.4:api-1.8.2.3 或 2.4:carry/game.zip）
	APKPackageName string `json:"apkPackageName"`  // APK 包名（Android）
	Path           string `json:"path"`            // 下载路径
}

// VersionListConfig 版本列表配置
type VersionListConfig struct {
	Android string `json:"android"` // Android 版本列表 URL
	Windows string `json:"windows"` // Windows 版本列表 URL
}

// ModInfo 模组信息
type ModInfo struct {
	ProjectID int    `json:"projectId"` // 模组项目 ID
	Version   string `json:"version"`   // 模组版本
	Name      string `json:"name"`      // 模组名称
	Required  bool   `json:"required"`  // 是否必须
	Path      string `json:"path"`      // 下载路径
	ModPath   string `json:"modPath"`   // 模组安装路径（可选，如果为空则使用全局ModPath）
}
