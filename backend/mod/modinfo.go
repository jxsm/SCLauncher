package mod

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// ModInfo 模组信息（来自模组包内的 modinfo.json）
// JSON 标签为小驼峰，便于前端直接消费；解析时按 PascalCase/小驼峰大小写不敏感读取。
type ModInfo struct {
	Name                string       `json:"name"`                // 模组名称
	Version             string       `json:"version"`             // 模组版本
	ApiVersion          string       `json:"apiVersion"`          // 适配的 API 版本（< 1.8 游戏会警告）
	PackageName         string       `json:"packageName"`         // 包名，唯一标识
	Description         string       `json:"description"`         // 描述
	ScVersion           string       `json:"scVersion"`           // 适配的游戏版本（无实际作用）
	LoadOrder           int          `json:"loadOrder"`           // 加载顺序，越小越先加载
	NonPersistentMod    bool         `json:"nonPersistentMod"`    // 非持久化（不写入存档）
	GameplayImpactLevel string       `json:"gameplayImpactLevel"` // 玩法影响等级：Cosmetic/Assist/Turbo/Break/Godmode
	Link                string       `json:"link"`                // 模组链接
	Author              string       `json:"author"`              // 作者
	Dependencies        []Dependency `json:"dependencies"`        // 依赖的其他模组
}

// Dependency 模组依赖
type Dependency struct {
	PackageName  string `json:"packageName"`            // 依赖模组的包名
	VersionRange string `json:"versionRange"`           // 版本范围约束（空字符串表示任意）
	DisplayName  string `json:"displayName,omitempty"`  // 显示名（可选）
}

// maxModInfoSize 限制读取的 modinfo.json 大小，避免异常大文件耗尽内存
const maxModInfoSize = 1 << 20 // 1 MiB

// parseModInfoFromModFile 从 .scmod/.zip 文件中解析 modinfo.json。
// 全程 best-effort：任何错误（非 zip、无 modinfo.json、JSON 非法）都返回 nil，永不 panic。
func parseModInfoFromModFile(fullPath string) (info *ModInfo) {
	defer func() {
		if r := recover(); r != nil {
			info = nil
		}
	}()

	reader, err := zip.OpenReader(fullPath)
	if err != nil {
		return nil
	}
	defer reader.Close()

	// 找到最浅（优先根目录）的 modinfo.json，大小写不敏感
	var best *zip.File
	bestDepth := -1
	for _, f := range reader.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if base != "modinfo.json" {
			continue
		}
		dir := strings.TrimSuffix(f.Name, filepath.Base(f.Name))
		depth := strings.Count(dir, "/")
		if best == nil || depth < bestDepth {
			best = f
			bestDepth = depth
		}
	}
	if best == nil {
		return nil
	}

	rc, err := best.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxModInfoSize))
	if err != nil {
		return nil
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}

	return parseModInfoJSON(data)
}

// parseModInfoJSON lenient 地解析 modinfo.json 字节
func parseModInfoJSON(data []byte) *ModInfo {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	info := &ModInfo{Dependencies: []Dependency{}}
	info.Name = getStringField(raw, "Name")
	info.Version = getStringField(raw, "Version")
	info.ApiVersion = getStringField(raw, "ApiVersion")
	info.PackageName = getStringField(raw, "PackageName")
	info.Description = getStringField(raw, "Description")
	info.ScVersion = getStringField(raw, "ScVersion", "GameVersion", "SurvivalcraftVersion")
	info.Link = getStringField(raw, "Link", "Homepage", "Website", "Url")
	info.Author = getStringField(raw, "Author", "Authors")
	info.LoadOrder = getIntField(raw, 0, "LoadOrder")
	info.NonPersistentMod = getBoolField(raw, false, "NonPersistentMod")
	info.GameplayImpactLevel = getStringField(raw, "GameplayImpactLevel")
	info.Dependencies = parseDependenciesField(raw)
	return info
}

// findRaw 在 JSON 对象中按候选名大小写不敏感查找字段
func findRaw(m map[string]json.RawMessage, names ...string) (json.RawMessage, bool) {
	lower := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		lower[strings.ToLower(k)] = v
	}
	for _, n := range names {
		if v, ok := lower[strings.ToLower(n)]; ok {
			return v, true
		}
	}
	return nil, false
}

func getStringField(m map[string]json.RawMessage, names ...string) string {
	v, ok := findRaw(m, names...)
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	// 容错：值可能是数字等其他类型
	var any interface{}
	if err := json.Unmarshal(v, &any); err == nil {
		if b, err := json.Marshal(any); err == nil {
			return strings.Trim(string(b), "\"")
		}
	}
	return ""
}

func getIntField(m map[string]json.RawMessage, def int, names ...string) int {
	v, ok := findRaw(m, names...)
	if !ok {
		return def
	}
	var n int
	if err := json.Unmarshal(v, &n); err == nil {
		return n
	}
	var f float64
	if err := json.Unmarshal(v, &f); err == nil {
		return int(f)
	}
	// 容错：值可能是字符串编码的数字（如 "10000"）
	if s := rawTextToString(v); s != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return int(f)
		}
	}
	return def
}

func getBoolField(m map[string]json.RawMessage, def bool, names ...string) bool {
	v, ok := findRaw(m, names...)
	if !ok {
		return def
	}
	var b bool
	if err := json.Unmarshal(v, &b); err == nil {
		return b
	}
	// 容错：值可能是字符串（如 "true"/"false"）
	if s := rawTextToString(v); s != "" {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return def
}

// rawTextToString 把 JSON 原始值转为字符串，兼容字符串/数字/布尔。
func rawTextToString(v json.RawMessage) string {
	v = bytes.TrimSpace(v)
	if len(v) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	var any interface{}
	if err := json.Unmarshal(v, &any); err == nil {
		b, _ := json.Marshal(any)
		str := string(b)
		if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
			str = str[1 : len(str)-1]
		}
		return str
	}
	return ""
}

