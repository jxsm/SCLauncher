package savegame

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"SCLauncher/backend/config"
)

// Manager 存档管理器
type Manager struct {
	config *config.Config
	paths  *config.Paths
}

// SaveGame 存档信息
type SaveGame struct {
	ID             string    `json:"id"`             // 存档 ID（目录名）
	Name           string    `json:"name"`           // 世界名称
	GameVersion    string    `json:"gameVersion"`    // 游戏版本
	GameMode       string    `json:"gameMode"`       // 游戏模式
	LastModified   time.Time `json:"lastModified"`   // 最后修改时间
	IsAutoSave     bool      `json:"isAutoSave"`     // 是否自动保存
	ProjectPath    string    `json:"projectPath"`    // Project文件路径
	WorldPath      string    `json:"worldPath"`      // 存档目录路径
	IsImported     bool      `json:"isImported"`     // 是否来自导入的版本
}

// ProjectXML 存档Project.xml/Project.json结构
type ProjectXML struct {
	// JSON 结构
	Subsystems struct {
		GameInfo struct {
			WorldName                    interface{} `json:"WorldName" xml:"-"` // JSON专用
			OriginalSerializationVersion interface{} `json:"OriginalSerializationVersion" xml:"-"`
			GameMode                     interface{} `json:"GameMode" xml:"-"`
		} `json:"GameInfo"`
		// XML 结构 - 更简单的层级
		Values []struct {
			Name  string `xml:"Name,attr"`
			Value []struct {
				Name  string `xml:"Name,attr"`
				Type  string `xml:"Type,attr"`
				Value string `xml:"Value,attr"`
			} `xml:"Value"`
		} `xml:"Values"`
	} `json:"Subsystems" xml:"Subsystems"`
}

// NewManager 创建存档管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
		paths:  config.NewPaths(cfg),
	}
}

// getImportedVersionOriginalPath 获取导入版本的原始路径
func getImportedVersionOriginalPath(versionPath string) (string, error) {
	importedMetaFile := filepath.Join(versionPath, ".imported")
	if _, err := os.Stat(importedMetaFile); err == nil {
		// 是导入的版本，从元数据文件中读取原始路径
		content, err := os.ReadFile(importedMetaFile)
		if err != nil {
			return "", fmt.Errorf("failed to read import metadata: %w", err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "original_path=") {
				originalPath := strings.TrimPrefix(line, "original_path=")
				return originalPath, nil
			}
		}

		return "", fmt.Errorf("invalid import metadata file")
	}
	return "", nil // 不是导入版本
}

// GetSaveGames 获取指定版本的存档列表
func (m *Manager) GetSaveGames(versionID string) ([]SaveGame, error) {
	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	isImported := err == nil && originalPath != ""

	// 收集所有可能的存档目录
	worldsDirs := []string{
		m.paths.GetGameWorldsDir(versionID),
		m.paths.GetGameDocWorldsDir(versionID),
	}

	// 如果是导入版本，添加原始路径的存档目录
	if isImported && originalPath != "" {
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "Worlds"))
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "doc", "Worlds"))
	}

	var allSaveGames []SaveGame
	processedPaths := make(map[string]bool) // 用于去重

	// 遍历所有可能的存档目录
	for _, worldsDir := range worldsDirs {
		// 检查目录是否存在
		if _, err := os.Stat(worldsDir); os.IsNotExist(err) {
			continue
		}

		// 读取目录
		entries, err := os.ReadDir(worldsDir)
		if err != nil {
			continue // 跳过无法读取的目录
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			worldPath := filepath.Join(worldsDir, entry.Name())

			// 去重：如果这个路径已经处理过，跳过
			if processedPaths[worldPath] {
				continue
			}
			processedPaths[worldPath] = true

			// 尝试读取存档信息
			saveGame, err := m.parseSaveGame(worldPath, isImported)
			if err != nil {
				continue // 跳过无法解析的存档
			}

			allSaveGames = append(allSaveGames, saveGame)
		}
	}

	return allSaveGames, nil
}

