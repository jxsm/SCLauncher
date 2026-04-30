package modpack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SCBBSApiClient SCBBS API 客户端
type SCBBSApiClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewSCBBSApiClient 创建 SCBBS API 客户端
func NewSCBBSApiClient() *SCBBSApiClient {
	return &SCBBSApiClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://m.suancaixianyu.cn/api",
	}
}

// PostDetailResponse 文章详情响应
type PostDetailResponse struct {
	Code int        `json:"code"`
	Data PostData   `json:"data"`
	Msg  string     `json:"msg"`
}

// PostData 文章数据
type PostData struct {
	ID             int                `json:"id"`
	Title          string             `json:"title"`
	PostVersions   []PostVersion      `json:"postVersions"`
}

// PostVersion 版本信息
type PostVersion struct {
	ID      int        `json:"id"`
	Version string     `json:"version"`
	Files   []ModFile  `json:"files"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
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
	url := fmt.Sprintf("%s/post/detail?id=%d", c.baseURL, projectID)

	resp, err := c.httpClient.Get(url)
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
	// 支持 "latest" 和 "lastest"（兼容旧文档的拼写错误）
	if version == "latest" || version == "lastest" || version == "" {
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
