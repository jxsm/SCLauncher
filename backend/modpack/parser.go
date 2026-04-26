package modpack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"archive/zip"

	v0_1 "SCLauncher/backend/modpack/v0.1"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Parser 解析器接口
type Parser interface {
	// Parse 解析整合包
	Parse(modpackPath string) (any, error)
	// Validate 验证整合包
	Validate(manifest any) error
	// GetVersion 获取解析器版本
	GetVersion() string
}

// ParserRegistry 解析器注册表
type ParserRegistry struct {
	parsers map[string]Parser
}

// NewParserRegistry 创建解析器注册表
func NewParserRegistry() *ParserRegistry {
	registry := &ParserRegistry{
		parsers: make(map[string]Parser),
	}

	// 注册内置解析器
	registry.RegisterParser(v0_1.NewV01Parser())

	return registry
}

// RegisterParser 注册解析器
func (r *ParserRegistry) RegisterParser(parser Parser) {
	version := parser.GetVersion()
	r.parsers[version] = parser
}

// GetParser 获取指定版本的解析器
func (r *ParserRegistry) GetParser(version string) (Parser, error) {
	parser, exists := r.parsers[version]
	if !exists {
		return nil, fmt.Errorf("不支持的清单版本: %s", version)
	}
	return parser, nil
}

// ParseModpack 解析整合包（自动选择合适的解析器）
func (r *ParserRegistry) ParseModpack(modpackPath string) (*Manifest, error) {
	// 先读取清单文件获取版本信息
	manifest, err := readManifestFile(modpackPath)
	if err != nil {
		return nil, err
	}

	// 获取对应版本的解析器
	parser, err := r.GetParser(fmt.Sprintf("%.1f", manifest.ManifestVersion))
	if err != nil {
		return nil, err
	}

	// 使用解析器解析
	result, err := parser.Parse(modpackPath)
	if err != nil {
		return nil, err
	}

	// 根据解析器版本进行类型转换
	version := parser.GetVersion()
	switch version {
	case "0.1":
		// 将 v0.1 的 Manifest 转换为主包的 Manifest
		v1Manifest, ok := result.(*v0_1.V01Manifest)
		if !ok {
			return nil, fmt.Errorf("解析器返回了无效的类型")
		}
		return convertV01ToMain(v1Manifest), nil
	default:
		return nil, fmt.Errorf("不支持的解析器版本: %s", version)
	}
}

// convertV01ToMain 将 v0.1 的 Manifest 转换为主包的 Manifest
func convertV01ToMain(v1 *v0_1.V01Manifest) *Manifest {
	mods := make([]ModInfo, len(v1.Mods))
	for i, mod := range v1.Mods {
		mods[i] = ModInfo{
			ProjectID: mod.ProjectID,
			Version:   mod.Version,
			Name:      mod.Name,
			Required:  mod.Required,
			Path:      mod.Path,
		}
	}

	var survivalcraft *SurvivalcraftConfig
	if v1.Survivalcraft != nil {
		survivalcraft = &SurvivalcraftConfig{
			Version: VersionConfig{
				Android: convertPlatformVersion(v1.Survivalcraft.Version.Android),
				Windows: convertPlatformVersion(v1.Survivalcraft.Version.Windows),
			},
			VersionList: VersionListConfig{
				Android: v1.Survivalcraft.VersionList.Android,
				Windows: v1.Survivalcraft.VersionList.Windows,
			},
		}
	}

	// 检查是否有外部链接
	hasExternalLinks := checkExternalLinks(v1)

	return &Manifest{
		ManifestType:    v1.ManifestType,
		ManifestVersion: v1.ManifestVersion,
		Name:            v1.Name,
		Version:         v1.Version,
		Author:          v1.Author,
		Description:     v1.Description,
		Icon:            v1.Icon,
		Created:         v1.Created,
		Changelog:       v1.Changelog,
		Survivalcraft:   survivalcraft,
		Mods:            mods,
		Overrides:       v1.Overrides,
		Checksum:        v1.Checksum,
		FilePath:        v1.FilePath,
		FileHash:        v1.FileHash,
		HasExternalLinks: hasExternalLinks,
	}
}

// checkExternalLinks 检查整合包中是否包含外部链接
func checkExternalLinks(v1 *v0_1.V01Manifest) bool {
	// 检查 Windows 版本是否有外部下载链接
	if v1.Survivalcraft != nil && v1.Survivalcraft.Version.Windows != nil {
		if v1.Survivalcraft.Version.Windows.Path != "" {
			return true
		}
	}

	// 检查模组列表中是否有外部下载链接
	for _, mod := range v1.Mods {
		if mod.Path != "" {
			return true
		}
	}

	return false
}

