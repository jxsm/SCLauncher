package scbbs

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"SCLauncher/backend/utils"
)

// ErrNotScbbsPack 表示文件不是 scbbs（ScbbsPlus）格式的整合包。
// 调用方（app 层）据此回退到其他整合包格式（v0.1 scmodpack）。
var ErrNotScbbsPack = errors.New("不是 scbbs 格式的整合包")

// 清单与图标的读取上限，防止异常大的条目耗尽内存
const (
	maxManifestSize = 1 << 20 // 1 MiB
	maxIconSize     = 4 << 20 // 4 MiB
)

// ParseResult 解析结果
type ParseResult struct {
	Manifest *Manifest
	FilePath string // 整合包文件路径

	// IsZipCarrier 是否为 ZIP 载体（packinfo.json + 内嵌文件）；
	// false 表示纯 JSON 清单载体（所有条目走网络/FileId）
	IsZipCarrier bool

	// MissingEmbedded 清单用相对路径 Url 引用了包内不存在、且无 FileId 兜底的条目
	// （解析阶段容忍，安装阶段逐项失败）
	MissingEmbedded []*Item

	// EmbeddedIconBytes / EmbeddedIconExt 包内图标字节与扩展名（png/webp），
	// 供前端直接展示（data URL）
	EmbeddedIconBytes []byte
	EmbeddedIconExt   string
}

// Parse 解析 scbbs 整合包。识别规则（对齐手机版 ScbbsPackParser.parseFromFile）：
//   - .json 文件 → JSON 清单载体，顶层必须有 Items 数组
//   - ZIP 文件（按内容识别，不限扩展名）→ 根目录必须有 packinfo.json
//
// 不是本格式返回 ErrNotScbbsPack；是本格式但内容无效返回具体校验错误。
func Parse(path string) (*ParseResult, error) {
	if strings.ToLower(filepath.Ext(path)) == ".json" {
		return parseJSONCarrier(path)
	}
	return parseZIPCarrier(path)
}

// parseJSONCarrier 解析纯 JSON 清单载体
func parseJSONCarrier(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取清单文件失败: %w", err)
	}
	data = preprocessManifestJSON(data)

	root, err := unmarshalJSONObject(data)
	if err != nil {
		return nil, fmt.Errorf("清单不是有效的 JSON 对象: %w", err)
	}

	// 嗅探：顶层必须有 Items 数组，否则不是 scbbs 清单
	itemsRaw, ok := lookupField(root, "Items")
	if !ok {
		return nil, ErrNotScbbsPack
	}
	var itemsArr []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &itemsArr); err != nil {
		return nil, ErrNotScbbsPack
	}

	manifest := manifestFromJSON(root)
	normalizeManifest(manifest)
	if err := validateManifest(manifest, false); err != nil {
		return nil, err
	}
	if _, err := validateItemsForCarrier(manifest, false, nil); err != nil {
		return nil, err
	}

	return &ParseResult{
		Manifest:     manifest,
		FilePath:     path,
		IsZipCarrier: false,
	}, nil
}

// parseZIPCarrier 解析 ZIP 载体（packinfo.json + 内嵌文件）
func parseZIPCarrier(path string) (*ParseResult, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		// 打不开的一律不是本格式，交给上层回退其他解析器
		return nil, ErrNotScbbsPack
	}
	defer reader.Close()

	// 第一遍：找根目录 packinfo.json + 建内嵌文件索引
	var manifestEntry *zip.File
	embedded := buildEmbeddedIndex(reader.File)
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := utils.DecodeArchiveEntryName(f.Name)
		if strings.EqualFold(NormalizeArchivePath(name), "packinfo.json") {
			manifestEntry = f
			break
		}
	}
	if manifestEntry == nil {
		return nil, ErrNotScbbsPack
	}

	data, err := readZipEntry(manifestEntry, maxManifestSize)
	if err != nil {
		return nil, fmt.Errorf("读取 packinfo.json 失败: %w", err)
	}
	data = preprocessManifestJSON(data)

	root, err := unmarshalJSONObject(data)
	if err != nil {
		return nil, fmt.Errorf("packinfo.json 不是有效的 JSON 对象: %w", err)
	}

	manifest := manifestFromJSON(root)
	normalizeManifest(manifest)
	if err := validateManifest(manifest, true); err != nil {
		return nil, err
	}

	result := &ParseResult{
		Manifest:     manifest,
		FilePath:     path,
		IsZipCarrier: true,
	}
	result.MissingEmbedded, err = validateItemsForCarrier(manifest, true, embedded)
	if err != nil {
		return nil, err
	}

	// 图标：Icon 为空时探测包内根目录 Icon.png/Icon.webp；
	// Icon 为包内相对路径时直接读取（对齐 ScbbsPlus 的默认图标探测）
	loadEmbeddedIcon(result, manifest, embedded)

	return result, nil
}

