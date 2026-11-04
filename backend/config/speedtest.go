package config

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// 清单线路标识（前端据此做本地化显示）
const (
	ManifestLineMain   = "main"   // 主线 sc.btos.top
	ManifestLineMirror = "mirror" // 镜像 dl.sc.btos.top
)

// ManifestSpeedResult 单条线路的测速结果
type ManifestSpeedResult struct {
	Line      string `json:"line"`               // 线路标识 main / mirror
	URL       string `json:"url"`                // 线路地址
	LatencyMs int64  `json:"latencyMs"`          // 耗时（毫秒），失败时为 -1
	OK        bool   `json:"ok"`                 // 是否测速成功
	Error     string `json:"error,omitempty"`    // 失败原因
}

// TestManifestSpeed 并发测速默认清单源的两条线路
// 清单文件本身很小（十几 KB），直接完整下载并计时即可反映线路快慢
func TestManifestSpeed() []ManifestSpeedResult {
	lines := []struct {
		line string
		url  string
	}{
		{ManifestLineMain, DefaultManifestURLMain},
		{ManifestLineMirror, DefaultManifestURLMirror},
	}

	results := make([]ManifestSpeedResult, len(lines))
	var wg sync.WaitGroup
	for i, l := range lines {
		wg.Add(1)
		go func(i int, line, url string) {
			defer wg.Done()
			latency, err := measureURL(url)
			r := ManifestSpeedResult{Line: line, URL: url, LatencyMs: latency}
			if err != nil {
				r.LatencyMs = -1
				r.Error = err.Error()
			} else {
				r.OK = true
			}
			results[i] = r
		}(i, l.line, l.url)
	}
	wg.Wait()
	return results
}

// measureURL 测量下载指定 URL 的耗时（毫秒）
func measureURL(url string) (int64, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return -1, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("status %d", resp.StatusCode)
	}

	// 读完整个响应（文件很小），确保测的是完整下载耗时
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return -1, fmt.Errorf("read body failed: %w", err)
	}
	return time.Since(start).Milliseconds(), nil
}

// PickFastestLine 从测速结果中选出最快线路（仅考虑成功的结果）
// 并列时优先主线；全部失败时返回 false（保持原选择不变）
func PickFastestLine(results []ManifestSpeedResult) (ManifestSpeedResult, bool) {
	var best ManifestSpeedResult
	found := false
	for _, r := range results {
		if !r.OK {
			continue
		}
		if !found || r.LatencyMs < best.LatencyMs {
			best = r
			found = true
		}
	}
	return best, found
}

// ApplyDefaultManifestSpeedTest 执行测速并应用结果：
// 更新选中的默认清单地址、各线路延迟、测速时间，并同步默认源条目与当前清单 URL
func (c *Config) ApplyDefaultManifestSpeedTest() ([]ManifestSpeedResult, error) {
	results := TestManifestSpeed()

	// 全部失败时保持现有选择不变，仅记录延迟结果
	if best, ok := PickFastestLine(results); ok {
		c.DefaultManifestURL = best.URL
	}

	for _, r := range results {
		switch r.Line {
		case ManifestLineMain:
			c.DefaultManifestMainLatencyMs = r.LatencyMs
		case ManifestLineMirror:
			c.DefaultManifestMirrorLatencyMs = r.LatencyMs
		}
	}
	c.DefaultManifestLastTest = time.Now().Unix()

	// 同步默认源条目的 URL（设置页显示用）
	for i := range c.ManifestSources {
		if c.ManifestSources[i].ID == "default" {
			c.ManifestSources[i].URL = c.DefaultManifestURL
		}
	}

	// 当前选中的是默认源时，同步实际使用的清单 URL
	if c.CurrentManifestSourceID == "default" {
		c.ManifestURL = c.DefaultManifestURL
	}

	// 保存失败不影响本次会话使用（内存中已更新），仅向上返回错误供记录
	saveErr := c.Save()
	return results, saveErr
}
