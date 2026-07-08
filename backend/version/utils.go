package version

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckDownloadFile 检查下载的文件是否存在
func CheckDownloadFile(filePath string) error {
	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("downloaded file not found: %s", filePath)
		}
		return fmt.Errorf("failed to check file: %w", err)
	}

	// 检查文件大小
	if info.Size() == 0 {
		return fmt.Errorf("downloaded file is empty: %s", filePath)
	}

	return nil
}

// ReadGameVersion 从游戏目录下读取版本号
// 优先读 Survivalcraft.dll（2.4+），没有则读 Survivalcraft.exe（2.3-）
func ReadGameVersion(gameDir string) (string, error) {
	dllPath := filepath.Join(gameDir, "Survivalcraft.dll")
	exePath := filepath.Join(gameDir, "Survivalcraft.exe")

	// 优先尝试 dll
	if _, err := os.Stat(dllPath); err == nil {
		ver, err := readPEVersion(dllPath)
		if err == nil {
			return ver, nil
		}
		// dll 读取失败，继续尝试 exe
	}

	// 回退到 exe
	if _, err := os.Stat(exePath); err == nil {
		return readPEVersion(exePath)
	}

	// 递归查找 dll
	var found string
	filepath.Walk(gameDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.EqualFold(info.Name(), "Survivalcraft.dll") || strings.EqualFold(info.Name(), "Survivalcraft.exe") {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	if found != "" {
		return readPEVersion(found)
	}

	return "", fmt.Errorf("目录中未找到 Survivalcraft.dll 或 Survivalcraft.exe: %s", gameDir)
}

// VS_FIXEDFILEINFO 签名
var vsSignature = []byte{0xBD, 0x04, 0xEF, 0xFE}

// readPEVersion 从 PE 文件中读取 VS_FIXEDFILEINFO 的 FileVersion
func readPEVersion(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 搜索 VS_FIXEDFILEINFO 签名 (0xBD04EFFE)
	sigPos := -1
	for i := 0; i <= len(data)-4; i++ {
		if data[i] == 0xBD && data[i+1] == 0x04 && data[i+2] == 0xEF && data[i+3] == 0xFE {
			sigPos = i
			break
		}
	}
	if sigPos < 0 {
		return "", fmt.Errorf("未找到 VS_FIXEDFILEINFO 签名")
	}

	// FileVersion 在签名偏移 +8 (DWORD) 和 +12 (DWORD)
	if sigPos+16 > len(data) {
		return "", fmt.Errorf("文件数据不足")
	}

	fvMs := binary.LittleEndian.Uint32(data[sigPos+8:])
	fvLs := binary.LittleEndian.Uint32(data[sigPos+12:])

	major := (fvMs >> 16) & 0xFFFF
	minor := fvMs & 0xFFFF
	patch := (fvLs >> 16) & 0xFFFF
	build := fvLs & 0xFFFF

	version := fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
	if version == "0.0.0.0" {
		return "", fmt.Errorf("版本号为 0.0.0.0")
	}

	return version, nil
}
