package v0_1

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// V01Parser 版本 0.1 的解析器
type V01Parser struct{}

// NewV01Parser 创建版本 0.1 的解析器
func NewV01Parser() *V01Parser {
	return &V01Parser{}
}

// GetVersion 获取解析器版本
func (p *V01Parser) GetVersion() string {
	return "0.1"
}

// V01Manifest 整合包清单（v0.1）
type V01Manifest struct {
	// 基本信息
	ManifestType    string  `json:"manifestType"`    // 固定值：SurvivalcraftModpack
	ManifestVersion float64 `json:"manifestVersion"` // 清单版本号
	Name            string  `json:"name"`            // 整合包名称
	Version         string  `json:"version"`         // 整合包版本
	Author          string  `json:"author"`          // 作者
	Description     string  `json:"description"`     // 描述
	Icon            string  `json:"icon"`            // 图标文件路径
	Created         string  `json:"created"`         // 创建时间
	Changelog       string  `json:"changelog"`       // 更新日志

	// 游戏核心配置
	Survivalcraft *V01SurvivalcraftConfig `json:"survivalcraft"` // 生存战争配置

	// 模组列表
	Mods    []V01ModInfo `json:"mods"`    // 模组列表
	ModPath string        `json:"modPath"` // 模组存放路径（默认为/Mods，联机版为/NetMods）

	// 自定义覆盖文件
	Overrides string `json:"overrides"` // 覆盖文件目录名

	// 校验
	Checksum string `json:"checksum"` // 校验方式（sha256）

	// 解析后的额外信息
	FilePath string `json:"-"` // 整合包文件路径
	FileHash string `json:"-"` // 文件哈希值
}

// V01SurvivalcraftConfig 生存战争配置
type V01SurvivalcraftConfig struct {
	Version    V01VersionConfig      `json:"version"`    // 版本配置
	VersionList V01VersionListConfig `json:"versionList"` // 版本列表配置
}

// V01VersionConfig 版本配置
type V01VersionConfig struct {
	Manual  bool                 `json:"manual"`  // 是否手动选择版本（默认false）
	Android *V01PlatformVersion `json:"android"` // Android 平台配置
	Windows *V01PlatformVersion `json:"windows"` // Windows 平台配置
}

// V01PlatformVersion 平台版本信息
type V01PlatformVersion struct {
	Version        string `json:"version"`         // 版本号（如：2.4:api-1.8.2.3）
	APKPackageName string `json:"apkPackageName"`  // APK 包名（Android）
	Path           string `json:"path"`            // 下载路径
}

// V01VersionListConfig 版本列表配置
type V01VersionListConfig struct {
	Android string `json:"android"` // Android 版本列表 URL
	Windows string `json:"windows"` // Windows 版本列表 URL
}

// V01ModInfo 模组信息
type V01ModInfo struct {
	ProjectID int    `json:"projectId"` // 模组项目 ID
	Version   string `json:"version"`   // 模组版本
	Name      string `json:"name"`      // 模组名称
	Required  bool   `json:"required"`  // 是否必须
	Path      string `json:"path"`      // 下载路径
	ModPath   string `json:"modPath"`   // 模组安装路径（可选，如果为空则使用全局ModPath）
}

// Parse 解析整合包
func (p *V01Parser) Parse(modpackPath string) (any, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "sc-modpack-v0.1-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压整合包
	if err := p.extractZip(modpackPath, tempDir); err != nil {
		return nil, fmt.Errorf("解压整合包失败: %v", err)
	}

	// 查找清单文件
	manifestFile, err := p.findManifestFile(tempDir)
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
	fmt.Printf("=== 原始清单文件内容 ===\n%s\n=== 结束 ===\n", originalContent)

	// 移除 JSON 注释（支持 jsonc）
	data = p.removeJSONComments(data)

	// 修复常见的JSON错误（去除多余的逗号）
	data = p.fixJSONCommonErrors(data)

	// 输出处理后的内容用于调试
	processedContent := string(data)
	fmt.Printf("=== 处理后的清单文件内容 ===\n%s\n=== 结束 ===\n", processedContent)

	// 解析清单
	var manifest V01Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		// 输出详细的错误信息
		fmt.Printf("=== JSON 解析错误详情 ===\n")
		fmt.Printf("错误: %v\n", err)
		fmt.Printf("处理后的内容长度: %d\n", len(data))
		fmt.Printf("前100个字符: %s\n", string(data[:min(100, len(data))]))
		fmt.Printf("=== 错误详情结束 ===\n")
		return nil, fmt.Errorf("解析清单文件失败: %v", err)
	}

	// 验证清单
	if err := p.Validate(&manifest); err != nil {
		return nil, err
	}

	// 检查 Windows 平台支持
	if err := p.checkPlatformSupport(&manifest); err != nil {
		return nil, err
	}

	// 设置默认的模组安装路径
	if manifest.ModPath == "" {
		manifest.ModPath = "/Mods"
	}

	// 计算文件哈希
	fileHash, err := p.calculateFileHash(modpackPath)
	if err != nil {
		return nil, fmt.Errorf("计算文件哈希失败: %v", err)
	}

	// 校验文件完整性
	if err := p.verifyChecksum(&manifest, fileHash); err != nil {
		return nil, err
	}

	// 设置额外信息
	manifest.FilePath = modpackPath
	manifest.FileHash = fileHash

	return &manifest, nil
}

