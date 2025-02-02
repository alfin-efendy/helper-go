package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// Logger logger for gorm2
type ZapGormLogger struct {
	log Logger
	gormLogger.Config
	customFields []func(ctx context.Context) zap.Field
}

// Option logger/recover option
type Option func(l *ZapGormLogger)

// WithCustomFields optional custom field
func WithCustomFields(fields ...func(ctx context.Context) zap.Field) Option {
	return func(l *ZapGormLogger) {
		l.customFields = fields
	}
}

// WithConfig optional custom logger.Config
func WithConfig(cfg gormLogger.Config) Option {
	return func(l *ZapGormLogger) {
		l.Config = cfg
	}
}

// New logger form gorm2
func New(opts ...Option) gormLogger.Interface {
	l := &ZapGormLogger{
		log: NewLogger(GetZapLogger()),
		Config: gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			LogLevel:                  gormLogger.Info,
		},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LogMode log mode
func (l *ZapGormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info print info
func (l ZapGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormLogger.Info {
		var fields []zap.Field
		for _, field := range args {
			switch v := field.(type) {
			case zap.Field:
				fields = append(fields, v)
			default:
				fields = append(fields, zap.Any("extra", v))
			}
		}

		l.log.Info(ctx, msg, fields...)
	}
}

// Warn print warn messages
func (l ZapGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormLogger.Warn {
		var fields []zap.Field
		for _, field := range args {
			switch v := field.(type) {
			case zap.Field:
				fields = append(fields, v)
			default:
				fields = append(fields, zap.Any("extra", v))
			}
		}

		l.log.Warn(ctx, msg, fields...)
	}
}

// Error print error messages
func (l ZapGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormLogger.Error {
		var fields []zap.Field
		for _, field := range args {
			switch v := field.(type) {
			case zap.Field:
				fields = append(fields, v)
			default:
				fields = append(fields, zap.Any("extra", v))
			}
		}

		var err error
		if msg != "" {
			err = errors.New(msg)
		}

		l.log.Error(ctx, err, fields...)
	}
}

// Trace print sql message
func (l ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}

	fields := make([]zap.Field, 0, 6+len(l.customFields))
	end := time.Now()
	latency := end.Sub(begin)
	elapsed := time.Since(begin)
	sql, rows := fc()

	for _, customField := range l.customFields {
		fields = append(fields, customField(ctx))
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	fields = append(fields,
		zap.String("file", utils.FileWithLineNum()),
		zap.Duration("latency", latency),
	)

	if rows == -1 {
		fields = append(fields, zap.String("rows", "-"))
	} else {
		fields = append(fields, zap.Int64("rows", rows))
	}

	fields = append(fields, zap.String("sql", sql))

	msg := fmt.Sprintf("[%v] [rows:%v] %s", latency, rows, sql)

	switch {
	case err != nil && l.LogLevel >= gormLogger.Error && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		l.log.Error(ctx, err, fields...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormLogger.Warn:
		fields = append(fields,
			zap.String("slow!!!", fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)),
		)

		l.log.Warn(ctx, msg, fields...)
	case l.LogLevel == gormLogger.Info:
		l.log.Info(ctx, msg, fields...)
	}
}

// Immutable custom immutable field
// Deprecated: use Any instead
func Immutable(key string, value interface{}) func(ctx context.Context) zap.Field {
	return Any(key, value)
}

// Any custom immutable any field
func Any(key string, value interface{}) func(ctx context.Context) zap.Field {
	field := zap.Any(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// String custom immutable string field
func String(key string, value string) func(ctx context.Context) zap.Field {
	field := zap.String(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Int64 custom immutable int64 field
func Int64(key string, value int64) func(ctx context.Context) zap.Field {
	field := zap.Int64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Uint64 custom immutable uint64 field
func Uint64(key string, value uint64) func(ctx context.Context) zap.Field {
	field := zap.Uint64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Float64 custom immutable float32 field
func Float64(key string, value float64) func(ctx context.Context) zap.Field {
	field := zap.Float64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}
