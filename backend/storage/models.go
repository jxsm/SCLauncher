package storage

import (
	"time"
)

// VersionModel 版本数据模型
type VersionModel struct {
	ID          string    `gorm:"primaryKey;column:id" json:"id"`
	VersionType string    `gorm:"column:version_type;index" json:"versionType"` // api, net, original
	GameVersion string    `gorm:"column:game_version;index" json:"gameVersion"` // 2.31, 2.4 等
	SubVersion  string    `gorm:"column:sub_version" json:"subVersion"`         // API1.60 等
	Name        string    `gorm:"column:name" json:"name"`                      // 显示名称
	Size        int64     `gorm:"column:size" json:"size"`                      // 文件大小
	DownloadURL string    `gorm:"column:download_url" json:"downloadUrl"`       // 下载地址
	Checksum    string    `gorm:"column:checksum" json:"checksum"`              // SHA256 校验和
	FileFormat  string    `gorm:"column:file_format" json:"fileFormat"`        // 文件格式 (zip)
	Installed   bool      `gorm:"column:installed;index" json:"installed"`      // 是否已安装
	IsPrimary   bool      `gorm:"column:is_primary;index" json:"isPrimary"`     // 是否为主要版本
	LocalPath   string    `gorm:"column:local_path" json:"localPath,omitempty"` // 本地路径
	Illustrate  string    `gorm:"column:illustrate" json:"illustrate"`          // 说明
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (VersionModel) TableName() string {
	return "versions"
}

// ModModel 模组数据模型
type ModModel struct {
	ID          string    `gorm:"primaryKey;column:id" json:"id"`
	VersionID   string    `gorm:"column:version_id;index" json:"versionId"` // 关联的版本 ID
	Name        string    `gorm:"column:name" json:"name"`                  // 模组名称
	FileName    string    `gorm:"column:file_name" json:"fileName"`         // 文件名称
	Version     string    `gorm:"column:version" json:"version"`            // 模组版本
	Author      string    `gorm:"column:author" json:"author"`              // 作者
	Description string    `gorm:"column:description" json:"description"`    // 描述
	Enabled     bool      `gorm:"column:enabled;default:true" json:"enabled"` // 是否启用
	InstallDate time.Time `gorm:"column:install_date" json:"installDate"`  // 安装日期
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (ModModel) TableName() string {
	return "mods"
}

// DownloadTaskModel 下载任务数据模型
type DownloadTaskModel struct {
	ID            string    `gorm:"primaryKey;column:id" json:"id"`
	Type          string    `gorm:"column:type;index" json:"type"` // version, mod
	Name          string    `gorm:"column:name" json:"name"`       // 显示名称
	URL           string    `gorm:"column:url" json:"url"`         // 下载地址
	SavePath      string    `gorm:"column:save_path" json:"savePath"` // 保存路径
	TotalSize     int64     `gorm:"column:total_size" json:"totalSize"` // 总大小
	Downloaded    int64     `gorm:"column:downloaded" json:"downloaded"` // 已下载大小
	Speed         int64     `gorm:"column:speed" json:"speed"`       // 下载速度（字节/秒）
	Status        string    `gorm:"column:status;index" json:"status"` // pending, downloading, paused, completed, failed
	Error         string    `gorm:"column:error" json:"error,omitempty"` // 错误信息
	RetryCount    int       `gorm:"column:retry_count;default:0" json:"retryCount"` // 重试次数
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定表名
func (DownloadTaskModel) TableName() string {
	return "download_tasks"
}

// GameProcessModel 游戏进程记录模型
type GameProcessModel struct {
	ID         string    `gorm:"primaryKey;column:id" json:"id"`
	VersionID  string    `gorm:"column:version_id;index" json:"versionId"` // 关联的版本 ID
	PID        int       `gorm:"column:pid" json:"pid"`                     // 进程 ID
	StartTime  time.Time `gorm:"column:start_time" json:"startTime"`       // 启动时间
	EndTime    *time.Time `gorm:"column:end_time" json:"endTime,omitempty"` // 结束时间
	ExitCode   *int      `gorm:"column:exit_code" json:"exitCode,omitempty"` // 退出码
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName 指定表名
func (GameProcessModel) TableName() string {
	return "game_processes"
}
