// Package scbbsapi 封装 SCBBS 社区 API 客户端。
// 从 backend/modpack 包迁入（scbbs 整合包安装也需要访问该 API，为避免包间循环依赖抽出）。
package scbbsapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SCBBSApiClient SCBBS API 客户端
type SCBBSApiClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewSCBBSApiClient 创建 SCBBS API 客户端
func NewSCBBSApiClient() *SCBBSApiClient {
	return NewSCBBSApiClientWithBaseURL("https://m.suancaixianyu.cn/api")
}

// NewSCBBSApiClientWithBaseURL 创建指定 baseURL 的 SCBBS API 客户端（测试用）
func NewSCBBSApiClientWithBaseURL(baseURL string) *SCBBSApiClient {
	return &SCBBSApiClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// PostDetailResponse 文章详情响应
type PostDetailResponse struct {
	Code int      `json:"code"`
	Data PostData `json:"data"`
	Msg  string   `json:"msg"`
}

// PostData 文章数据
type PostData struct {
	ID           int           `json:"id"`
	Title        string        `json:"title"`
	PostVersions []PostVersion `json:"postVersions"`
}

// PostVersion 版本信息
type PostVersion struct {
	ID        int       `json:"id"`
	Version   string    `json:"version"`
	Files     []ModFile `json:"files"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

// ModFile 模组文件信息
type ModFile struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     string `json:"size"`
	Hash     string `json:"hash"`
}

// GetModVersions 获取模组的版本信息
func (c *SCBBSApiClient) GetModVersions(projectID int) (*PostDetailResponse, error) {
	reqURL := fmt.Sprintf("%s/post/detail?id=%d", c.baseURL, projectID)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("请求SCBBS API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCBBS API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var result PostDetailResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("SCBBS API返回错误: %s (code: %d)", result.Msg, result.Code)
	}

	return &result, nil
}

// FindMatchingVersion 查找匹配的版本
// 如果 version 为 "latest"，返回最新的版本
// 否则返回匹配指定版本号的版本
func (c *SCBBSApiClient) FindMatchingVersion(versions []PostVersion, version string) (*PostVersion, *ModFile, error) {
	if len(versions) == 0 {
		return nil, nil, fmt.Errorf("该模组没有任何版本")
	}

	// 如果指定了 latest，返回最新版本
	if version == "latest" || version == "" {
		// 按更新时间倒序排序，取第一个
		latest := versions[0]
		for _, v := range versions[1:] {
			if v.UpdatedAt > latest.UpdatedAt {
				latest = v
			}
		}

		if len(latest.Files) == 0 {
			return nil, nil, fmt.Errorf("最新版本没有可用的文件")
		}

		return &latest, &latest.Files[0], nil
	}

	// 查找匹配的版本
	for i := range versions {
		if versions[i].Version == version {
			if len(versions[i].Files) == 0 {
				return nil, nil, fmt.Errorf("版本 %s 没有可用的文件", version)
			}
			return &versions[i], &versions[i].Files[0], nil
		}
	}

	return nil, nil, fmt.Errorf("未找到版本 %s", version)
}

// UploadFileInfo 社区上传文件元数据（GET /upload/info?id=）
type UploadFileInfo struct {
	ID             int    `json:"id"`
	Filename       string `json:"filename"`
	URL            string `json:"url"`
	PackageName    string `json:"packageName"`
	PackageVersion string `json:"packageVersion"`
}

// FetchUploadInfo 按 FileId 反查社区文件元数据（对齐手机版 fetchFileInfoById /
// ScbbsPlus FetchIntegrationPackFileInfoByIdAsync）。
// 响应可能是 {code,data:{...}} 包装，也可能直接是文件对象本身；
// 字段名大小写不敏感（服务端 PascalCase/camelCase 均兼容）。
func (c *SCBBSApiClient) FetchUploadInfo(fileID string) (*UploadFileInfo, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil, fmt.Errorf("FileId 不能为空")
	}

	reqURL := fmt.Sprintf("%s/upload/info?id=%s", c.baseURL, url.QueryEscape(fileID))
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("请求SCBBS API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SCBBS API返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 先按 {code,data} 包装尝试
	var wrapped struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	var raw json.RawMessage = body
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		if wrapped.Code != 0 && wrapped.Code != 200 {
			return nil, fmt.Errorf("SCBBS API返回错误: %s (code: %d)", wrapped.Msg, wrapped.Code)
		}
		raw = wrapped.Data
	}

	// 大小写不敏感读取字段
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	info := &UploadFileInfo{}
	info.ID = int(readJSONNumber(fields, "id"))
	info.Filename = readJSONString(fields, "filename")
	info.URL = readJSONString(fields, "url")
	info.PackageName = readJSONString(fields, "packageName")
	info.PackageVersion = readJSONString(fields, "packageVersion")

	if info.URL == "" {
		return nil, fmt.Errorf("社区文件 %s 没有可用的下载链接", fileID)
	}

	return info, nil
}

// readJSONString 大小写不敏感地读取字符串字段
func readJSONString(fields map[string]json.RawMessage, key string) string {
	raw, ok := lookupField(fields, key)
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// readJSONNumber 大小写不敏感地读取数字字段
func readJSONNumber(fields map[string]json.RawMessage, key string) float64 {
	raw, ok := lookupField(fields, key)
	if !ok {
		return 0
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0
	}
	return n
}

// lookupField 按 key 大小写不敏感查找 JSON 字段
func lookupField(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	if raw, ok := fields[key]; ok {
		return raw, true
	}
	lower := strings.ToLower(key)
	for k, v := range fields {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return nil, false
}