// convertPlatformVersion 转换平台版本信息
func convertPlatformVersion(v *v0_1.V01PlatformVersion) *PlatformVersion {
	if v == nil {
		return nil
	}
	return &PlatformVersion{
		Version:        v.Version,
		APKPackageName: v.APKPackageName,
		Path:           v.Path,
	}
}

// readManifestFile 从整合包中读取清单文件
func readManifestFile(modpackPath string) (*Manifest, error) {
	// 打开 zip 文件（.scmodpack 实际上是 zip 文件）
	// 这里需要先解压到临时目录读取 manifest
	// 暂时返回一个基本的 manifest 用于获取版本号

	// 为了简化，我们先创建一个临时目录解压
	tempDir, err := os.MkdirTemp("", "sc-modpack-parse-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压整合包
	if err := extractZip(modpackPath, tempDir); err != nil {
		return nil, fmt.Errorf("解压整合包失败: %v", err)
	}

	// 查找清单文件
	manifestFile, err := findManifestFile(tempDir)
	if err != nil {
		return nil, err
	}

	// 读取清单文件
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("读取清单文件失败: %v", err)
	}

	// 输出原始内容用于调试
	originalContent := string(data)
	fmt.Printf("=== 主解析器 - 原始清单文件内容 ===\n%s\n=== 结束 ===\n", originalContent)

	// 解析 JSON（支持 jsonc 格式，去除注释）
	data = removeJSONComments(data)

	// 修复常见的JSON错误（去除多余的逗号）
	data = fixJSONCommonErrors(data)

	// 输出处理后的内容用于调试
	processedContent := string(data)
	fmt.Printf("=== 主解析器 - 处理后的清单文件内容 ===\n%s\n=== 结束 ===\n", processedContent)

	// 解析基本信息（只需要获取 manifestVersion）
	var basicInfo struct {
		ManifestVersion float64 `json:"manifestVersion"`
	}
	if err := json.Unmarshal(data, &basicInfo); err != nil {
		// 输出详细的错误信息
		fmt.Printf("=== 主解析器 - JSON 解析错误详情 ===\n")
		fmt.Printf("错误: %v\n", err)
		fmt.Printf("处理后的内容长度: %d\n", len(data))
		if len(data) > 0 {
			fmt.Printf("前100个字符: %s\n", string(data[:min(100, len(data))]))
		}
		fmt.Printf("=== 错误详情结束 ===\n")
		return nil, fmt.Errorf("解析清单文件失败: %v", err)
	}

	// 创建基本的 Manifest
	manifest := &Manifest{
		ManifestVersion: basicInfo.ManifestVersion,
	}

	return manifest, nil
}

// findManifestFile 查找清单文件
func findManifestFile(dir string) (string, error) {
	// 尝试查找 manifest.jsonc 或 manifest.json
	for _, name := range []string{"manifest.jsonc", "manifest.json"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到清单文件 (manifest.jsonc 或 manifest.json)")
}

// extractZip 解压 zip 文件（简化版本，仅用于读取清单）
func extractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// 只解压清单文件
		if file.Name != "manifest.json" && file.Name != "manifest.jsonc" {
			continue
		}

		path := filepath.Join(destDir, file.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := dst.ReadFrom(src); err != nil {
			return err
		}
	}

	return nil
}

// removeJSONComments 移除 JSON 中的注释（支持 jsonc）
func removeJSONComments(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	var result []string

	for _, line := range lines {
		// 移除行内注释（但要小心字符串中的 "//"）
		inString := false
		escaped := false
		commentStart := -1

		for i, char := range line {
			if escaped {
				escaped = false
				continue
			}

			switch char {
			case '\\':
				escaped = true
			case '"':
				inString = !inString
			case '/':
				if !inString && i+1 < len(line) && line[i+1] == '/' {
					commentStart = i
				}
			}

			if commentStart != -1 {
				break
			}
		}

		if commentStart != -1 {
			line = strings.TrimSpace(line[:commentStart])
		}

		// 保留非空行
		if line != "" {
			result = append(result, line)
		}
	}

	joined := strings.Join(result, "\n")

	// 如果处理后为空，返回原始数据
	if len(joined) == 0 {
		return data
	}

	return []byte(joined)
}

// fixJSONCommonErrors 修复常见的JSON错误
func fixJSONCommonErrors(data []byte) []byte {
	content := string(data)

	// 修复多余的逗号（在 } 或 ] 之前的逗号）
	// 匹配 ",}" 或 ",]" 并替换为 "}" 或 "]"
	content = strings.ReplaceAll(content, ",}", "}")
	content = strings.ReplaceAll(content, ",]", "]")

	return []byte(content)
}
