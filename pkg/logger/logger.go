package logger

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	globalLogger *zap.Logger
	globalSugar  *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level      string // debug, info, warn, error
	Output     string // stdout, file, both
	FilePath   string
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int // days
	Compress   bool
}

// Init 初始化日志系统
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{
			Level:  "info",
			Output: "stdout",
		}
	}

	// 解析日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	// 控制台输出
	if cfg.Output == "stdout" || cfg.Output == "both" {
		stdoutEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		stdoutCore := zapcore.NewCore(stdoutEncoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, stdoutCore)
	}

	// 文件输出
	if cfg.Output == "file" || cfg.Output == "both" {
		if cfg.FilePath != "" {
			// 确保目录存在
			dir := filepath.Dir(cfg.FilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			// 使用 lumberjack 进行日志轮转
			fileWriter := &lumberjack.Logger{
				Filename:   cfg.FilePath,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			}

			fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
			fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(fileWriter), level)
			cores = append(cores, fileCore)
		}
	}

	// 创建 logger
	core := zapcore.NewTee(cores...)
	globalLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	globalSugar = globalLogger.Sugar()

	return nil
}

// GetLogger 获取全局 logger
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// 如果未初始化，使用默认配置
		_ = Init(&Config{Level: "info", Output: "stdout"})
	}
	return globalLogger
}

// GetSugar 获取全局 sugared logger
func GetSugar() *zap.SugaredLogger {
	if globalSugar == nil {
		GetLogger()
	}
	return globalSugar
}

// Debug 日志方法
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 日志方法
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 日志方法
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 日志方法
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 日志方法
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sync 刷新日志缓冲
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Debugf 格式化日志
func Debugf(template string, args ...interface{}) {
	GetSugar().Debugf(template, args...)
}

// Infof 格式化日志
func Infof(template string, args ...interface{}) {
	GetSugar().Infof(template, args...)
}

// Warnf 格式化日志
func Warnf(template string, args ...interface{}) {
	GetSugar().Warnf(template, args...)
}

// Errorf 格式化日志
func Errorf(template string, args ...interface{}) {
	GetSugar().Errorf(template, args...)
}

// With 创建带字段的 logger
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// String 创建字符串字段
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

// Int 创建整数字段
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// Int64 创建 int64 字段
func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

// Err 创建错误字段
func Err(err error) zap.Field {
	return zap.NamedError("error", err)
}

// Any 创建任意类型字段
func Any(key string, val any) zap.Field {
	return zap.Any(key, val)
}

// Bool 创建布尔字段
func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

// Now 获取当前时间（用于日志记录时间差）
func Now() time.Time {
	return time.Now()
}
