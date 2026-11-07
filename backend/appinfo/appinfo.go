package appinfo

import "encoding/json"

// Version of the application.
// 唯一来源是项目根目录 wails.json 的 info.productVersion 字段（也是 Wails
// 生成 exe 文件属性/安装包版本信息的来源），由 main.go 通过 //go:embed
// 嵌入并在启动时调用 SetWailsConfig 注入，改版本号只需改 wails.json 一处。
var Version = "0.0.0"

// RepoOwner GitHub repository owner
const RepoOwner = "jxsm"

// RepoName GitHub repository name
const RepoName = "SCLauncher"

// SetWailsConfig 从嵌入的 wails.json 内容中解析 info.productVersion 并设置
// Version。解析失败或字段缺失时保留当前值，保证配置手误不会导致启动异常。
func SetWailsConfig(data []byte) {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	if cfg.Info.ProductVersion != "" {
		Version = cfg.Info.ProductVersion
	}
}
