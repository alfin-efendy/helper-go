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
	Info(ctx context.Context, msg string, args ...zap.Field)
	Warn(ctx context.Context, msg string, args ...zap.Field)
	Error(ctx context.Context, err error, args ...zap.Field)
	Fatal(ctx context.Context, err error, args ...zap.Field)
	Panic(ctx context.Context, err error, args ...zap.Field)
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
	zapConfig := zap.NewProductionEncoderConfig()

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

func (l *logger) Info(ctx context.Context, msg string, args ...zap.Field) {
	utility.PrintInfo(fmt.Sprint(msg))
	stdEntries(ctx, l.log).Info(msg, args...)
}

func Info(ctx context.Context, msg string, args ...zap.Field) {
	loggerInstance.Info(ctx, msg, args...)
}

func (l *logger) Warn(ctx context.Context, msg string, args ...zap.Field) {
	utility.PrintWarning(fmt.Sprint(msg))
	stdEntries(ctx, l.log).Warn(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...zap.Field) {
	loggerInstance.Warn(ctx, msg, args...)
}

func (l *logger) Error(ctx context.Context, err error, args ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(err.Error())

	args = append(args, zap.Error(err))

	stdEntries(ctx, l.log).Error(err.Error(), args...)
}

func Error(ctx context.Context, err error, msg ...string) {
	var fields []zap.Field

	for _, m := range msg {
		fields = append(fields, zap.String("message", m))
	}

	loggerInstance.Error(ctx, err, fields...)
}

func (l *logger) Fatal(ctx context.Context, err error, args ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	args = append(args, zap.Error(err))

	stdEntries(ctx, l.log).Fatal(err.Error(), args...)
}

func Fatal(ctx context.Context, err error, msg ...string) {
	var fields []zap.Field

	for _, m := range msg {
		fields = append(fields, zap.String("message", m))
	}

	loggerInstance.Fatal(ctx, err, fields...)
}

func (l *logger) Panic(ctx context.Context, err error, args ...zap.Field) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(err.Error())

	args = append(args, zap.Error(err))

	stdEntries(ctx, l.log).Panic(err.Error(), args...)
}

func Panic(ctx context.Context, err error, msg ...string) {
	var fields []zap.Field

	for _, m := range msg {
		fields = append(fields, zap.String("message", m))
	}

	loggerInstance.Panic(ctx, err, fields...)
}
