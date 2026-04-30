package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DownloadWithRetry 带重试的下载函数
// ctx: context 用于取消操作
// client: HTTP 客户端
// url: 下载地址
// destPath: 目标文件路径
// maxRetries: 最大重试次数（临时性错误）
// progressCb: 可选的进度回调函数
func DownloadWithRetry(
	ctx context.Context,
	client *http.Client,
	url string,
	destPath string,
	maxRetries int,
	progressCb func(downloaded, total int64),
) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：2^attempt 秒，最大 30 秒
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// 尝试下载
		err := downloadOnce(ctx, client, url, destPath, progressCb)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !shouldRetry(err) || attempt == maxRetries {
			break
		}

		// 删除部分下载的文件
		os.Remove(destPath)
	}

	return fmt.Errorf("下载失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// downloadOnce 执行一次下载
func downloadOnce(
	ctx context.Context,
	client *http.Client,
	url string,
	destPath string,
	progressCb func(downloaded, total int64),
) error {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer func() {
		// 确保响应体被关闭
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 创建目标文件
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer func() {
		// 如果有错误，删除文件
		if out != nil {
			out.Close()
			os.Remove(destPath)
		}
	}()

	// 获取文件大小
	totalSize := resp.ContentLength

	// 复制数据并报告进度
	downloaded := int64(0)
	buffer := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := out.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}

			downloaded += int64(n)
			if progressCb != nil {
				progressCb(downloaded, totalSize)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		// 检查是否被取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// 下载成功，清除文件的删除标记
	out.Close()
	out = nil

	return nil
}

// shouldRetry 判断错误是否应该重试
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// 临时性错误应该重试
	errStr := err.Error()
	temporaryErrors := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"broken pipe",
	}

	return ContainsAny(errStr, temporaryErrors)
}