// Validate 验证清单
func (p *V01Parser) Validate(manifest any) error {
	m, ok := manifest.(*V01Manifest)
	if !ok {
		return fmt.Errorf("无效的清单类型")
	}

	// 验证清单类型
	if m.ManifestType != "SurvivalcraftModpack" {
		return fmt.Errorf("不支持的清单类型: %s", m.ManifestType)
	}

	// 验证清单版本
	if m.ManifestVersion != 0.1 {
		return fmt.Errorf("不支持的清单版本: %.1f", m.ManifestVersion)
	}

	// 验证必填字段
	if m.Name == "" {
		return fmt.Errorf("整合包名称不能为空")
	}

	if m.Version == "" {
		return fmt.Errorf("整合包版本不能为空")
	}

	if m.Author == "" {
		return fmt.Errorf("作者不能为空")
	}

	return nil
}

// checkPlatformSupport 检查平台支持
func (p *V01Parser) checkPlatformSupport(manifest *V01Manifest) error {
	// survivalcraft 配置是可选的，如果缺失则提示用户
	if manifest.Survivalcraft == nil {
		return fmt.Errorf("该整合包未配置游戏版本信息，暂不支持安装")
	}

	// 如果是手动选择版本模式，则跳过平台版本配置检查
	if manifest.Survivalcraft.Version.Manual {
		return nil
	}

	// 检查 Version 配置是否存在
	if manifest.Survivalcraft.Version.Windows == nil && manifest.Survivalcraft.Version.Android == nil {
		return fmt.Errorf("该整合包未配置任何平台版本信息")
	}

	// 检查是否有 Windows 版本配置
	if manifest.Survivalcraft.Version.Windows == nil {
		return fmt.Errorf("该整合包暂不支持 Windows 平台（仅支持 Android）")
	}

	// 检查 Windows 版本配置是否有效
	windowsConfig := manifest.Survivalcraft.Version.Windows
	if windowsConfig.Version == "" {
		return fmt.Errorf("该整合包的 Windows 版本配置无效")
	}

	// carry格式检查：验证版本号格式
	if strings.HasPrefix(windowsConfig.Version, "2.4:carry/") {
		// carry格式，版本号应该是 2.4:carry/<游戏文件路径>
		// 例如: 2.4:carry/game.zip 或 2.4:carry/sc-game.zip
		carryPath := strings.TrimPrefix(windowsConfig.Version, "2.4:carry/")
		if carryPath == "" {
			return fmt.Errorf("carry格式版本号无效: 缺少游戏文件路径")
		}
		// 游戏文件路径不能包含路径遍历字符
		if strings.Contains(carryPath, "..") {
			return fmt.Errorf("carry格式版本号无效: 游戏文件路径包含非法字符")
		}
	}

	return nil
}

// extractZip 解压 zip 文件
func (p *V01Parser) extractZip(zipPath, destDir string) error {
	// 打开 zip 文件
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败: %v", err)
	}
	defer reader.Close()

	// 解压每个文件
	for _, file := range reader.File {
		// 构建目标路径
		path := filepath.Join(destDir, file.Name)

		// 检查是否是 Zip Slip 漏洞
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// 创建目录
			os.MkdirAll(path, 0755)
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("创建目录失败: %v", err)
		}

		// 解压文件
		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}
		defer fileReader.Close()

		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, fileReader); err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}
	}

	return nil
}

// findManifestFile 查找清单文件
func (p *V01Parser) findManifestFile(dir string) (string, error) {
	// 尝试查找 manifest.jsonc 或 manifest.json
	for _, name := range []string{"manifest.jsonc", "manifest.json"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到清单文件 (manifest.jsonc 或 manifest.json)")
}

// removeJSONComments 移除 JSON 中的注释（支持 jsonc）
func (p *V01Parser) removeJSONComments(data []byte) []byte {
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

		// 保留非空行和包含逗号或括号的行（保持JSON结构）
		if line != "" || len(result) > 0 {
			result = append(result, line)
		}
	}

	joined := strings.Join(result, "\n")

	// 验证结果是否为有效的JSON
	if len(joined) == 0 {
		return data
	}

	return []byte(joined)
}

// fixJSONCommonErrors 修复常见的JSON错误
func (p *V01Parser) fixJSONCommonErrors(data []byte) []byte {
	content := string(data)

	// 修复多余的逗号（在 } 或 ] 之前的逗号）
	// 匹配 ",}" 或 ",]" 并替换为 "}" 或 "]"
	content = strings.ReplaceAll(content, ",}", "}")
	content = strings.ReplaceAll(content, ",]", "]")

	// 修复空字符串值（将 "path": "" 等改为 null）
	// 但要小心不要破坏正常的空字符串
	content = p.fixEmptyStringValues(content)

	return []byte(content)
}

// fixEmptyStringValues 修复空字符串值
func (p *V01Parser) fixEmptyStringValues(content string) string {
	// 暂时不处理空字符串，因为它们在某些情况下是有效的
	// 如果需要，可以在这里添加逻辑来处理特定的空字符串情况
	return content
}

// calculateFileHash 计算文件哈希
func (p *V01Parser) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyChecksum 验证文件校验和
func (p *V01Parser) verifyChecksum(manifest *V01Manifest, fileHash string) error {
	// 如果没有配置 checksum，跳过校验
	if manifest.Checksum == "" {
		return nil
	}

	// 检查校验方式
	if manifest.Checksum != "sha256" && manifest.Checksum != "SHA256" {
		return fmt.Errorf("不支持的校验方式: %s", manifest.Checksum)
	}

	// TODO: 这里应该有一个预存的哈希值来比较
	// 目前只是计算哈希并存储，实际应用中需要从清单中获取期望的哈希值进行比较
	// 可以在未来添加校验逻辑：
	// expectedHash := manifest.ExpectedChecksum // 从清单中读取期望的哈希值
	// if fileHash != expectedHash {
	//     return fmt.Errorf("整合包已损坏，校验和不匹配")
	// }
	_ = fileHash // 暂时避免未使用参数警告

	return nil
}
