// Package scbbs 实现 scbbs（ScbbsPlus 社区）格式整合包的解析与安装。
//
// 格式定义来自游戏内模组 scbbs-plus-mod（参考项目 E:\IT\参考项目\scbbs-plus-mod），
// 手机版启动器（Flutter）已支持同格式，本包行为与其对齐：
//   - 载体两种：ZIP 包（根目录 packinfo.json + 其余条目为内嵌文件）或纯 JSON 清单
//   - 清单字段 Name/PackageName/Version/Description/Icon/Items，键名大小写不敏感，
//     允许注释、尾逗号与 UTF-8 BOM
//   - 子项按 TargetDirectory 根段分流：mods/worlds/texturepacks/characterskins/
//     furniturepacks 为五类标准资源，其余为自定义数据文件
//   - 无游戏本体配置，安装目标是启动器中已安装的某个版本
package scbbs

import (
	"encoding/json"
	"net/url"
	"path"
	"strings"
)

// ItemType 子项资源类型（由 TargetDirectory 根段决定）
type ItemType string

const (
	ItemTypeMod       ItemType = "mod"       // 模组 → Mods/NetMods
	ItemTypeWorld     ItemType = "world"     // 存档 → Worlds（解压为世界文件夹）
	ItemTypeTexture   ItemType = "texture"   // 材质 → BlocksTextures
	ItemTypeSkin      ItemType = "skin"      // 皮肤 → CharacterSkins
	ItemTypeFurniture ItemType = "furniture" // 家具包 → FurniturePacks
	ItemTypeData      ItemType = "data"      // 自定义数据文件（净化后写入版本目录）
)

// Manifest scbbs 整合包清单
type Manifest struct {
	Name        string  // 显示名
	PackageName string  // 包名（唯一标识）
	Version     string  // 整合包版本
	Description string  // 描述
	Icon        string  // 图标：http(s) 链接或 ZIP 包内相对路径（仅 png/webp）
	Items       []*Item // 子项列表
}

// Item 整合包子项
type Item struct {
	Category        string // 旧版类别字段，仅在未填 TargetDirectory 时兜底
	FileName        string // 显式指定的落地文件名
	URL             string // http(s) 直链，或 ZIP 包内相对路径
	FileID          string // 社区文件 ID（JSON 中可能是数字或字符串）
	PackageName     string // 模组期望包名（下载后与 modinfo.json 比对，防链接调包）
	Version         string // 模组期望版本（下载后比对）
	TargetPath      string // 旧版字段：完整目标路径，缺 TargetDirectory 时反推
	TargetDirectory string // 推荐字段：目标目录，根段决定资源类型
	IsEnabled       bool   // 运行时开关，禁用项不参与安装

	// 以下为解析期填充
	HasExplicitFileName bool   // 清单是否显式指定了 FileName
	ManifestFileName    string // 显式文件名（仅 basename）
}

// EnabledItems 参与安装的子项
func (m *Manifest) EnabledItems() []*Item {
	enabled := make([]*Item, 0, len(m.Items))
	for _, item := range m.Items {
		if item.IsEnabled {
			enabled = append(enabled, item)
		}
	}
	return enabled
}

// DisplayName 条目显示名（信息弹窗/进度消息用）
func (i *Item) DisplayName() string {
	if name := strings.TrimSpace(i.FileName); name != "" {
		return name
	}
	if tail := URLTailName(i.URL); tail != "" {
		return tail
	}
	if id := strings.TrimSpace(i.FileID); id != "" {
		return "#" + id
	}
	if p := strings.TrimSpace(i.TargetPath); p != "" {
		return path.Base(strings.ReplaceAll(p, "\\", "/"))
	}
	return "(未命名)"
}

// ResolveType 解析资源类型（对齐 ScbbsPlus ResolveIntegrationPackItemContentType）：
// TargetDirectory 根段优先——五个标准目录之外一律视为自定义数据文件；
// 未填写 TargetDirectory 时按旧版 Category 兜底。
func (i *Item) ResolveType() ItemType {
	dir := strings.ReplaceAll(strings.TrimSpace(i.TargetDirectory), "\\", "/")
	for _, segment := range strings.Split(dir, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		switch strings.ToLower(segment) {
		case "mods":
			return ItemTypeMod
		case "worlds":
			return ItemTypeWorld
		case "texturepacks":
			return ItemTypeTexture
		case "characterskins":
			return ItemTypeSkin
		case "furniturepacks":
			return ItemTypeFurniture
		default:
			return ItemTypeData
		}
	}

	switch strings.ToLower(strings.TrimSpace(i.Category)) {
	case "world", "scworld":
		return ItemTypeWorld
	case "furniture", "scfpack":
		return ItemTypeFurniture
	case "texture", "scbtex", "png":
		return ItemTypeTexture
	case "skin", "scskin":
		return ItemTypeSkin
	case "mod", "scmod":
		return ItemTypeMod
	default:
		return ItemTypeData
	}
}

