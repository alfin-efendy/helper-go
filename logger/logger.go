package logger

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/utility"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggerInstance Logger
	level          zapcore.Level
)

const (
	TraceIdKey      = "traceID"
	SpanIdKey       = "spanID"
	SpanParentIdKey = "spanParentID"
	CallerFileKey   = "callerFile"
	CallerFuncKey   = "callerFunc"
	CallerLineKey   = "callerLine"
)

type Logger interface {
	GetLevel() zapcore.Level
	GetZapLogger() *zap.Logger
	Printf(format string, v ...interface{})
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Warn(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, err error, msg string)
	Fatal(ctx context.Context, err error, msg string)
	Panic(ctx context.Context, err error, msg string)
}

type logger struct {
	log *zap.Logger
}

func NewLogger(log *zap.Logger) Logger {
	return &logger{log: log}
}

func Init() {
	var location string
	if config.Config.Log.Location != nil {
		location = *config.Config.Log.Location
	} else {
		location = os.TempDir() + "/logs/" + "app.log"
	}

	os.MkdirAll(location, os.ModePerm)

	// Set retention policy for logs
	fileWriter := &lumberjack.Logger{
		Filename:   location,
		MaxSize:    config.Config.Log.MaxSize,
		MaxAge:     config.Config.Log.MaxAge,
		MaxBackups: config.Config.Log.MaxBackups,
		Compress:   config.Config.Log.Compress,
	}

	// Build log configuration
	zapConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set log level
	err := level.UnmarshalText([]byte(config.Config.Log.Level))
	if err != nil {
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapConfig),
		zapcore.AddSync(fileWriter),
		level,
	)

	// Create logger
	logger := zap.New(core)
	loggerInstance = NewLogger(logger)
}

func addTraceEntries(ctx context.Context, logger *zap.Logger) *zap.Logger {
	sc := trace.SpanContextFromContext(ctx)
	newLogger := logger.With(
		zap.String(TraceIdKey, sc.TraceID().String()),
		zap.String(SpanIdKey, sc.SpanID().String()),
		zap.String(SpanParentIdKey, sc.TraceID().String()),
	)
	return newLogger
}

func addCallerEntries(logger *zap.Logger) *zap.Logger {
	if pc, file, line, ok := runtime.Caller(4); ok {
		newLogger := logger.With(
			zap.String(CallerFileKey, file),
			zap.String(CallerFuncKey, runtime.FuncForPC(pc).Name()),
			zap.Int(CallerLineKey, line),
		)

		return newLogger
	}
	return logger
}

// StdEntries Return entries with trace ID entry from span context,
// span ID entry from span context, and
// span parent ID entry from context
func stdEntries(ctx context.Context, logger *zap.Logger) *zap.Logger {
	logger = addTraceEntries(ctx, logger)
	logger = addCallerEntries(logger)
	return logger
}

func (l *logger) GetLevel() zapcore.Level {
	return level
}

func GetLevel() zapcore.Level {
	return loggerInstance.GetLevel()
}

func (l *logger) GetZapLogger() *zap.Logger {
	return l.log
}

func GetZapLogger() *zap.Logger {
	return loggerInstance.GetZapLogger()
}

func (l *logger) Printf(format string, v ...interface{}) {
	utility.PrintInfo(fmt.Sprintf(format, v...))
	stdEntries(context.Background(), l.log).Info(fmt.Sprintf(format, v...))
}

func Printf(format string, v ...interface{}) {
	loggerInstance.Printf(format, v...)
}

func (l *logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	utility.PrintInfo(fmt.Sprint(msg))
	stdEntries(ctx, l.log).Info(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	loggerInstance.Info(ctx, msg, fields...)
}

func (l *logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	utility.PrintWarning(fmt.Sprint(msg))
	stdEntries(ctx, l.log).Warn(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	loggerInstance.Warn(ctx, msg, fields...)
}

func (l *logger) Error(ctx context.Context, err error, msg string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", msg, err))

	stdEntries(ctx, l.log).Error(msg, zap.Error(err))
}

func Error(ctx context.Context, err error, msg ...string) {
	message := ""

	if len(msg) > 0 {
		message = msg[0]
	}

	loggerInstance.Error(ctx, err, message)
}

func (l *logger) Fatal(ctx context.Context, err error, msg string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", msg, err))

	stdEntries(ctx, l.log).Fatal(msg, zap.Error(err))
}

func Fatal(ctx context.Context, err error, msg ...string) {
	message := ""

	if len(msg) > 0 {
		message = msg[0]
	}

	loggerInstance.Fatal(ctx, err, message)
}

func (l *logger) Panic(ctx context.Context, err error, msg string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", msg, err))

	stdEntries(ctx, l.log).Panic(msg, zap.Error(err))
}

func Panic(ctx context.Context, err error, msg ...string) {
	message := ""

	if len(msg) > 0 {
		message = msg[0]
	}

	loggerInstance.Panic(ctx, err, message)
}