// parseSaveGame 解析存档信息
func (m *Manager) parseSaveGame(worldPath string, isImported bool) (SaveGame, error) {
	var saveGame SaveGame

	// 获取目录信息
	info, err := os.Stat(worldPath)
	if err != nil {
		return SaveGame{}, err
	}

	saveGame.WorldPath = worldPath
	saveGame.ID = filepath.Base(worldPath)
	saveGame.LastModified = info.ModTime()
	saveGame.IsImported = isImported

	// 默认使用目录名作为名称
	saveGame.Name = saveGame.ID

	// 尝试读取 Project.json
	projectJSONPath := filepath.Join(worldPath, "Project.json")
	if data, err := os.ReadFile(projectJSONPath); err == nil {
		// 移除 UTF-8 BOM（如果存在）
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

		var projectData ProjectXML
		if err := json.Unmarshal(data, &projectData); err == nil {
			// 从嵌套结构中提取数据，值可能是数组或字符串
			if projectData.Subsystems.GameInfo.WorldName != nil {
				saveGame.Name = extractStringValue(projectData.Subsystems.GameInfo.WorldName)
			}
			if projectData.Subsystems.GameInfo.OriginalSerializationVersion != nil {
				saveGame.GameVersion = extractStringValue(projectData.Subsystems.GameInfo.OriginalSerializationVersion)
			}
			if projectData.Subsystems.GameInfo.GameMode != nil {
				saveGame.GameMode = extractStringValue(projectData.Subsystems.GameInfo.GameMode)
			}
			saveGame.ProjectPath = projectJSONPath
			fmt.Printf("[SaveGame] JSON解析成功: 名称=%s, 版本=%s, 模式=%s\n",
				saveGame.Name, saveGame.GameVersion, saveGame.GameMode)
		} else {
			fmt.Printf("[SaveGame] JSON解析失败: %v, 路径: %s\n", err, projectJSONPath)
			// 打印文件内容的前100个字符用于调试
			if len(data) > 0 {
				preview := string(data)
				if len(preview) > 100 {
					preview = preview[:100]
				}
				fmt.Printf("[SaveGame] 文件内容预览: %s\n", preview)
			}
		}
	} else {
		fmt.Printf("[SaveGame] JSON文件不存在或读取失败: %v\n", err)
	}

	// 尝试读取 Project.xml（不管JSON是否成功，都尝试XML以补充缺失信息）
	projectXMLPath := filepath.Join(worldPath, "Project.xml")
	if data, err := os.ReadFile(projectXMLPath); err == nil {
		var projectData ProjectXML
		if err := xml.Unmarshal(data, &projectData); err == nil {
			// 从 XML 结构中提取数据
			xmlName, xmlVersion, xmlMode := extractXMLData(projectData)
			if xmlName != "" && saveGame.Name == saveGame.ID {
				saveGame.Name = xmlName
			}
			if xmlVersion != "" && saveGame.GameVersion == "" {
				saveGame.GameVersion = xmlVersion
			}
			if xmlMode != "" && saveGame.GameMode == "" {
				saveGame.GameMode = xmlMode
			}
			if saveGame.ProjectPath == "" {
				saveGame.ProjectPath = projectXMLPath
			}
			fmt.Printf("[SaveGame] XML解析成功: 名称=%s, 版本=%s, 模式=%s\n",
				saveGame.Name, saveGame.GameVersion, saveGame.GameMode)
		} else {
			fmt.Printf("[SaveGame] XML解析失败: %v, 路径: %s\n", err, projectXMLPath)
		}
	} else {
		fmt.Printf("[SaveGame] XML文件不存在或读取失败: %v\n", err)
	}

	// 如果最终还是没有名称，使用目录名
	if saveGame.Name == "" || saveGame.Name == saveGame.ID {
		saveGame.Name = saveGame.ID
	}

	// 判断是否自动保存
	saveGame.IsAutoSave = strings.Contains(strings.ToLower(saveGame.ID), "autosave") ||
		strings.Contains(strings.ToLower(saveGame.ID), "auto")

	fmt.Printf("[SaveGame] 最终结果: ID=%s, 名称=%s, 版本=%s, 模式=%s\n",
		saveGame.ID, saveGame.Name, saveGame.GameVersion, saveGame.GameMode)

	return saveGame, nil
}

// extractStringValue 从 interface{} 中提取字符串值
// 处理两种格式：["string", "value"] 或直接是字符串
func extractStringValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		if len(val) >= 2 {
			if str, ok := val[1].(string); ok {
				return str
			}
		}
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// extractXMLData 从 XML 结构中提取存档数据
func extractXMLData(data ProjectXML) (name, version, mode string) {
	// 遍历 Subsystems.Values 查找 GameInfo
	for _, values := range data.Subsystems.Values {
		if values.Name == "GameInfo" {
			// 直接遍历 Value 元素
			for _, value := range values.Value {
				switch value.Name {
				case "WorldName":
					name = value.Value
				case "OriginalSerializationVersion":
					version = value.Value
				case "GameMode":
					mode = value.Value
				}
			}
		}
	}
	return
}

