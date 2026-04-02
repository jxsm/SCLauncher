package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

// Database 数据库管理器
type Database struct {
	db *gorm.DB
}

// New 创建数据库连接
func New(dataDir string) (*Database, error) {
	dbPath := filepath.Join(dataDir, "database.db")

	// 使用纯 Go 的 SQLite 驱动
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// 使用 GORM 包装
	db, err := gorm.Open(sqlite.Dialector{
		Conn: sqlDB,
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 默认静默日志
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&VersionModel{},
		&ModModel{},
		&DownloadTaskModel{},
		&GameProcessModel{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Database{db: db}, nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB 获取 GORM DB 实例
func (d *Database) DB() *gorm.DB {
	return d.db
}

// SetLogger 设置日志级别
func (d *Database) SetLogger(level logger.LogLevel) {
	d.db.Logger = logger.Default.LogMode(level)
}
