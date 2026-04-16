package log

import (
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	defaultLoggerOnce sync.Once
	defaultLogger     *zap.Logger
)

func Default() *zap.Logger {
	defaultLoggerOnce.Do(func() {
		if defaultLogger != nil {
			return
		}

		ec := zap.NewProductionEncoderConfig()
		ec.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)
		ec.EncodeLevel = zapcore.CapitalLevelEncoder

		var ws zapcore.WriteSyncer
		var wss []zapcore.WriteSyncer

		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.TempDir()
		}

		logDir := home + "/.mattermost-mcp"
		if err := os.MkdirAll(logDir, 0o700); err != nil {
			logDir = os.TempDir()
		}

		wss = append(wss, zapcore.AddSync(&lumberjack.Logger{
			Filename:   logDir + "/mattermost-mcp.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
		}))

		ws = zapcore.NewMultiWriteSyncer(wss...)

		enc := zapcore.NewConsoleEncoder(ec)
		core := zapcore.NewCore(enc, ws, zapcore.InfoLevel)
		options := []zap.Option{
			zap.AddStacktrace(zapcore.DPanicLevel),
			zap.AddCaller(),
			zap.AddCallerSkip(1),
		}
		defaultLogger = zap.New(core, options...)
	})

	return defaultLogger
}

func SetDefault(logger *zap.Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

type Logger struct {
	*zap.Logger
	ctx context.Context
}

func New() *Logger {
	return WithContext(context.Background())
}

func WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: Default(),
		ctx:    ctx,
	}
}

func Debug(msg string, fields ...zap.Field) {
	Default().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Default().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Default().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Default().Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	Default().Panic(msg, fields...)
}

func Debugf(format string, args ...any) {
	Default().Sugar().Debugf(format, args...)
}

func Infof(format string, args ...any) {
	Default().Sugar().Infof(format, args...)
}

func Warnf(format string, args ...any) {
	Default().Sugar().Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	Default().Sugar().Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	Default().Sugar().Fatalf(format, args...)
}

func Initialize(level string) error {
	// Logging is already initialized in Default()
	return nil
}

func Sync() {
	Default().Sync()
}

type Field = zap.Field

func String(key string, value string) Field {
	return zap.String(key, value)
}