// DeleteSaveGame 删除存档
func (m *Manager) DeleteSaveGame(versionID, saveID string) error {
	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	isImported := err == nil && originalPath != ""

	// 收集所有可能的存档目录
	worldsDirs := []string{
		m.paths.GetGameWorldsDir(versionID),
		m.paths.GetGameDocWorldsDir(versionID),
	}

	// 如果是导入版本，添加原始路径的存档目录
	if isImported && originalPath != "" {
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "Worlds"))
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "doc", "Worlds"))
	}

	// 在所有可能的目录中查找并删除存档
	for _, worldsDir := range worldsDirs {
		worldPath := filepath.Join(worldsDir, saveID)

		// 检查是否存在
		if _, err := os.Stat(worldPath); err == nil {
			// 删除整个存档目录
			if err := os.RemoveAll(worldPath); err != nil {
				return fmt.Errorf("failed to delete save game: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("save game not found: %s", saveID)
}

// GetSaveGamePath 获取存档的路径
func (m *Manager) GetSaveGamePath(versionID, saveID string) (string, error) {
	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	isImported := err == nil && originalPath != ""

	// 收集所有可能的存档目录
	worldsDirs := []string{
		m.paths.GetGameWorldsDir(versionID),
		m.paths.GetGameDocWorldsDir(versionID),
	}

	// 如果是导入版本，添加原始路径的存档目录
	if isImported && originalPath != "" {
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "Worlds"))
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "doc", "Worlds"))
	}

	// 在所有可能的目录中查找存档
	for _, worldsDir := range worldsDirs {
		worldPath := filepath.Join(worldsDir, saveID)

		// 检查是否存在
		if _, err := os.Stat(worldPath); err == nil {
			return worldPath, nil
		}
	}

	return "", fmt.Errorf("save game not found: %s", saveID)
}

// ExportSaveGame 导出存档为.scword文件
func (m *Manager) ExportSaveGame(versionID, saveID, savePath string) error {
	// 获取存档路径
	worldPath, err := m.GetSaveGamePath(versionID, saveID)
	if err != nil {
		return err
	}

	// 确保使用.scword后缀
	if !strings.HasSuffix(strings.ToLower(savePath), ".scword") {
		savePath = savePath + ".scword"
	}

	// 创建临时zip文件
	tempZipPath := savePath + ".tmp.zip"

	// 创建zip文件
	zipFile, err := os.Create(tempZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历存档文件夹并添加到zip
	err = filepath.Walk(worldPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(worldPath, filePath)
		if err != nil {
			return err
		}

		// 如果是目录，创建目录条目
		if info.IsDir() {
			_, err = zipWriter.Create(relPath + "/")
			return err
		}

		// 如果是文件，添加到zip
		if info.Mode().IsRegular() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			writer, err := zipWriter.Create(relPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(writer, file)
			file.Close()

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		os.Remove(tempZipPath)
		return fmt.Errorf("failed to create zip: %w", err)
	}

	zipWriter.Close()
	zipFile.Close()

	// 重命名为.scword文件
	if err := os.Rename(tempZipPath, savePath); err != nil {
		os.Remove(tempZipPath)
		return fmt.Errorf("failed to rename to .scword: %w", err)
	}

	fmt.Printf("[SaveGame] 导出成功: %s\n", savePath)
	return nil
}

// RenameSaveGame 重命名存档
func (m *Manager) RenameSaveGame(versionID, saveID, newName string) error {
	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	isImported := err == nil && originalPath != ""

	// 收集所有可能的存档目录
	worldsDirs := []string{
		m.paths.GetGameWorldsDir(versionID),
		m.paths.GetGameDocWorldsDir(versionID),
	}

	// 如果是导入版本，添加原始路径的存档目录
	if isImported && originalPath != "" {
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "Worlds"))
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "doc", "Worlds"))
	}

	// 在所有可能的目录中查找存档
	for _, worldsDir := range worldsDirs {
		worldPath := filepath.Join(worldsDir, saveID)

		// 检查是否存在
		if _, err := os.Stat(worldPath); err == nil {
			// 找到了存档，尝试重命名
			return m.renameSaveGameInPath(worldPath, newName)
		}
	}

	return fmt.Errorf("save game not found: %s", saveID)
}

