package storage

import (
	"time"

	"gorm.io/gorm"
)

// Repository 数据访问层
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository
func NewRepository(db *Database) *Repository {
	return &Repository{db: db.DB()}
}

// ========== 版本相关 ==========

// CreateVersion 创建版本记录
func (r *Repository) CreateVersion(version *VersionModel) error {
	return r.db.Create(version).Error
}

// GetVersion 获取版本
func (r *Repository) GetVersion(id string) (*VersionModel, error) {
	var version VersionModel
	err := r.db.Where("id = ?", id).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// ListVersions 列出所有版本
func (r *Repository) ListVersions() ([]VersionModel, error) {
	var versions []VersionModel
	err := r.db.Order("game_version DESC, sub_version DESC").Find(&versions).Error
	return versions, err
}

// ListVersionsByType 按类型列出版本
func (r *Repository) ListVersionsByType(versionType string) ([]VersionModel, error) {
	var versions []VersionModel
	err := r.db.Where("version_type = ?", versionType).
		Order("game_version DESC, sub_version DESC").
		Find(&versions).Error
	return versions, err
}

// ListInstalledVersions 列出已安装的版本
func (r *Repository) ListInstalledVersions() ([]VersionModel, error) {
	var versions []VersionModel
	err := r.db.Where("installed = ?", true).
		Order("game_version DESC, sub_version DESC").
		Find(&versions).Error
	return versions, err
}

// UpdateVersion 更新版本
func (r *Repository) UpdateVersion(version *VersionModel) error {
	return r.db.Save(version).Error
}

// UpdateVersionInstalledStatus 更新版本安装状态
func (r *Repository) UpdateVersionInstalledStatus(id string, installed bool, localPath string) error {
	return r.db.Model(&VersionModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"installed":  installed,
			"local_path": localPath,
		}).Error
}

// DeleteVersion 删除版本
func (r *Repository) DeleteVersion(id string) error {
	return r.db.Delete(&VersionModel{}, "id = ?", id).Error
}

// BatchCreateVersions 批量创建版本
func (r *Repository) BatchCreateVersions(versions []VersionModel) error {
	if len(versions) == 0 {
		return nil
	}
	return r.db.Create(&versions).Error
}

