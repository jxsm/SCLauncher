package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DownloadProgress 下载进度回调
type DownloadProgress func(downloaded int64, total int64, speed int64)

// Downloader 版本下载器
type Downloader struct {
	client       *http.Client
	activeJobs   map[string]*downloadJob
	jobMutex     sync.RWMutex
	maxConcurrent int
}

// downloadJob 下载任务
type downloadJob struct {
	id          string
	url         string
	savePath    string
	total       int64
	downloaded  int64
	speed       int64
	status      string
	cancel      chan struct{}
	progressCb  DownloadProgress
	lastUpdate  time.Time
	lastBytes   int64
	completed   bool // 下载完成标志
}

// NewDownloader 创建下载器
func NewDownloader(maxConcurrent int) *Downloader {
	return &Downloader{
		client:        &http.Client{},
		activeJobs:    make(map[string]*downloadJob),
		maxConcurrent: maxConcurrent,
	}
}

// Download 下载文件
func (d *Downloader) Download(id, url, savePath string, progress DownloadProgress) error {
	// 创建下载任务
	job := &downloadJob{
		id:         id,
		url:        url,
		savePath:   savePath,
		status:     "downloading",
		cancel:     make(chan struct{}),
		progressCb: progress,
		lastUpdate: time.Now(),
		lastBytes:  0,
		completed:  false,
	}

	// 添加到活跃任务列表
	d.jobMutex.Lock()
	d.activeJobs[id] = job
	d.jobMutex.Unlock()

	// 确保保存目录存在
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// 检查是否支持断点续传
	tempPath := savePath + ".tmp"
	downloaded := int64(0)

	if info, err := os.Stat(tempPath); err == nil {
		downloaded = info.Size()
		job.downloaded = downloaded
	}

	// 初始化完成标志
	job.completed = false
	job.lastUpdate = time.Now()
	job.lastBytes = 0

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置断点续传头
	if downloaded > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloaded))
	}

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 获取文件大小
	job.total = downloaded + resp.ContentLength

	// 打开文件（追加模式）
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	// 注意：不要在这里 defer file.Close()，因为我们在协程中手动关闭

	// 创建写入缓冲区
	buffer := make([]byte, 32*1024) // 32KB 缓冲区
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// 创建进度更新通道
	progressChan := make(chan int64)
	doneChan := make(chan error)

	// 启动下载协程
	go func() {
		defer close(doneChan)
		defer close(progressChan)

		for {
			select {
			case <-job.cancel:
				doneChan <- fmt.Errorf("download cancelled")
				return
			default:
				// 读取数据
				n, err := resp.Body.Read(buffer)
				if n > 0 {
					if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
						doneChan <- writeErr
						return
					}
					job.downloaded += int64(n)
					progressChan <- job.downloaded
				}

				if err != nil {
					if err == io.EOF {
						// 下载完成，立即关闭文件
						file.Close()
						doneChan <- nil
						return
					}
					doneChan <- err
					return
				}
			}
		}
	}()

	// 启动进度更新协程
	go func() {
		for {
			select {
			case bytes, ok := <-progressChan:
				if !ok {
					return // 通道关闭，退出
				}
				// 计算速度
				now := time.Now()
				timeDiff := now.Sub(job.lastUpdate).Seconds()
				if timeDiff > 0 {
					bytesDiff := bytes - job.lastBytes
					job.speed = int64(float64(bytesDiff) / timeDiff)
					job.lastBytes = bytes
					job.lastUpdate = now
				}

				// 调用进度回调（只在未完成时调用）
				if job.progressCb != nil && !job.completed {
					job.progressCb(job.downloaded, job.total, job.speed)
				}
			}
		}
	}()

	// 等待下载完成
	if err := <-doneChan; err != nil {
		return err
	}

	// 标记为完成，停止发送进度事件
	job.completed = true

	// 等待文件句柄完全释放并重命名文件
	// 添加重试机制
	var renameErr error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 等待一小段时间再重试
			time.Sleep(time.Duration(i*50) * time.Millisecond)
		}
		renameErr = os.Rename(tempPath, savePath)
		if renameErr == nil {
			break // 成功
		}

		// 检查错误是否是文件被占用
		if strings.Contains(renameErr.Error(), "used by another process") {
			// 继续重试
			continue
		} else {
			// 其他错误直接返回
			return fmt.Errorf("failed to rename file: %w", renameErr)
		}
	}

	if renameErr != nil {
		return fmt.Errorf("failed to rename file after %d retries: %w", maxRetries, renameErr)
	}

	// 从活跃任务中移除
	d.jobMutex.Lock()
	delete(d.activeJobs, id)
	d.jobMutex.Unlock()

	return nil
}

// Cancel 取消下载
func (d *Downloader) Cancel(id string) error {
	d.jobMutex.RLock()
	job, exists := d.activeJobs[id]
	d.jobMutex.RUnlock()

	if !exists {
		return fmt.Errorf("download job not found: %s", id)
	}

	close(job.cancel)
	return nil
}

// VerifyChecksum 验证文件校验和
func VerifyChecksum(filePath, expectedChecksum string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	return actualChecksum == expectedChecksum, nil
}
