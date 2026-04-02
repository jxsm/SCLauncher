package version

import (
	"fmt"
	"os"
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
