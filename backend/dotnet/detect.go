// Package dotnet 负责检测与补齐游戏所需的 .NET Desktop Runtime。
//
// 该包刻意不依赖 wails / 数据库 / 配置，所有外部副作用（命令执行、HTTP、文件系统）
// 都通过参数或接口注入，便于单元测试。
package dotnet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// RequiredRuntime 描述一次探测的结果。
type RequiredRuntime struct {
	// Needed 为 true 表示游戏依赖一个需要系统安装的 .NET Desktop Runtime。
	// 为 false 时可能是：旧版原生游戏、自包含（self-contained）发布、或无依据。
	Needed bool
	// MajorVersion 所需运行时大版本，例如 10、9。Needed=false 时为 0。
	MajorVersion int
	// Source 探测来源："runtimeconfig" | "deps" | "self-contained" | "none"。
	Source string
	// TFM 原始目标框架名，例如 "net10.0"。
	TFM string
}

// runtimeConfigFile 对应 *.runtimeconfig.json，只声明关心的字段。
type runtimeConfigFile struct {
	RuntimeOptions struct {
		TFM        string `json:"tfm"`
		Frameworks []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"frameworks"`
	} `json:"runtimeOptions"`
}

// depsFile 对应 *.deps.json，只声明关心的字段。
type depsFile struct {
	RuntimeTarget struct {
		Name string `json:"name"`
	} `json:"runtimeTarget"`
}

// depsVersionRe 匹配 runtimeTarget.name 中的 "v10." 形式主版本。
var depsVersionRe = regexp.MustCompile(`v(\d+)\.`)

// DetectRequired 在游戏目录下探测所需运行时。
//
// 优先级：
//  1. *.runtimeconfig.json 的 runtimeOptions.tfm（若 frameworks 为空则视为自包含）
//  2. *.deps.json 的 runtimeTarget.name
//  3. 都没有 → Needed=false
func DetectRequired(gameDir string) (RequiredRuntime, error) {
	res := RequiredRuntime{Source: SourceNone}

	// 1. 优先 runtimeconfig.json：只要能解析出 tfm（netX.Y）即视为需要系统运行时。
	//    与原始需求一致（读 runtimeOptions.tfm）。不再因 frameworks 为空而判定为自包含跳过——
	//    否则框架依赖型游戏的 runtimeconfig 若未显式列出 frameworks 会被漏检，导致不提示直接启动。
	if _, data, err := findFileBySuffix(gameDir, ".runtimeconfig.json"); err == nil {
		tfm, _, err := parseRuntimeConfig(data)
		if err == nil && tfm != "" {
			if major, ok := tfmToMajor(tfm); ok {
				res.Needed = true
				res.MajorVersion = major
				res.Source = SourceRuntimeConfig
				res.TFM = tfm
				return res, nil
			}
		}
	}

	// 2. 回退 deps.json
	if _, data, err := findFileBySuffix(gameDir, ".deps.json"); err == nil {
		if major, ok := parseDepsRuntimeTarget(data); ok {
			res.Needed = true
			res.MajorVersion = major
			res.Source = SourceDeps
			return res, nil
		}
	}

	return res, nil
}

// 探测来源常量。
const (
	SourceNone          = "none"
	SourceRuntimeConfig = "runtimeconfig"
	SourceDeps          = "deps"
	SourceSelfContained = "self-contained"
)

// parseRuntimeConfig 解析 runtimeconfig.json，返回 tfm 与「是否依赖共享框架」。
func parseRuntimeConfig(data []byte) (tfm string, frameworkDependent bool, err error) {
	var cfg runtimeConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", false, err
	}
	return cfg.RuntimeOptions.TFM, len(cfg.RuntimeOptions.Frameworks) > 0, nil
}

// parseDepsRuntimeTarget 从 deps.json 的 runtimeTarget.name 提取主版本。
func parseDepsRuntimeTarget(data []byte) (int, bool) {
	var d depsFile
	if err := json.Unmarshal(data, &d); err != nil {
		return 0, false
	}
	return parseRuntimeTargetName(d.RuntimeTarget.Name)
}

// parseRuntimeTargetName 从 ".NETCoreApp,Version=v10.0/win-x64" 提取主版本 10。
func parseRuntimeTargetName(name string) (int, bool) {
	m := depsVersionRe.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// tfmToMajor 将 "net10.0" 转为 10。
// 仅接受 "net<major>.<minor>" 形式，从而排除 "net48" 之类的 .NET Framework TFM
// 以及 "netcoreapp3.1" 这种旧式 TFM。
func tfmToMajor(tfm string) (int, bool) {
	s := strings.TrimPrefix(tfm, "net")
	if s == tfm || s == "" {
		return 0, false
	}
	dot := strings.IndexByte(s, '.')
	if dot <= 0 { // 必须有 "major." 形式
		return 0, false
	}
	major := s[:dot]
	n, err := strconv.Atoi(major)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// findFileBySuffix 在 gameDir（含子目录）查找第一个以 suffix 结尾的文件，
// 返回其路径与内容。找不到时返回 error。
func findFileBySuffix(gameDir, suffix string) (string, []byte, error) {
	var found string
	_ = filepath.Walk(gameDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), strings.ToLower(suffix)) {
			if found == "" {
				found = path
			}
		}
		return nil
	})
	if found == "" {
		return "", nil, fmt.Errorf("no file ending with %s", suffix)
	}
	data, err := os.ReadFile(found)
	if err != nil {
		return "", nil, err
	}
	return found, data, nil
}