// GetPrimaryVersion 获取主要版本
func (r *Repository) GetPrimaryVersion() (*VersionModel, error) {
	var version VersionModel
	err := r.db.Where("is_primary = ?", true).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// SetPrimaryVersion 设置主要版本（同时取消其他版本的主要标记）
func (r *Repository) SetPrimaryVersion(id string) error {
	// 使用事务：先取消所有版本的主要标记，再设置新的主要版本
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 取消所有版本的主要标记
		if err := tx.Model(&VersionModel{}).Where("is_primary = ?", true).Update("is_primary", false).Error; err != nil {
			return err
		}
		// 如果提供了ID，设置新的主要版本
		if id != "" {
			if err := tx.Model(&VersionModel{}).Where("id = ?", id).Update("is_primary", true).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// AutoSetPrimaryVersion 自动设置主要版本（选择最新的已安装版本）
func (r *Repository) AutoSetPrimaryVersion() error {
	// 检查是否已有主要版本
	var count int64
	if err := r.db.Model(&VersionModel{}).Where("is_primary = ?", true).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		// 已有主要版本，不需要设置
		return nil
	}

	// 没有主要版本，选择最新的已安装版本
	var version VersionModel
	err := r.db.Where("installed = ?", true).
		Order("game_version DESC, sub_version DESC, created_at DESC").
		First(&version).Error
	if err != nil {
		// 没有已安装版本，不需要设置
		return nil
	}

	// 设置为主要版本
	return r.SetPrimaryVersion(version.ID)
}

// CheckVersionNameExists 检查版本名称是否存在（排除指定ID）
func (r *Repository) CheckVersionNameExists(name string, excludeID string) (bool, error) {
	var count int64
	err := r.db.Model(&VersionModel{}).
		Where("name = ? AND id != ?", name, excludeID).
		Count(&count).Error
	return count > 0, err
}

// RenameVersion 重命名版本
func (r *Repository) RenameVersion(id, newName string) error {
	return r.db.Model(&VersionModel{}).
		Where("id = ?", id).
		Update("name", newName).Error
}

// ========== 模组相关 ==========

// CreateMod 创建模组记录
func (r *Repository) CreateMod(mod *ModModel) error {
	return r.db.Create(mod).Error
}

// GetMod 获取模组
func (r *Repository) GetMod(id string) (*ModModel, error) {
	var mod ModModel
	err := r.db.Where("id = ?", id).First(&mod).Error
	if err != nil {
		return nil, err
	}
	return &mod, nil
}

// ListMods 列出指定版本的所有模组
func (r *Repository) ListMods(versionID string) ([]ModModel, error) {
	var mods []ModModel
	err := r.db.Where("version_id = ?", versionID).
		Order("created_at DESC").
		Find(&mods).Error
	return mods, err
}

// ListEnabledMods 列出启用的模组
func (r *Repository) ListEnabledMods(versionID string) ([]ModModel, error) {
	var mods []ModModel
	err := r.db.Where("version_id = ? AND enabled = ?", versionID, true).
		Order("created_at DESC").
		Find(&mods).Error
	return mods, err
}

// UpdateMod 更新模组
func (r *Repository) UpdateMod(mod *ModModel) error {
	return r.db.Save(mod).Error
}

// ToggleModEnabled 切换模组启用状态
func (r *Repository) ToggleModEnabled(id string, enabled bool) error {
	return r.db.Model(&ModModel{}).
		Where("id = ?", id).
		Update("enabled", enabled).Error
}

// DeleteMod 删除模组
func (r *Repository) DeleteMod(id string) error {
	return r.db.Delete(&ModModel{}, "id = ?", id).Error
}

// DeleteModsByVersion 删除指定版本的所有模组
func (r *Repository) DeleteModsByVersion(versionID string) error {
	return r.db.Where("version_id = ?", versionID).Delete(&ModModel{}).Error
}

// ========== 下载任务相关 ==========

// CreateDownloadTask 创建下载任务
func (r *Repository) CreateDownloadTask(task *DownloadTaskModel) error {
	return r.db.Create(task).Error
}

// GetDownloadTask 获取下载任务
func (r *Repository) GetDownloadTask(id string) (*DownloadTaskModel, error) {
	var task DownloadTaskModel
	err := r.db.Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListDownloadTasks 列出所有下载任务
func (r *Repository) ListDownloadTasks() ([]DownloadTaskModel, error) {
	var tasks []DownloadTaskModel
	err := r.db.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// ListActiveDownloadTasks 列出活跃的下载任务
func (r *Repository) ListActiveDownloadTasks() ([]DownloadTaskModel, error) {
	var tasks []DownloadTaskModel
	err := r.db.Where("status IN ?", []string{"pending", "downloading"}).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

// UpdateDownloadTask 更新下载任务
func (r *Repository) UpdateDownloadTask(task *DownloadTaskModel) error {
	return r.db.Save(task).Error
}

// UpdateDownloadTaskProgress 更新下载进度
func (r *Repository) UpdateDownloadTaskProgress(id string, downloaded int64, speed int64) error {
	return r.db.Model(&DownloadTaskModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"downloaded": downloaded,
			"speed":      speed,
		}).Error
}

// UpdateDownloadTaskStatus 更新下载任务状态
func (r *Repository) UpdateDownloadTaskStatus(id string, status string, errMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return r.db.Model(&DownloadTaskModel{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteDownloadTask 删除下载任务
func (r *Repository) DeleteDownloadTask(id string) error {
	return r.db.Delete(&DownloadTaskModel{}, "id = ?", id).Error
}

// DeleteCompletedDownloadTasks 删除已完成的下载任务
func (r *Repository) DeleteCompletedDownloadTasks() error {
	return r.db.Where("status = ?", "completed").Delete(&DownloadTaskModel{}).Error
}

// ========== 游戏进程相关 ==========

// CreateGameProcess 创建游戏进程记录
func (r *Repository) CreateGameProcess(process *GameProcessModel) error {
	return r.db.Create(process).Error
}

// GetGameProcess 获取游戏进程记录
func (r *Repository) GetGameProcess(id string) (*GameProcessModel, error) {
	var process GameProcessModel
	err := r.db.Where("id = ?", id).First(&process).Error
	if err != nil {
		return nil, err
	}
	return &process, nil
}

// GetRunningGameProcess 获取正在运行的游戏进程
func (r *Repository) GetRunningGameProcess() (*GameProcessModel, error) {
	var process GameProcessModel
	err := r.db.Where("end_time IS NULL").First(&process).Error
	if err != nil {
		return nil, err
	}
	return &process, nil
}

// UpdateGameProcessEnded 更新游戏进程结束信息
func (r *Repository) UpdateGameProcessEnded(id string, endTime time.Time, exitCode int) error {
	return r.db.Model(&GameProcessModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"end_time":  endTime,
			"exit_code": exitCode,
		}).Error
}

// ListGameProcessHistory 列出游戏进程历史
func (r *Repository) ListGameProcessHistory(limit int) ([]GameProcessModel, error) {
	var processes []GameProcessModel
	err := r.db.Order("start_time DESC").Limit(limit).Find(&processes).Error
	return processes, err
}

// ========== 通用方法 ==========

// BeginTx 开始事务
func (r *Repository) BeginTx() *gorm.DB {
	return r.db.Begin()
}

// Transaction 执行事务
func (r *Repository) Transaction(fn func(*Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &Repository{db: tx}
		return fn(txRepo)
	})
}

// Count 统计记录数
func (r *Repository) Count(model interface{}) (int64, error) {
	var count int64
	err := r.db.Model(model).Count(&count).Error
	return count, err
}
