package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestInitAndWrite 验证初始化创建当天日志文件并按格式写入
func TestInitAndWrite(t *testing.T) {
	dir := t.TempDir()
	defer Close()

	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	Info("hello", " ", "world")
	Error("boom")

	day := time.Now().Format(dateLayout)
	data, err := os.ReadFile(filepath.Join(dir, logFilePrefix+day+logFileExt))
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[INFO] hello world") {
		t.Errorf("info line missing, got: %q", content)
	}
	if !strings.Contains(content, "[ERROR] boom") {
		t.Errorf("error line missing, got: %q", content)
	}
	if Dir() != dir {
		t.Errorf("Dir() = %q, want %q", Dir(), dir)
	}
}

// TestPercentNotMangled 验证消息中的 % 不会被格式化转义
func TestPercentNotMangled(t *testing.T) {
	dir := t.TempDir()
	defer Close()

	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	Info("url https://example.com/a%20b?x=1&p=2 100%")

	day := time.Now().Format(dateLayout)
	data, err := os.ReadFile(filepath.Join(dir, logFilePrefix+day+logFileExt))
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(data), "url https://example.com/a%20b?x=1&p=2 100%") {
		t.Errorf("percent message mangled: %q", string(data))
	}
}

// TestCleanRetainSevenDays 验证只保留最近 7 天的日志
func TestCleanRetainSevenDays(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 11, 5, 10, 0, 0, 0, time.Local)

	// 预置 9 个历史日志文件（01-01 ~ 01-09），加上当天文件共 10 个
	for i := 1; i <= 9; i++ {
		day := time.Date(2026, 1, i, 0, 0, 0, 0, time.Local).Format(dateLayout)
		if err := os.WriteFile(filepath.Join(dir, logFilePrefix+day+logFileExt), nil, 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	l := &Logger{nowFunc: func() time.Time { return base }}
	if err := l.initLocked(dir); err != nil {
		t.Fatalf("initLocked failed: %v", err)
	}
	defer func() { _ = l.file.Close() }() // Windows 下打开的文件会阻塞 TempDir 清理

	// 保留 7 个最新的：11-05（当天）+ 01-09 ~ 01-04；01-01 ~ 01-03 应被删除
	for i := 1; i <= 3; i++ {
		day := time.Date(2026, 1, i, 0, 0, 0, 0, time.Local).Format(dateLayout)
		if _, err := os.Stat(filepath.Join(dir, logFilePrefix+day+logFileExt)); !os.IsNotExist(err) {
			t.Errorf("old log %s should be removed", day)
		}
	}
	for i := 4; i <= 9; i++ {
		day := time.Date(2026, 1, i, 0, 0, 0, 0, time.Local).Format(dateLayout)
		if _, err := os.Stat(filepath.Join(dir, logFilePrefix+day+logFileExt)); err != nil {
			t.Errorf("recent log %s should be kept: %v", day, err)
		}
	}
}

// TestRotateAcrossDays 验证跨天写入时滚动到新文件并触发清理
func TestRotateAcrossDays(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 5, 23, 59, 50, 0, time.Local)

	l := &Logger{nowFunc: func() time.Time { return now }}
	if err := l.initLocked(dir); err != nil {
		t.Fatalf("initLocked failed: %v", err)
	}
	defer func() { _ = l.file.Close() }() // Windows 下打开的文件会阻塞 TempDir 清理
	l.write(LevelInfo, "day-1 message")

	// 跨天：写入应切到新文件
	now = now.Add(2 * time.Minute)
	l.write(LevelInfo, "day-2 message")

	day1 := "2026-11-05"
	day2 := "2026-11-06"
	data1, err := os.ReadFile(filepath.Join(dir, logFilePrefix+day1+logFileExt))
	if err != nil {
		t.Fatalf("day1 log missing: %v", err)
	}
	if !strings.Contains(string(data1), "day-1 message") {
		t.Errorf("day1 content wrong: %q", string(data1))
	}
	data2, err := os.ReadFile(filepath.Join(dir, logFilePrefix+day2+logFileExt))
	if err != nil {
		t.Fatalf("day2 log missing: %v", err)
	}
	if !strings.Contains(string(data2), "day-2 message") {
		t.Errorf("day2 content wrong: %q", string(data2))
	}
	if strings.Contains(string(data2), "day-1 message") {
		t.Errorf("day1 message leaked into day2 file")
	}
}

// TestWriteBeforeInit 验证未初始化时写入不 panic
func TestWriteBeforeInit(t *testing.T) {
	l := &Logger{}
	l.write(LevelInfo, "no file yet") // 不应 panic
}
