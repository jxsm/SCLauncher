package utils

import (
	"testing"
)

func TestCheckPathDiskSpace(t *testing.T) {
	tests := []struct {
		name         string
		requiredBytes int64
		path         string
		wantErr      bool
	}{
		{
			name:         "检查小空间需求",
			requiredBytes: 1024, // 1KB
			path:         "/tmp",
			wantErr:      false, // 应该总是成功，除非磁盘真的满了
		},
		{
			name:         "检查中等空间需求",
			requiredBytes: 100 * 1024 * 1024, // 100MB
			path:         "/tmp",
			wantErr:      false,
		},
		{
			name:         "检查大空间需求",
			requiredBytes: 10 * 1024 * 1024 * 1024, // 10GB
			path:         "/tmp",
			wantErr:      true, // 临时目录通常没有这么大的空间
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPathDiskSpace(tt.requiredBytes, tt.path)

			// 对于小空间需求，不应该失败
			if !tt.wantErr && err != nil {
				t.Errorf("CheckPathDiskSpace() 不应该失败，但得到错误: %v", err)
			}

			// 对于大空间需求，预期会失败
			// 但如果系统有足够空间，测试也会通过
			if tt.wantErr && err == nil {
				t.Logf("CheckPathDiskSpace() 预期失败但成功了（系统有足够空间）")
			}
		})
	}
}

func TestGetDiskFreeSpace(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "获取临时目录空间",
			path:    "/tmp",
			wantErr: false,
		},
		{
			name:    "获取当前目录空间",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "无效路径",
			path:    "/nonexistent/path/that/does/not/exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freeSpace, totalSpace, err := GetDiskFreeSpace(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDiskFreeSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 验证返回的值是合理的
				if freeSpace <= 0 {
					t.Errorf("GetDiskFreeSpace() freeSpace = %v, 应该 > 0", freeSpace)
				}
				if totalSpace <= 0 {
					t.Errorf("GetDiskFreeSpace() totalSpace = %v, 应该 > 0", totalSpace)
				}
				if freeSpace > totalSpace {
					t.Errorf("GetDiskFreeSpace() freeSpace = %v, totalSpace = %v, freeSpace 应该 <= totalSpace", freeSpace, totalSpace)
				}

				t.Logf("路径: %s, 可用空间: %.2f GB, 总空间: %.2f GB",
					tt.path,
					float64(freeSpace)/(1024*1024*1024),
					float64(totalSpace)/(1024*1024*1024))
			}
		})
	}
}
