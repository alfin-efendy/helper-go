package logger

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/utility"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggerInstance Logger
	level          logrus.Level
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
	GetLevel() logrus.Level
	GetLogrusLogger() logrus.Ext1FieldLogger
	Info(ctx context.Context, args ...interface{})
	Warn(ctx context.Context, args ...interface{})
	Error(ctx context.Context, err error, args ...interface{})
	Fatal(ctx context.Context, err error, args ...interface{})
	Panic(ctx context.Context, err error, args ...interface{})
}

type logger struct {
	log logrus.Ext1FieldLogger
}

func NewLogger(log logrus.Ext1FieldLogger) Logger {
	return &logger{log: log}
}

func Init() {
	standardLogger := logrus.StandardLogger()

	var err error
	level, err = logrus.ParseLevel(config.Config.Log.Level)
	if err != nil {
		standardLogger.SetLevel(logrus.InfoLevel)
	} else {
		standardLogger.SetLevel(level)
	}

	standardLogger.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:     true,
		TimestampFormat: time.RFC3339,
	})

	var location string
	if config.Config.Log.Location != nil {
		location = *config.Config.Log.Location
	} else {
		location = os.TempDir() + "/logs/" + "app.log"
	}

	os.MkdirAll(location, os.ModePerm)

	standardLogger.SetOutput(&lumberjack.Logger{
		Filename:   location,
		MaxSize:    config.Config.Log.MaxSize,
		MaxAge:     config.Config.Log.MaxAge,
		MaxBackups: config.Config.Log.MaxBackups,
		Compress:   config.Config.Log.Compress,
	})

	loggerInstance = &logger{log: standardLogger}
}

func addTraceEntries(ctx context.Context, logger logrus.Ext1FieldLogger) logrus.Ext1FieldLogger {
	sc := trace.SpanContextFromContext(ctx)
	newLogger := logger.
		WithField(TraceIdKey, sc.TraceID().String()).
		WithField(SpanIdKey, sc.SpanID().String()).
		WithField(SpanParentIdKey, ctx.Value(SpanParentIdKey))
	return newLogger
}

func addCallerEntries(logger logrus.Ext1FieldLogger) logrus.Ext1FieldLogger {
	if pc, file, line, ok := runtime.Caller(4); ok {
		newLogger := logger.
			WithField(CallerFileKey, file).
			WithField(CallerFuncKey, runtime.FuncForPC(pc).Name()).
			WithField(CallerLineKey, line)
		return newLogger
	}
	return logger
}

// StdEntries Return entries with trace ID entry from span context,
// span ID entry from span context, and
// span parent ID entry from context
func stdEntries(ctx context.Context, logger logrus.Ext1FieldLogger) logrus.Ext1FieldLogger {
	logger = addTraceEntries(ctx, logger)
	logger = addCallerEntries(logger)
	return logger
}

func (l *logger) GetLevel() logrus.Level {
	return level
}

func GetLevel() logrus.Level {
	return loggerInstance.GetLevel()
}

func (l *logger) GetLogrusLogger() logrus.Ext1FieldLogger {
	return l.log
}

func GetLogrusLogger() logrus.Ext1FieldLogger {
	return loggerInstance.GetLogrusLogger()
}

func (l *logger) Info(ctx context.Context, args ...interface{}) {
	utility.PrintInfo(fmt.Sprint(args...))
	stdEntries(ctx, l.log).Info(args...)
}

func Info(ctx context.Context, args ...interface{}) {
	loggerInstance.Info(ctx, args...)
}

func (l *logger) Warn(ctx context.Context, args ...interface{}) {
	utility.PrintWarning(fmt.Sprint(args...))
	stdEntries(ctx, l.log).Warn(args...)
}

func Warn(ctx context.Context, args ...interface{}) {
	loggerInstance.Warn(ctx, args...)
}

func (l *logger) Error(ctx context.Context, err error, args ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", args, err))

	stdEntries(ctx, l.log).WithError(err).Error(args...)
}

func Error(ctx context.Context, err error, args ...interface{}) {
	loggerInstance.Error(ctx, err, args...)
}

func (l *logger) Fatal(ctx context.Context, err error, args ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", args, err))

	stdEntries(ctx, l.log).WithError(err).Fatal(args...)
}

func Fatal(ctx context.Context, err error, args ...interface{}) {
	loggerInstance.Fatal(ctx, err, args...)
}

func (l *logger) Panic(ctx context.Context, err error, args ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
	}

	utility.PrintError(fmt.Sprintf("%s: %v", args, err))

	stdEntries(ctx, l.log).WithError(err).Panic(args...)
}

func Panic(ctx context.Context, err error, args ...interface{}) {
	loggerInstance.Panic(ctx, err, args...)
}