// renameSaveGameInPath 在指定路径中重命名存档
func (m *Manager) renameSaveGameInPath(worldPath, newName string) error {
	// 尝试修改 Project.json
	projectJSONPath := filepath.Join(worldPath, "Project.json")
	if data, err := os.ReadFile(projectJSONPath); err == nil {
		// 移除 UTF-8 BOM
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

		var projectData map[string]interface{}
		if err := json.Unmarshal(data, &projectData); err == nil {
			// 修改 JSON 中的 WorldName
			if subsystems, ok := projectData["Subsystems"].(map[string]interface{}); ok {
				if gameInfo, ok := subsystems["GameInfo"].(map[string]interface{}); ok {
					if worldName, ok := gameInfo["WorldName"].([]interface{}); ok && len(worldName) >= 2 {
						// 修改数组格式 ["string", "旧名称"] -> ["string", "新名称"]
						worldName[1] = newName
					}
				}
			}

			// 写回文件
			newData, err := json.MarshalIndent(projectData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}

			if err := os.WriteFile(projectJSONPath, newData, 0644); err != nil {
				return fmt.Errorf("failed to write JSON file: %w", err)
			}

			fmt.Printf("[SaveGame] 重命名成功 (JSON): %s -> %s\n", projectJSONPath, newName)
			return nil
		}
	}

	// 尝试修改 Project.xml
	projectXMLPath := filepath.Join(worldPath, "Project.xml")
	if data, err := os.ReadFile(projectXMLPath); err == nil {
		content := string(data)

		// 使用XML解析找到WorldName的当前值
		var projectData ProjectXML
		if err := xml.Unmarshal(data, &projectData); err == nil {
			// 找到WorldName的当前值
			var oldValue string
			for _, values := range projectData.Subsystems.Values {
				if values.Name == "GameInfo" {
					for _, value := range values.Value {
						if value.Name == "WorldName" {
							oldValue = value.Value
							break
						}
					}
					if oldValue != "" {
						break
					}
				}
			}

			if oldValue != "" {
				// 使用精确的字符串替换，只替换 WorldName 的 Value 属性
				// 构建要替换的字符串：Value="旧值" -> Value="新值"
				oldStr := fmt.Sprintf(`Value="%s"`, oldValue)
				newStr := fmt.Sprintf(`Value="%s"`, newName)

				// 只替换第一个匹配项（WorldName的Value）
				newContent := strings.Replace(content, oldStr, newStr, 1)

				if newContent != content {
					if err := os.WriteFile(projectXMLPath, []byte(newContent), 0644); err != nil {
						return fmt.Errorf("failed to write XML file: %w", err)
					}
					fmt.Printf("[SaveGame] 重命名成功 (XML): '%s' -> '%s'\n", oldValue, newName)
					return nil
				} else {
					fmt.Printf("[SaveGame] 字符串替换失败，未找到: %s\n", oldStr)
				}
			} else {
				fmt.Printf("[SaveGame] 未找到WorldName字段\n")
			}
		} else {
			fmt.Printf("[SaveGame] XML解析失败: %v\n", err)
		}
	}

	return fmt.Errorf("failed to rename: no valid Project file found")
}

// PreviewSaveGame 预览存档信息（从.scword文件中）
func (m *Manager) PreviewSaveGame(sourcePath string) (SaveGame, error) {
	var saveGame SaveGame

	// 检查文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return saveGame, fmt.Errorf("source file not found: %s", sourcePath)
	}

	// 检查文件扩展名
	if !strings.HasSuffix(strings.ToLower(sourcePath), ".scword") {
		return saveGame, fmt.Errorf("invalid file format, expected .scword file")
	}

	// 打开 scword 文件（实际上是 zip 文件）
	zipReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return saveGame, fmt.Errorf("failed to open scword file: %w", err)
	}
	defer zipReader.Close()

	// 从 Project 文件中读取存档信息
	for _, file := range zipReader.File {
		if file.Name == "Project.json" {
			rc, err := file.Open()
			if err != nil {
				continue
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			// 移除 UTF-8 BOM
			data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

			var projectData ProjectXML
			if err := json.Unmarshal(data, &projectData); err == nil {
				if projectData.Subsystems.GameInfo.WorldName != nil {
					saveGame.Name = extractStringValue(projectData.Subsystems.GameInfo.WorldName)
				}
				if projectData.Subsystems.GameInfo.OriginalSerializationVersion != nil {
					saveGame.GameVersion = extractStringValue(projectData.Subsystems.GameInfo.OriginalSerializationVersion)
				}
				if projectData.Subsystems.GameInfo.GameMode != nil {
					saveGame.GameMode = extractStringValue(projectData.Subsystems.GameInfo.GameMode)
				}
				fmt.Printf("[SaveGame] 预览JSON成功: 名称=%s, 版本=%s, 模式=%s\n",
					saveGame.Name, saveGame.GameVersion, saveGame.GameMode)
				break
			}
		} else if file.Name == "Project.xml" {
			rc, err := file.Open()
			if err != nil {
				continue
			}

			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			var projectData ProjectXML
			if err := xml.Unmarshal(data, &projectData); err == nil {
				name, version, mode := extractXMLData(projectData)
				if name != "" {
					saveGame.Name = name
				}
				if version != "" {
					saveGame.GameVersion = version
				}
				if mode != "" {
					saveGame.GameMode = mode
				}
				fmt.Printf("[SaveGame] 预览XML成功: 名称=%s, 版本=%s, 模式=%s\n",
					saveGame.Name, saveGame.GameVersion, saveGame.GameMode)
				break
			}
		}
	}

	// 如果没有读取到名称，使用文件名
	if saveGame.Name == "" {
		saveGame.Name = strings.TrimSuffix(filepath.Base(sourcePath), ".scword")
		saveGame.Name = strings.TrimSuffix(saveGame.Name, ".SCWORD")
	}

	return saveGame, nil
}

