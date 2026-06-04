package logger

import (
	"os"
	"path/filepath"

	"gateway/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Logger 全局日志实例
	Logger *zap.Logger
	// SugaredLogger 全局糖化日志实例（方便使用）
	SugaredLogger *zap.SugaredLogger
)

// InitLogger 初始化日志系统
func InitLogger(cfg config.LoggerConfig) error {
	// 创建日志目录
	logDir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 配置日志级别
	level := parseLevel(cfg.Level)

	// 配置日志编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	} else {
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}

	// 配置日志输出（文件 + 控制台）
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,    // MB
		MaxBackups: cfg.MaxBackups, // 保留文件数
		MaxAge:     cfg.MaxAge,     // 天数
		Compress:   cfg.Compress,   // 是否压缩
	})

	// 创建核心
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level), // 控制台输出
		zapcore.NewCore(encoder, fileWriter, level),                  // 文件输出
	)

	// 创建 logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	SugaredLogger = Logger.Sugar()

	return nil
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync 同步日志缓冲区到磁盘
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