// buildEmbeddedIndex 构建内嵌文件索引：key 为归一化（小写）相对路径。
// 跳过目录条目、__MACOSX/.DS_Store 垃圾条目与含 ".." 的穿越条目；同名以先出现为准。
func buildEmbeddedIndex(files []*zip.File) map[string]*zip.File {
	index := make(map[string]*zip.File)
	for _, f := range files {
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			continue
		}
		name := utils.DecodeArchiveEntryName(f.Name)
		key := strings.ToLower(NormalizeArchivePath(name))
		if key == "" {
			continue
		}
		if key == ".ds_store" || strings.HasPrefix(key, "__macosx/") {
			continue
		}
		if _, exists := index[key]; !exists {
			index[key] = f
		}
	}
	return index
}

// loadEmbeddedIcon 读取包内图标字节填充 ParseResult
func loadEmbeddedIcon(result *ParseResult, manifest *Manifest, embedded map[string]*zip.File) {
	var candidates []string
	if manifest.Icon == "" {
		// 默认图标：包内根目录 Icon.png / Icon.webp
		candidates = []string{"icon.png", "icon.webp"}
	} else if !isHTTPURL(manifest.Icon) {
		// Icon 指向包内相对路径
		candidates = []string{strings.ToLower(NormalizeArchivePath(manifest.Icon))}
	}

	for _, key := range candidates {
		if key == "" {
			continue
		}
		if f, ok := embedded[key]; ok {
			if data, err := readZipEntry(f, maxIconSize); err == nil && len(data) > 0 {
				result.EmbeddedIconBytes = data
				result.EmbeddedIconExt = strings.TrimPrefix(filepath.Ext(key), ".")
				return
			}
		}
	}
}

// normalizeManifest 归一化清单（对齐手机版 _normalizeManifest /
// ScbbsPlus NormalizeIntegrationPackManifest）：只有旧字段 TargetPath 的条目
// 反推出 TargetDirectory 与文件名。
func normalizeManifest(manifest *Manifest) {
	for _, item := range manifest.Items {
		if strings.TrimSpace(item.TargetDirectory) == "" && strings.TrimSpace(item.TargetPath) != "" {
			normalized := strings.ReplaceAll(item.TargetPath, "\\", "/")
			index := strings.LastIndex(normalized, "/")
			if index > 0 {
				item.TargetDirectory = strings.TrimSpace(normalized[:index])
			} else {
				item.TargetDirectory = strings.TrimSpace(normalized)
			}
			if strings.TrimSpace(item.FileName) == "" {
				if index < 0 {
					item.FileName = normalized
				} else {
					item.FileName = strings.TrimSpace(normalized[index+1:])
				}
			}
		}
	}
}

