// Package logger 提供启动器自身的文件日志：
// 写入 <数据目录>/Logs/launcher-YYYY-MM-DD.log，按天分隔，只保留最近 7 天。
// 全局单例，后端任意包可直接调用 logger.Info / logger.Warn / logger.Error，
// 写入失败只降级（静默丢弃），绝不影响主流程。
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// logFilePrefix 日志文件名前缀，配合日期组成 launcher-2026-11-05.log
	logFilePrefix = "launcher-"
	// logFileExt 日志文件扩展名
	logFileExt = ".log"
	// retainDays 日志保留天数（含当天）
	retainDays = 7
	// timeLayout 日志文件名中的日期格式
	dateLayout = "2006-01-02"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// levelNames 级别标签（写入日志行）
var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger 按天滚动的文件日志器
type Logger struct {
	mu      sync.Mutex
	dir     string // 日志目录（绝对路径）
	day     string // 当前日志文件对应的日期（dateLayout 格式）
	file    io.WriteCloser
	enabled bool             // Init 成功后为 true
	nowFunc func() time.Time // 可注入时钟，测试跨天滚动用
}

// std 全局日志器。Init 之前调用各日志方法只会落到 stderr。
var std = &Logger{nowFunc: time.Now}

// Init 初始化全局日志器：创建目录、打开当天日志文件、清理过期日志。
// 失败时返回错误（此时日志只输出到 stderr，主流程可继续运行）。
func Init(dir string) error {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.initLocked(dir)
}

func (l *Logger) initLocked(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	l.dir = dir
	l.day = l.now().Format(dateLayout)
	file, err := os.OpenFile(l.filePath(l.day), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	l.file = file
	l.enabled = true
	l.cleanLocked()
	return nil
}

// Close 关闭当前日志文件（应用退出时调用）
func Close() {
	std.mu.Lock()
	defer std.mu.Unlock()
	if std.file != nil {
		_ = std.file.Close()
		std.file = nil
	}
	std.enabled = false
}

// Dir 返回日志目录（未初始化时返回空字符串）
func Dir() string {
	std.mu.Lock()
	defer std.mu.Unlock()
	return std.dir
}

func (l *Logger) now() time.Time {
	if l.nowFunc != nil {
		return l.nowFunc()
	}
	return time.Now()
}

func (l *Logger) filePath(day string) string {
	return filepath.Join(l.dir, logFilePrefix+day+logFileExt)
}

// write 写入一条日志。参数经 fmt.Sprint 拼接（不解析 %），消息里的
// URL 编码、百分号等不会被二次转义。
func (l *Logger) write(level Level, args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	line := fmt.Sprintf("%s [%s] %s\r\n", now.Format("15:04:05.000"), levelNames[level], msg)

	// 同时输出到 stderr，方便 wails dev / 命令行启动时直接看到
	fmt.Fprint(os.Stderr, line)

	if !l.enabled || l.file == nil {
		return
	}

	// 跨天滚动：切换到新文件并清理过期日志
	if day := now.Format(dateLayout); day != l.day {
		_ = l.file.Close()
		file, err := os.OpenFile(l.filePath(day), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// 打开失败时停用文件日志，后续只走 stderr
			l.file = nil
			l.enabled = false
			fmt.Fprintf(os.Stderr, "%s [%s] 日志滚动失败，文件日志停用: %v\r\n",
				now.Format("15:04:05.000"), levelNames[LevelError], err)
			return
		}
		l.file = file
		l.day = day
		l.cleanLocked()
	}

	if _, err := io.WriteString(l.file, line); err != nil {
		// 写失败（磁盘满等）：静默，下次写仍尝试
	}
}

// cleanLocked 删除保留期之外的日志文件（调用方需持有锁）。
// 按文件名中的日期判断，无法解析的名字不动。
func (l *Logger) cleanLocked() {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}

	type logFile struct {
		day  string
		path string
	}
	var files []logFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, logFilePrefix) || !strings.HasSuffix(name, logFileExt) {
			continue
		}
		day := strings.TrimSuffix(strings.TrimPrefix(name, logFilePrefix), logFileExt)
		if _, err := time.Parse(dateLayout, day); err != nil {
			continue
		}
		files = append(files, logFile{day: day, path: filepath.Join(l.dir, name)})
	}
	if len(files) <= retainDays {
		return
	}

	// 按日期从旧到新排序，保留最新 retainDays 个
	sort.Slice(files, func(i, j int) bool { return files[i].day < files[j].day })
	for _, f := range files[:len(files)-retainDays] {
		if err := os.Remove(f.path); err != nil {
			fmt.Fprintf(os.Stderr, "清理过期日志失败 %s: %v\r\n", f.path, err)
		}
	}
}

// ---------- 全局便捷方法（fmt.Sprint 语义，消息中的 % 不会被转义） ----------

// Debug 记录调试日志
func Debug(args ...interface{}) { std.write(LevelDebug, args...) }

// Info 记录一般信息
func Info(args ...interface{}) { std.write(LevelInfo, args...) }

// Warn 记录警告
func Warn(args ...interface{}) { std.write(LevelWarn, args...) }

// Error 记录错误
func Error(args ...interface{}) { std.write(LevelError, args...) }

// Debugf 记录调试日志（Printf 风格）
func Debugf(format string, args ...interface{}) { std.write(LevelDebug, fmt.Sprintf(format, args...)) }

// Infof 记录一般信息（Printf 风格）
func Infof(format string, args ...interface{}) { std.write(LevelInfo, fmt.Sprintf(format, args...)) }

// Warnf 记录警告（Printf 风格）
func Warnf(format string, args ...interface{}) { std.write(LevelWarn, fmt.Sprintf(format, args...)) }

// Errorf 记录错误（Printf 风格）
func Errorf(format string, args ...interface{}) { std.write(LevelError, fmt.Sprintf(format, args...)) }