// InferFileName 推断落地文件名（优先级对齐 ScbbsPlus EnsureIntegrationPackItemFileName）：
// 显式 FileName > URL 尾段 > TargetPath 文件名。
// FileId 项由安装器在获取社区文件元数据后回填。
func (i *Item) InferFileName() string {
	if i.HasExplicitFileName && i.ManifestFileName != "" {
		return i.ManifestFileName
	}
	if tail := URLTailName(i.URL); tail != "" {
		return tail
	}
	if p := strings.TrimSpace(i.TargetPath); p != "" {
		return path.Base(strings.ReplaceAll(p, "\\", "/"))
	}
	return ""
}

// HasHTTPURL Url 是否为 http(s) 链接
func (i *Item) HasHTTPURL() bool {
	return isHTTPURL(i.URL)
}

// isHTTPURL 判断字符串是否为 http(s) 链接
func isHTTPURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

// URLTailName 提取 URL / 相对路径的尾段文件名（对齐手机版 _urlTailName）：
// http(s) 链接先做 URL 解码取路径尾段，否则直接取最后一个非空 "/" 段。
func URLTailName(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if normalized == "" {
		return ""
	}
	if u, err := url.Parse(normalized); err == nil && u.Scheme != "" {
		if decoded, err := url.PathUnescape(u.EscapedPath()); err == nil {
			normalized = strings.ReplaceAll(decoded, "\\", "/")
		} else {
			normalized = strings.ReplaceAll(u.Path, "\\", "/")
		}
	}
	segments := strings.Split(normalized, "/")
	for idx := len(segments) - 1; idx >= 0; idx-- {
		trimmed := strings.TrimSpace(segments[idx])
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// NormalizeArchivePath 归一化压缩包条目路径（解析器与安装器共用）：
// 反斜杠转正斜杠、去前导斜杠、剔除 "." / 空段；包含 ".." 返回空串（拒绝路径穿越）。
// 返回值用于内嵌文件索引（统一转小写做大小写不敏感匹配，对齐 ScbbsPlus 的
// OrdinalIgnoreCase 字典）。
func NormalizeArchivePath(p string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	normalized = strings.TrimLeft(normalized, "/")
	segments := make([]string, 0, 8)
	for _, raw := range strings.Split(normalized, "/") {
		segment := strings.TrimSpace(raw)
		if segment == ".." {
			return ""
		}
		if segment == "" || segment == "." {
			continue
		}
		segments = append(segments, segment)
	}
	return strings.Join(segments, "/")
}

// manifestFromJSON 从（已剥离注释/尾逗号的）JSON 对象解析清单，字段名大小写不敏感
func manifestFromJSON(root map[string]json.RawMessage) *Manifest {
	manifest := &Manifest{
		Name:        readStringField(root, "Name"),
		PackageName: readStringField(root, "PackageName"),
		Version:     readStringField(root, "Version"),
		Description: readStringField(root, "Description"),
		Icon:        readStringField(root, "Icon"),
	}
	if itemsRaw, ok := lookupField(root, "Items"); ok {
		var arr []json.RawMessage
		if err := json.Unmarshal(itemsRaw, &arr); err == nil {
			for _, raw := range arr {
				if item, err := itemFromJSON(raw); err == nil && item != nil {
					manifest.Items = append(manifest.Items, item)
				}
			}
		}
	}
	return manifest
}

// itemFromJSON 从 JSON 对象解析子项，字段名大小写不敏感
func itemFromJSON(data []byte) (*Item, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	item := &Item{
		Category:        readStringField(fields, "Category"),
		FileName:        readStringField(fields, "FileName"),
		URL:             readStringField(fields, "Url"),
		FileID:          readFlexibleStringField(fields, "FileId"),
		PackageName:     readStringField(fields, "PackageName"),
		Version:         readStringField(fields, "Version"),
		TargetPath:      readStringField(fields, "TargetPath"),
		TargetDirectory: readStringField(fields, "TargetDirectory"),
		IsEnabled:       true,
	}
	if raw, ok := lookupField(fields, "IsEnabled"); ok {
		var b bool
		if err := json.Unmarshal(raw, &b); err == nil {
			item.IsEnabled = b
		}
	}

	item.HasExplicitFileName = item.FileName != ""
	if item.HasExplicitFileName {
		item.ManifestFileName = path.Base(strings.ReplaceAll(item.FileName, "\\", "/"))
	}
	return item, nil
}

// readStringField 大小写不敏感地读取字符串字段（前后去空白）
func readStringField(fields map[string]json.RawMessage, key string) string {
	raw, ok := lookupField(fields, key)
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

// readFlexibleStringField 兼容字符串/数字两种 JSON 类型读取字段
// （对应 ScbbsPlus 的 FlexibleStringJsonConverter，FileId 可能写成数字）
func readFlexibleStringField(fields map[string]json.RawMessage, key string) string {
	raw, ok := lookupField(fields, key)
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return ""
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
