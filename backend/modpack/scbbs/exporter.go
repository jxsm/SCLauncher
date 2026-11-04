// exporter.go 实现 scbbs 整合包导出（ZIP 载体）。
//
// 打包布局对齐手机版 ScbbsPackExporter.createPackZip：
//
//	packinfo.json                 清单（Name/PackageName/Version/Description/Items）
//	worlds/<存档名>.scworld        世界文件夹内容压缩成的内嵌 zip
//	mods/<模组文件名>              启用模组原文件（.scmod/.netmod 同放 mods 根段）
package scbbs

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ExportMod 待内嵌的模组文件
type ExportMod struct {
	SourcePath  string // 模组文件路径
	FileName    string // 包内/落地文件名
	PackageName string // modinfo 包名（可空，空则不写入校验字段）
	Version     string // modinfo 版本（可空）
}

// exportItem 导出用条目（PascalCase 键与格式规范一致；解析端大小写不敏感）
type exportItem struct {
	Category        string `json:"Category"`
	FileName        string `json:"FileName"`
	Url             string `json:"Url"`
	TargetDirectory string `json:"TargetDirectory"`
	PackageName     string `json:"PackageName,omitempty"`
	Version         string `json:"Version,omitempty"`
}

// exportManifest 导出用清单
type exportManifest struct {
	Name        string       `json:"Name"`
	PackageName string       `json:"PackageName"`
	Version     string       `json:"Version"`
	Description string       `json:"Description"`
	Items       []exportItem `json:"Items"`
}

// ExportSaveGamePack 把存档世界（及启用模组）打包为 scbbs ZIP 整合包。
// worldPath 为世界文件夹；description 空则回退 worldName（Description 必填非空）；
// mods 可为空（纯存档包）。destPath 建议以 .zip 结尾（scbbs ZIP 载体惯例）。
func ExportSaveGamePack(worldPath, worldName, description string, mods []ExportMod, destPath string) error {
	baseName := sanitizeFileBaseName(worldName)
	if baseName == "" {
		return fmt.Errorf("存档名无效: %q", worldName)
	}
	if info, err := os.Stat(worldPath); err != nil || !info.IsDir() {
		return fmt.Errorf("世界目录不存在: %s", worldPath)
	}
	if strings.TrimSpace(description) == "" {
		description = worldName
	}

	worldEntry := "worlds/" + baseName + ".scworld"
	items := []exportItem{{
		Category:        "world",
		FileName:        baseName + ".scworld",
		Url:             worldEntry,
		TargetDirectory: "worlds",
	}}

	// 先写 .tmp 再重命名，避免导出中途失败留下半成品
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("创建导出文件失败: %w", err)
	}

	zw := zip.NewWriter(f)

	// 模组文件（同名去重，先出现为准）
	seenModNames := make(map[string]bool)
	for _, m := range mods {
		name := sanitizeFileBaseName(m.FileName)
		if name == "" || seenModNames[name] {
			continue
		}
		seenModNames[name] = true

		entry := "mods/" + name
		if err := addExportFile(zw, entry, m.SourcePath); err != nil {
			f.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("写入模组 %s 失败: %w", m.FileName, err)
		}

		item := exportItem{
			Category:        "mod",
			FileName:        name,
			Url:             entry,
			TargetDirectory: "mods",
		}
		if strings.TrimSpace(m.PackageName) != "" {
			item.PackageName = m.PackageName
		}
		if strings.TrimSpace(m.Version) != "" {
			item.Version = m.Version
		}
		items = append(items, item)
	}

	// 世界内嵌 .scworld（世界文件夹内容的 zip，条目不带包裹目录）
	worldWriter, err := zw.CreateHeader(&zip.FileHeader{
		Name:   worldEntry,
		Method: zip.Deflate,
	})
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("创建世界条目失败: %w", err)
	}
	if err := writeWorldZip(worldWriter, worldPath); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("打包世界失败: %w", err)
	}

	// packinfo.json
	manifest := exportManifest{
		Name:        worldName,
		PackageName: worldName,
		Version:     "1.0.0",
		Description: description,
		Items:       items,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("生成清单失败: %w", err)
	}
	mw, err := zw.Create("packinfo.json")
	if err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("创建清单条目失败: %w", err)
	}
	if _, err := mw.Write(manifestBytes); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("写入清单失败: %w", err)
	}

	if err := zw.Close(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("完成压缩包失败: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("关闭导出文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名导出文件失败: %w", err)
	}
	return nil
}

// writeWorldZip 把 worldPath 文件夹内容压缩进 w（条目为相对路径、正斜杠分隔，不带包裹目录）
func writeWorldZip(w io.Writer, worldPath string) error {
	inner := zip.NewWriter(w)
	err := filepath.Walk(worldPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(worldPath, p)
		if err != nil {
			return err
		}
		fw, err := inner.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		src, err := os.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(fw, src)
		return err
	})
	if err != nil {
		inner.Close()
		return err
	}
	return inner.Close()
}

// addExportFile 把磁盘文件以 Deflate 写入 zip 条目
func addExportFile(zw *zip.Writer, entry, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	fw, err := zw.CreateHeader(&zip.FileHeader{
		Name:   entry,
		Method: zip.Deflate,
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, src)
	return err
}

// sanitizeFileBaseName 净化文件基础名（对齐手机版 _sanitizeFileBaseName）：
// 路径分隔符与 Windows 非法字符、控制字符替换为下划线，去首尾空白与尾点。
func sanitizeFileBaseName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\\", "/"), "/", "_"))
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|`, r) {
			b.WriteRune('_')
		} else if unicode.IsControl(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	result := strings.TrimRight(b.String(), " .")
	if result == "" || result == "." || result == ".." {
		return ""
	}
	return result
}