// parseDependenciesField 从 modinfo.json 中读取依赖，接受多种字段名与多种数据形态。
// 参考实现：scbbs-plus-mod 的 LocalResources.cs（GetModDependencies/ReadJsonModDependencies/ReadDependencyTextList）。
func parseDependenciesField(m map[string]json.RawMessage) []Dependency {
	depRaw, ok := findRaw(m, "Dependencies", "DependencyRanges", "Dependency", "RequiredMods", "References")
	if !ok {
		return []Dependency{}
	}
	deps := readDependencies(depRaw)

	// 按包名大小写不敏感去重，保留首个非空版本
	out := make([]Dependency, 0, len(deps))
	indexByKey := map[string]int{}
	for _, d := range deps {
		key := strings.ToLower(strings.TrimSpace(d.PackageName))
		if key == "" {
			continue
		}
		if idx, exists := indexByKey[key]; exists {
			if out[idx].VersionRange == "" && d.VersionRange != "" {
				out[idx].VersionRange = d.VersionRange
			}
			continue
		}
		indexByKey[key] = len(out)
		out = append(out, d)
	}
	if out == nil {
		out = []Dependency{}
	}
	return out
}

// readDependencies 根据 JSON 形态分派解析依赖：
//   - 对象映射 { "pkg": "1.0" }
//   - 单个依赖对象 { "PackageName": "pkg", "Version": "1.0" }（未包裹在数组里）
//   - 数组 ["pkg:1.0"] 或 [{ "PackageName": "pkg", "Version": "1.0" }]
//   - 裸字符串 "pkg:1.0,pkg2:2.0"
func readDependencies(raw json.RawMessage) []Dependency {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}

	// 对象：可能是 pkg->version 映射，也可能是单个依赖描述对象
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &asMap); err == nil {
		// 若对象带有依赖描述字段（PackageName 等），视为单个依赖而非映射
		if _, isDep := findRaw(asMap, "PackageName", "Package", "Key", "Id", "Name", "Version", "VersionRange", "Range"); isDep {
			d := readDependencyElement(trimmed)
			if strings.TrimSpace(d.PackageName) != "" {
				return []Dependency{d}
			}
			return []Dependency{}
		}
		out := make([]Dependency, 0, len(asMap))
		for pkg, v := range asMap {
			pkg = strings.TrimSpace(pkg)
			if pkg == "" {
				continue
			}
			// 版本值可能是字符串或数字（如 1.0），rawTextToString 兼容两者
			out = append(out, Dependency{PackageName: pkg, VersionRange: strings.TrimSpace(rawTextToString(v))})
		}
		return out
	}

	// 数组
	var asArr []json.RawMessage
	if err := json.Unmarshal(trimmed, &asArr); err == nil {
		out := make([]Dependency, 0, len(asArr))
		for _, el := range asArr {
			d := readDependencyElement(el)
			if strings.TrimSpace(d.PackageName) != "" {
				out = append(out, d)
			}
		}
		return out
	}

	// 裸字符串
	var asStr string
	if err := json.Unmarshal(trimmed, &asStr); err == nil {
		return readDependencyTextList(asStr)
	}
	return nil
}

// readDependencyElement 解析数组中的单个依赖元素（字符串或对象）
func readDependencyElement(raw json.RawMessage) Dependency {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return Dependency{}
	}
	// 字符串元素："pkg:ver"
	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		return parseDepToken(s)
	}
	// 对象元素：{ PackageName, Version }
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &obj); err == nil {
		pkg := getStringField(obj, "PackageName", "Package", "Key", "Id", "Name")
		return Dependency{
			PackageName:  pkg,
			VersionRange: strings.TrimSpace(getStringField(obj, "Version", "VersionRange", "Range", "MinVersion", "RequiredVersion")),
		}
	}
	return Dependency{}
}

// readDependencyTextList 解析逗号/分号/换行分隔的依赖文本列表。
// 使用括号深度感知的切分，避免切到版本范围（如 [1.0,2.0)）内部的逗号。
func readDependencyTextList(s string) []Dependency {
	out := []Dependency{}
	for _, p := range splitDepTokens(s) {
		d := parseDepToken(p)
		if d.PackageName != "" {
			out = append(out, d)
		}
	}
	return out
}

// splitDepTokens 在顶层按 ','、';'、'\n' 切分，忽略出现在 [...] / (...) / <...> 版本范围括号内的分隔符。
func splitDepTokens(s string) []string {
	var parts []string
	depth := 0
	start := 0
	flush := func(end int) {
		parts = append(parts, s[start:end])
	}
	for i, r := range s {
		switch r {
		case '[', '(', '<':
			depth++
		case ']', ')', '>':
			if depth > 0 {
				depth--
			}
		case ',', ';', '\n':
			if depth == 0 {
				flush(i)
				start = i + len(string(r))
			}
		}
	}
	flush(len(s))
	return parts
}

// parseDepToken 解析单个 "pkg:ver" 或 "pkg" token
func parseDepToken(token string) Dependency {
	token = strings.TrimSpace(token)
	if token == "" {
		return Dependency{}
	}
	if idx := strings.Index(token, ":"); idx >= 0 {
		return Dependency{
			PackageName:  strings.TrimSpace(token[:idx]),
			VersionRange: strings.TrimSpace(token[idx+1:]),
		}
	}
	return Dependency{PackageName: token}
}
