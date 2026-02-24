// 日志工具库
package libs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 日志级别类型
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 普通信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
	// FATAL 致命错误级别
	FATAL
)

// levelNames 日志级别名称映射
var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// levelColors 日志级别颜色（控制台输出）
var levelColors = map[LogLevel]string{
	DEBUG: "\x1b[36m", // 青色
	INFO:  "\x1b[32m", // 绿色
	WARN:  "\x1b[33m", // 黄色
	ERROR: "\x1b[31m", // 红色
	FATAL: "\x1b[35m", // 紫色
}

const colorReset = "\x1b[0m"

// LogConfig 日志配置
type LogConfig struct {
	Enabled      bool   `mapstructure:"enabled"`      // 是否启用日志
	Level        string `mapstructure:"level"`        // 日志级别：debug, info, warn, error, fatal
	LogDir       string `mapstructure:"logDir"`       // 日志目录
	Console      bool   `mapstructure:"console"`      // 是否输出到控制台
	File         bool   `mapstructure:"file"`         // 是否输出到文件
	ColorConsole bool   `mapstructure:"colorConsole"` // 控制台是否彩色输出
	MaxAgeDays   int    `mapstructure:"maxAgeDays"`   // 日志保留天数（0表示不清理）
}

// LogLib 日志库
type LogLib struct {
	config    *LogConfig
	level     LogLevel
	file      *os.File
	fileDate  string // 当前日志文件日期 (YYYY-MM)
	mu        sync.Mutex
	logDir    string
}

var (
	globalLogLib *LogLib
	logLibOnce   sync.Once
)

// NewLogLib 创建日志库实例
func NewLogLib(config *LogConfig) (*LogLib, error) {
	// 设置默认值
	if config == nil {
		config = &LogConfig{
			Enabled:      true,
			Level:        "info",
			LogDir:       "runtime/server/logs",
			Console:      true,
			File:         true,
			ColorConsole: true,
			MaxAgeDays:   30,
		}
	}

	if config.Level == "" {
		config.Level = "info"
	}
	if config.LogDir == "" {
		config.LogDir = "runtime/server/logs"
	}

	// 解析日志级别
	level := parseLevel(config.Level)

	ll := &LogLib{
		config: config,
		level:  level,
		logDir: config.LogDir,
	}

	// 如果启用文件日志，初始化日志文件
	if config.Enabled && config.File {
		if err := ll.rotateLogFile(); err != nil {
			return nil, fmt.Errorf("初始化日志文件失败: %v", err)
		}
	}

	return ll, nil
}

// InitLog 初始化全局日志库
func InitLog(config *LogConfig) error {
	var err error
	logLibOnce.Do(func() {
		globalLogLib, err = NewLogLib(config)
	})
	return err
}

// GetLogLib 获取全局日志库实例
func GetLogLib() *LogLib {
	if globalLogLib == nil {
		// 使用默认配置初始化
		InitLog(nil)
	}
	return globalLogLib
}

// parseLevel 解析日志级别字符串
func parseLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug", "DEBUG":
		return DEBUG
	case "info", "INFO":
		return INFO
	case "warn", "WARN":
		return WARN
	case "error", "ERROR":
		return ERROR
	case "fatal", "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// rotateLogFile 轮转日志文件
func (ll *LogLib) rotateLogFile() error {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	now := time.Now()
	currentDate := now.Format("2006-01") // YYYY-MM 格式

	// 如果日期未变化，不需要轮转
	if ll.fileDate == currentDate && ll.file != nil {
		return nil
	}

	// 关闭旧文件
	if ll.file != nil {
		ll.file.Close()
	}

	// 创建日志目录（按月）
	logDir := filepath.Join(ll.logDir, currentDate)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件（按天）
	logFile := filepath.Join(logDir, now.Format("02")+".log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	ll.file = file
	ll.fileDate = currentDate

	return nil
}

// formatLog 格式化日志消息
// 格式：[2025-06-01 12:00:00] [INFO] 消息内容
func (ll *LogLib) formatLog(level LogLevel, msg string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] [%s] %s", timestamp, levelNames[level], msg)
}

// writeLog 写入日志
func (ll *LogLib) writeLog(level LogLevel, format string, args ...interface{}) {
	// 检查是否启用日志
	if !ll.config.Enabled {
		return
	}

	// 检查日志级别
	if level < ll.level {
		return
	}

	msg := fmt.Sprintf(format, args...)
	logLine := ll.formatLog(level, msg)

	// 检查是否需要轮转日志文件
	if ll.config.File {
		ll.rotateLogFile()
	}

	// 写入控制台
	if ll.config.Console {
		if ll.config.ColorConsole {
			// 彩色输出
			fmt.Printf("%s%s%s\n", levelColors[level], logLine, colorReset)
		} else {
			fmt.Println(logLine)
		}
	}

	// 写入文件
	if ll.config.File && ll.file != nil {
		ll.mu.Lock()
		ll.file.WriteString(logLine + "\n")
		ll.mu.Unlock()
	}

	// FATAL级别，退出程序
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 输出调试级别日志
func (ll *LogLib) Debug(format string, args ...interface{}) {
	ll.writeLog(DEBUG, format, args...)
}

// Info 输出普通信息级别日志
func (ll *LogLib) Info(format string, args ...interface{}) {
	ll.writeLog(INFO, format, args...)
}

// Warn 输出警告级别日志
func (ll *LogLib) Warn(format string, args ...interface{}) {
	ll.writeLog(WARN, format, args...)
}

// Error 输出错误级别日志
func (ll *LogLib) Error(format string, args ...interface{}) {
	ll.writeLog(ERROR, format, args...)
}

// Fatal 输出致命错误级别日志并退出程序
func (ll *LogLib) Fatal(format string, args ...interface{}) {
	ll.writeLog(FATAL, format, args...)
}

// Exception 输出异常日志（带堆栈信息）
// 区别于Error：exception用于记录异常，error用于记录错误
func (ll *LogLib) Exception(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	ll.writeLog(ERROR, "[EXCEPTION] %s", msg)
}

// SetLevel 设置日志级别
func (ll *LogLib) SetLevel(level string) {
	ll.level = parseLevel(level)
}

// SetOutput 设置输出目标（用于测试）
func (ll *LogLib) SetOutput(w io.Writer) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	if ll.file != nil {
		ll.file.Close()
	}
	ll.file = nil
}

// Close 关闭日志库
func (ll *LogLib) Close() error {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	if ll.file != nil {
		return ll.file.Close()
	}
	return nil
}

// ============ 全局日志函数 ============

// Debug 输出调试级别日志（全局）
func Debug(format string, args ...interface{}) {
	GetLogLib().Debug(format, args...)
}

// Info 输出普通信息级别日志（全局）
func Info(format string, args ...interface{}) {
	GetLogLib().Info(format, args...)
}

// Warn 输出警告级别日志（全局）
func Warn(format string, args ...interface{}) {
	GetLogLib().Warn(format, args...)
}

// Error 输出错误级别日志（全局）
func Error(format string, args ...interface{}) {
	GetLogLib().Error(format, args...)
}

// Fatal 输出致命错误级别日志并退出程序（全局）
func Fatal(format string, args ...interface{}) {
	GetLogLib().Fatal(format, args...)
}

// Exception 输出异常日志（全局）
func Exception(format string, args ...interface{}) {
	GetLogLib().Exception(format, args...)
}
