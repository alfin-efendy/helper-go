package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// ZerologGormLogger integrates GORM with our Zerolog-based logger
type ZerologGormLogger struct {
	logger       *Logger
	config       gormLogger.Config
	customFields []func(ctx context.Context) map[string]interface{}
}

// GormOption configures the GORM logger
type GormOption func(l *ZerologGormLogger)

// WithCustomFields adds custom fields to log entries
func WithCustomFields(fields ...func(ctx context.Context) map[string]interface{}) GormOption {
	return func(l *ZerologGormLogger) {
		l.customFields = fields
	}
}

// WithSlowThreshold sets the slow query threshold
func WithSlowThreshold(threshold time.Duration) GormOption {
	return func(l *ZerologGormLogger) {
		l.config.SlowThreshold = threshold
	}
}

// WithLogLevel sets the GORM log level
func WithLogLevel(level gormLogger.LogLevel) GormOption {
	return func(l *ZerologGormLogger) {
		l.config.LogLevel = level
	}
}

// WithIgnoreRecordNotFoundError configures whether to ignore record not found errors
func WithIgnoreRecordNotFoundError(ignore bool) GormOption {
	return func(l *ZerologGormLogger) {
		l.config.IgnoreRecordNotFoundError = ignore
	}
}

// WithParameterizedQueries enables/disables parameterized query logging
func WithParameterizedQueries(enabled bool) GormOption {
	return func(l *ZerologGormLogger) {
		l.config.ParameterizedQueries = enabled
	}
}

// NewGormLogger creates a new GORM logger integrated with Zerolog
func NewGormLogger(opts ...GormOption) gormLogger.Interface {
	l := &ZerologGormLogger{
		logger: GetDefault(),
		config: gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      false,
			LogLevel:                  gormLogger.Info,
		},
		customFields: make([]func(ctx context.Context) map[string]interface{}, 0),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// NewGormLoggerWithLogger creates a GORM logger with a specific logger instance
func NewGormLoggerWithLogger(logger *Logger, opts ...GormOption) gormLogger.Interface {
	l := &ZerologGormLogger{
		logger: logger,
		config: gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			ParameterizedQueries:      false,
			LogLevel:                  gormLogger.Info,
		},
		customFields: make([]func(ctx context.Context) map[string]interface{}, 0),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LogMode sets the log level and returns a new logger instance
func (l *ZerologGormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	newLogger := *l
	newLogger.config.LogLevel = level
	return &newLogger
}

// Info logs informational messages
func (l *ZerologGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.config.LogLevel < gormLogger.Info {
		return
	}

	fields := l.buildFields(ctx, args...)
	l.logger.Info(ctx, msg, fields...)
}

// Warn logs warning messages
func (l *ZerologGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.config.LogLevel < gormLogger.Warn {
		return
	}

	fields := l.buildFields(ctx, args...)
	l.logger.Warn(ctx, msg, fields...)
}

// Error logs error messages
func (l *ZerologGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.config.LogLevel < gormLogger.Error {
		return
	}

	fields := l.buildFields(ctx, args...)

	var err error
	if msg != "" {
		err = errors.New(msg)
	}

	l.logger.Error(ctx, err, msg, fields...)
}

// Trace logs SQL execution traces
func (l *ZerologGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.config.LogLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Build log fields
	fields := make([]interface{}, 0, 12+len(l.customFields)*2)

	// Add custom fields
	for _, customField := range l.customFields {
		if fieldMap := customField(ctx); fieldMap != nil {
			for k, v := range fieldMap {
				fields = append(fields, k, v)
			}
		}
	}

	// Add standard fields
	fields = append(fields,
		"file", utils.FileWithLineNum(),
		"elapsed", elapsed,
		"latency", elapsed.String(),
	)

	if rows == -1 {
		fields = append(fields, "rows", "-")
	} else {
		fields = append(fields, "rows", rows)
	}

	fields = append(fields, "sql", sql)

	// Determine log level and message
	switch {
	case err != nil && l.config.LogLevel >= gormLogger.Error &&
		(!l.config.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):

		fields = append(fields, "error", err.Error())
		msg := fmt.Sprintf("SQL Error [%v] [rows:%v] %s", elapsed, formatRows(rows), sql)
		l.logger.Error(ctx, err, msg, fields...)

	case elapsed > l.config.SlowThreshold && l.config.SlowThreshold != 0 && l.config.LogLevel >= gormLogger.Warn:
		fields = append(fields,
			"slow_query", true,
			"threshold", l.config.SlowThreshold.String(),
		)
		msg := fmt.Sprintf("SLOW SQL [%v] [rows:%v] %s", elapsed, formatRows(rows), sql)
		l.logger.Warn(ctx, msg, fields...)

	case l.config.LogLevel >= gormLogger.Info:
		msg := fmt.Sprintf("SQL [%v] [rows:%v] %s", elapsed, formatRows(rows), sql)
		l.logger.Info(ctx, msg, fields...)
	}
}

// buildFields converts args to key-value pairs for structured logging
func (l *ZerologGormLogger) buildFields(ctx context.Context, args ...interface{}) []interface{} {
	fields := make([]interface{}, 0, len(args)+len(l.customFields)*2)

	// Add custom fields first
	for _, customField := range l.customFields {
		if fieldMap := customField(ctx); fieldMap != nil {
			for k, v := range fieldMap {
				fields = append(fields, k, v)
			}
		}
	}

	// Handle args as key-value pairs or single values
	for i := 0; i < len(args); i++ {
		if i+1 < len(args) {
			// Try to use as key-value pair
			if key, ok := args[i].(string); ok {
				fields = append(fields, key, args[i+1])
				i++ // Skip next item as it's the value
				continue
			}
		}
		// Single value, use generic key
		fields = append(fields, fmt.Sprintf("arg_%d", i), args[i])
	}

	return fields
}

// formatRows formats row count for display
func formatRows(rows int64) string {
	if rows == -1 {
		return "-"
	}
	return fmt.Sprintf("%d", rows)
}

// Helper functions for creating custom fields

// AnyField creates a custom field function for any value
func AnyField(key string, value interface{}) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{key: value}
	}
}

// StringField creates a custom field function for string values
func StringField(key string, value string) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{key: value}
	}
}

// IntField creates a custom field function for integer values
func IntField(key string, value int64) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{key: value}
	}
}

// FloatField creates a custom field function for float values
func FloatField(key string, value float64) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{key: value}
	}
}

// DurationField creates a custom field function for duration values
func DurationField(key string, value time.Duration) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{key: value.String()}
	}
}

// ContextField creates a custom field function that extracts value from context
func ContextField(key string, contextKey interface{}) func(ctx context.Context) map[string]interface{} {
	return func(ctx context.Context) map[string]interface{} {
		if value := ctx.Value(contextKey); value != nil {
			return map[string]interface{}{key: value}
		}
		return nil
	}
}
