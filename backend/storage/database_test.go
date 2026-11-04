package storage

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConcurrentWritesNoBusy 回归测试：多个 goroutine 并发读写同一数据库时
// 不应出现 "database is locked (5) (SQLITE_BUSY)"。
// 复现线上场景：清单刷新批量 Create/Update 版本 与 下载/安装任务记录并发落库。
// 依赖 New() 中配置的 单连接池 + WAL + busy_timeout。
func TestConcurrentWritesNoBusy(t *testing.T) {
	db, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	const writers = 4
	const perWriter = 50

	var wg sync.WaitGroup
	errCh := make(chan error, writers*perWriter)

	start := make(chan struct{})
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			<-start
			for i := 0; i < perWriter; i++ {
				// 写路径 1：版本 Create/Update（清单刷新）
				if err := db.DB().Create(&VersionModel{
					ID:          fmt.Sprintf("v-%d-%d", w, i),
					VersionType: "api",
					GameVersion: "2.4",
					Name:        fmt.Sprintf("版本 %d-%d", w, i),
					UpdatedAt:   time.Now(),
				}).Error; err != nil {
					errCh <- fmt.Errorf("create version: %w", err)
					continue
				}
				// 写路径 2：下载任务记录（下载/安装流程）
				if err := db.DB().Create(&DownloadTaskModel{
					ID:     fmt.Sprintf("t-%d-%d", w, i),
					Type:   "version",
					Name:   fmt.Sprintf("task %d-%d", w, i),
					Status: "downloading",
				}).Error; err != nil {
					errCh <- fmt.Errorf("create task: %w", err)
					continue
				}
				// 读路径：并发查询
				var count int64
				if err := db.DB().Model(&VersionModel{}).Count(&count).Error; err != nil {
					errCh <- fmt.Errorf("count: %w", err)
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()
	close(errCh)

	for e := range errCh {
		t.Errorf("concurrent write failed: %v", e)
	}
}