// ImportSaveGame 导入存档
func (m *Manager) ImportSaveGame(versionID, sourcePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", sourcePath)
	}

	// 检查文件扩展名
	if !strings.HasSuffix(strings.ToLower(sourcePath), ".scword") {
		return fmt.Errorf("invalid file format, expected .scword file")
	}

	versionPath := m.paths.GetVersionPath(versionID)

	// 检查是否是导入的版本
	originalPath, err := getImportedVersionOriginalPath(versionPath)
	isImported := err == nil && originalPath != ""

	// 收集所有可能的存档目录（按优先级）
	worldsDirs := []string{
		m.paths.GetGameWorldsDir(versionID),
		m.paths.GetGameDocWorldsDir(versionID),
	}

	// 如果是导入版本，添加原始路径的存档目录
	if isImported && originalPath != "" {
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "Worlds"))
		worldsDirs = append(worldsDirs, filepath.Join(originalPath, "doc", "Worlds"))
	}

	// 检查是否存在 Worlds 目录
	var targetWorldsDir string
	for _, worldsDir := range worldsDirs {
		if info, err := os.Stat(worldsDir); err == nil && info.IsDir() {
			targetWorldsDir = worldsDir
			break
		}
	}

	// 如果所有地方都没有 Worlds 文件夹
	if targetWorldsDir == "" {
		return fmt.Errorf("因版本差异，存档文件的存放位置不一样，请启动游戏并创建一个世界后再使用导入功能")
	}

	// 打开 scword 文件（实际上是 zip 文件）
	zipReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open scword file: %w", err)
	}
	defer zipReader.Close()

	// 尝试从 Project 文件中读取存档名称
	var saveName string
	for _, file := range zipReader.File {
		if file.Name == "Project.json" || file.Name == "Project.xml" {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				continue
			}

			// 尝试解析 JSON
			if file.Name == "Project.json" {
				data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
				var projectData ProjectXML
				if err := json.Unmarshal(data, &projectData); err == nil {
					if projectData.Subsystems.GameInfo.WorldName != nil {
						saveName = extractStringValue(projectData.Subsystems.GameInfo.WorldName)
						break
					}
				}
			}

			// 尝试解析 XML
			if file.Name == "Project.xml" {
				var projectData ProjectXML
				if err := xml.Unmarshal(data, &projectData); err == nil {
					name, _, _ := extractXMLData(projectData)
					if name != "" {
						saveName = name
						break
					}
				}
			}
		}
	}

	// 如果没有读取到名称，使用文件名（去掉 .scword 后缀）
	if saveName == "" {
		saveName = strings.TrimSuffix(filepath.Base(sourcePath), ".scword")
		saveName = strings.TrimSuffix(saveName, ".SCWORD")
	}

	// 创建目标文件夹
	targetDir := filepath.Join(targetWorldsDir, saveName)

	// 检查文件夹是否已存在
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("存档已存在: %s", saveName)
	}

	// 创建目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 解压文件
	for _, file := range zipReader.File {
		// 跳过 macOS 产生的元数据文件
		if strings.HasPrefix(file.Name, "__MACOSX") || file.Name == ".DS_Store" {
			continue
		}

		// 构建目标路径
		targetPath := filepath.Join(targetDir, file.Name)

		// 如果是目录条目
		if file.Mode().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// 创建文件的父目录
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// 创建目标文件
		dstFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		// 打开 zip 中的文件
		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		// 复制文件内容
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		srcFile.Close()
		dstFile.Close()
	}

	fmt.Printf("[SaveGame] 导入成功: %s -> %s\n", sourcePath, targetDir)
	return nil
}