// validateManifest 校验清单级必填字段（对齐 ScbbsPlus IsValidIntegrationPackManifest）
func validateManifest(manifest *Manifest, isZipCarrier bool) error {
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("整合包名称 (Name) 不能为空")
	}
	if strings.TrimSpace(manifest.PackageName) == "" {
		return fmt.Errorf("整合包包名 (PackageName) 不能为空")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("整合包版本 (Version) 不能为空")
	}
	if strings.TrimSpace(manifest.Description) == "" {
		return fmt.Errorf("整合包描述 (Description) 不能为空")
	}
	if len(manifest.Items) == 0 {
		return fmt.Errorf("整合包子项列表 (Items) 不能为空")
	}

	if manifest.Icon != "" {
		lower := strings.ToLower(manifest.Icon)
		if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".webp") {
			return fmt.Errorf("图标 (Icon) 仅支持 .png 或 .webp: %s", manifest.Icon)
		}
		if !isZipCarrier && !isHTTPURL(manifest.Icon) {
			// JSON 清单没有包内文件，图标必须是 http(s) 链接
			return fmt.Errorf("JSON 清单的图标 (Icon) 必须是 http(s) 链接: %s", manifest.Icon)
		}
	}
	return nil
}

// validateItemsForCarrier 按载体校验每个子项来源
// （对齐手机版 _validateForFormat / ScbbsPlus TryValidateIntegrationPackItemSource）。
// ZIP 载体中引用了包内不存在文件且无 FileId 兜底的条目记入返回的缺失列表（不报错）。
func validateItemsForCarrier(manifest *Manifest, isZipCarrier bool, embedded map[string]*zip.File) ([]*Item, error) {
	var missing []*Item
	for _, item := range manifest.Items {
		if strings.TrimSpace(item.TargetDirectory) == "" {
			return nil, fmt.Errorf("子项 %q 未指定 TargetDirectory", item.DisplayName())
		}
		url := strings.TrimSpace(item.URL)
		fileID := strings.TrimSpace(item.FileID)
		if url == "" && fileID == "" {
			return nil, fmt.Errorf("子项 %q 的 Url 与 FileId 不能同时为空", item.DisplayName())
		}
		if url == "" || isHTTPURL(url) {
			continue
		}
		// 相对路径来源：仅 ZIP 载体支持（内嵌文件）
		if !isZipCarrier {
			if fileID != "" {
				continue
			}
			return nil, fmt.Errorf("JSON 清单的子项 %q 不支持相对路径 Url（需为 http(s) 链接或填写 FileId）", item.DisplayName())
		}
		key := strings.ToLower(NormalizeArchivePath(url))
		if key != "" {
			if _, ok := embedded[key]; ok {
				continue
			}
		}
		if fileID != "" {
			continue
		}
		// 包内缺失：解析阶段容忍，安装阶段逐项失败
		missing = append(missing, item)
	}
	return missing, nil
}

// preprocessManifestJSON 预处理清单 JSON 字节：去 UTF-8 BOM、剥离 // 与 /* */ 注释、
// 去除 } / ] 前的尾逗号（对齐 ScbbsPlus s_packManifestJsonOptions 的宽松解析：
// ReadCommentHandling.Skip + AllowTrailingCommas）。字符串内部的内容原样保留。
func preprocessManifestJSON(data []byte) []byte {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var out bytes.Buffer
	out.Grow(len(data))
	inString := false
	escaped := false

	for i := 0; i < len(data); {
		c := data[i]
		if inString {
			out.WriteByte(c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			i++
			continue
		}
		switch c {
		case '"':
			inString = true
			out.WriteByte(c)
			i++
		case '/':
			if i+1 < len(data) && data[i+1] == '/' {
				// 行注释：跳到行尾
				for i < len(data) && data[i] != '\n' {
					i++
				}
			} else if i+1 < len(data) && data[i+1] == '*' {
				i += 2
				for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
					i++
				}
				if i+1 < len(data) {
					i += 2
				} else {
					i = len(data)
				}
			} else {
				out.WriteByte(c)
				i++
			}
		case ',':
			// 尾逗号：后面（跳过空白）若是 } 或 ] 则丢弃
			j := i + 1
			for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\r' || data[j] == '\n') {
				j++
			}
			if j < len(data) && (data[j] == '}' || data[j] == ']') {
				i++
			} else {
				out.WriteByte(c)
				i++
			}
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.Bytes()
}

// unmarshalJSONObject 将 JSON 字节解析为对象 map
func unmarshalJSONObject(data []byte) (map[string]json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return root, nil
}

// readZipEntry 读取 zip 条目内容（带大小上限）
func readZipEntry(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, limit))
}
