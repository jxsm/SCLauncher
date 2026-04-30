package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// URLValidator URL验证器
type URLValidator struct {
	client *http.Client
}

// NewURLValidator 创建URL验证器
func NewURLValidator() *URLValidator {
	return &URLValidator{
		client: &http.Client{
			Timeout: 10 * time.Second,
			// 不跟随重定向，检查最终响应
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// ValidateURL 验证URL格式和可访问性
func (v *URLValidator) ValidateURL(urlStr string) error {
	// 检查基本格式
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL格式无效: %w", err)
	}

	// 检查协议和主机
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL缺少主机名")
	}

	return nil
}

// CheckURLAccessibility 检查URL的可访问性（HEAD请求）
// 返回是否可访问和状态码
func (v *URLValidator) CheckURLAccessibility(urlStr string) (accessible bool, statusCode int, err error) {
	// 先验证格式
	if err := v.ValidateURL(urlStr); err != nil {
		return false, 0, err
	}

	// 尝试 HEAD 请求
	req, err := http.NewRequest("HEAD", urlStr, nil)
	if err != nil {
		return false, 0, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent，避免某些服务器拒绝
	req.Header.Set("User-Agent", "SCLauncher/2.0")

	resp, err := v.client.Do(req)
	if err != nil {
		// 网络错误通常意味着不可访问
		return false, 0, nil
	}
	defer resp.Body.Close()

	// 检查状态码
	statusCode = resp.StatusCode
	accessible = statusCode >= 200 && statusCode < 400

	return accessible, statusCode, nil
}

// ValidateURLWithAccess 完整验证URL（格式+可访问性）
// skipAccessibilityCheck: 如果为true，跳过可访问性检查（仅验证格式）
func (v *URLValidator) ValidateURLWithAccess(urlStr string, skipAccessibilityCheck bool) error {
	// 验证格式
	if err := v.ValidateURL(urlStr); err != nil {
		return err
	}

	// 如果跳过可访问性检查，直接返回
	if skipAccessibilityCheck {
		return nil
	}

	// 检查可访问性
	accessible, statusCode, err := v.CheckURLAccessibility(urlStr)
	if err != nil {
		return fmt.Errorf("检查URL可访问性失败: %w", err)
	}

	if !accessible {
		return fmt.Errorf("URL不可访问（HTTP %d）", statusCode)
	}

	return nil
}

// IsExternalURL 检查是否是外部URL（非本地文件）
func IsExternalURL(urlStr string) bool {
	if len(urlStr) == 0 {
		return false
	}
	// 检查是否是本地路径
	if urlStr[0] == '/' || urlStr[0] == '.' {
		return false
	}
	// 检查是否包含本地路径特征
	localPatterns := []string{"file://", "./", "../", "\\"}
	return !ContainsAny(urlStr, localPatterns)
}
