package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"ai_push_message/config"
)

// Logger 全局日志记录器
var Logger *slog.Logger

// InitSlog 初始化slog日志系统
func InitSlog(cfg *config.Config) error {
	level := cfg.Log.Level
	format := cfg.Log.Format
	output := cfg.Log.Output
	filePath := cfg.Log.FilePath

	// 创建日志目录
	if filePath != "" {
		logDir := filepath.Dir(filePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}
	}

	// 设置日志级别
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// 设置输出目标
	var writer io.Writer
	switch strings.ToLower(output) {
	case "file":
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writer = file
	case "both":
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writer = io.MultiWriter(os.Stdout, file)
	default:
		writer = os.Stdout
	}

	// 设置日志格式
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	// 设置默认logger和全局Logger变量
	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	return nil
}

// Init 使用配置文件初始化日志系统
func Init(cfg *config.Config) error {
	return InitSlog(cfg)
}

// Debug 记录调试级别的日志
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Info 记录信息级别的日志
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn 记录警告级别的日志
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error 记录错误级别的日志
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}
