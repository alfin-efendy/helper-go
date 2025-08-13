package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

var (
	defaultLogger *Logger
	once          sync.Once
	mu            sync.RWMutex
)

const (
	// Field keys for structured logging
	TraceIDKey    = "traceID"
	SpanIDKey     = "spanID"
	CallerFileKey = "file"
	CallerFuncKey = "func"
	CallerLineKey = "line"

	// Default caller skip depth
	defaultCallerDepth = 3
)

// validateLogPath validates that the log file path is safe
func validateLogPath(path string) error {
	if path == "" {
		return fmt.Errorf("log path cannot be empty")
	}

	// Clean the path to resolve any relative path components
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("log path contains directory traversal: %s", path)
	}

	// Ensure the path is absolute or relative to current directory (no leading /)
	// This prevents writing to system directories
	if filepath.IsAbs(cleanPath) {
		// Allow absolute paths only in common log directories
		allowedPrefixes := []string{
			"/var/log/",
			"/tmp/",
			"/home/",
			"/var/folders/", // macOS temporary directories
			// Windows paths
			"C:\\Users\\",
			"C:/Users/",
			"C:\\temp\\",
			"C:/temp/",
			"C:\\tmp\\",
			"C:/tmp/",
		}

		// Add OS-specific temp directory
		if tempDir := os.TempDir(); tempDir != "" {
			allowedPrefixes = append(allowedPrefixes, tempDir)
			// Also add with both slash types for Windows
			if runtime.GOOS == "windows" {
				normalizedTemp := strings.ReplaceAll(tempDir, "\\", "/")
				if normalizedTemp != tempDir {
					allowedPrefixes = append(allowedPrefixes, normalizedTemp)
				}
			}
		}
		allowed := false
		for _, prefix := range allowedPrefixes {
			// Check both the original path and normalized versions for cross-platform compatibility
			if strings.HasPrefix(cleanPath, prefix) {
				allowed = true
				break
			}
			// On Windows, also check with normalized slashes
			if runtime.GOOS == "windows" {
				normalizedPath := strings.ReplaceAll(cleanPath, "\\", "/")
				normalizedPrefix := strings.ReplaceAll(prefix, "\\", "/")
				if strings.HasPrefix(normalizedPath, normalizedPrefix) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			return fmt.Errorf("absolute log path not allowed: %s", path)
		}
	}

	return nil
}

// Logger wraps zerolog with enhanced functionality
type Logger struct {
	logger  zerolog.Logger
	level   zerolog.Level
	closers []io.Closer
	mu      sync.RWMutex
}

// LoggerOption allows for functional configuration
type LoggerOption func(*Logger) error

// Config represents logger configuration
type Config struct {
	Level        string
	Location     string
	AppName      string
	EnableCaller bool
	CallerDepth  int
}

// NewLogger creates a new Logger instance with options
func NewLogger(opts ...LoggerOption) (*Logger, error) {
	l := &Logger{
		level:   zerolog.InfoLevel,
		closers: make([]io.Closer, 0),
		mu:      sync.RWMutex{},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return nil, fmt.Errorf("failed to apply logger option: %w", err)
		}
	}

	return l, nil
}

// WithConfig configures logger from config package
func WithConfig() LoggerOption {
	return func(l *Logger) error {
		cfg := config.Data
		if cfg == nil {
			return fmt.Errorf("config data is nil")
		}

		return WithConfigStruct(Config{
			Level:        cfg.Log.Level,
			Location:     cfg.Log.Location,
			AppName:      cfg.App.Name,
			EnableCaller: true,
			CallerDepth:  defaultCallerDepth,
		})(l)
	}
}

// WithConfigStruct configures logger from Config struct
func WithConfigStruct(cfg Config) LoggerOption {
	return func(l *Logger) error {
		// Parse log level
		if cfg.Level != "" {
			level, err := zerolog.ParseLevel(cfg.Level)
			if err != nil {
				return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
			}
			l.level = level
		}

		// Create writers
		writers := make([]io.Writer, 0, 2)

		// Console writer
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)

		// File writer (if specified)
		if cfg.Location != "" {
			if err := validateLogPath(cfg.Location); err != nil {
				return fmt.Errorf("invalid log file path: %w", err)
			}

			fileWriter, err := os.OpenFile(cfg.Location, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err != nil {
				return fmt.Errorf("failed to open log file %q: %w", cfg.Location, err)
			}
			writers = append(writers, fileWriter)
			l.closers = append(l.closers, fileWriter)
		}

		// Create multi-writer
		multiWriter := zerolog.MultiLevelWriter(writers...)

		// Build logger context
		loggerCtx := zerolog.New(multiWriter).With().Timestamp()

		if cfg.AppName != "" {
			loggerCtx = loggerCtx.Str("app", cfg.AppName)
		}

		if cfg.EnableCaller {
			depth := cfg.CallerDepth
			if depth == 0 {
				depth = defaultCallerDepth
			}
			loggerCtx = loggerCtx.CallerWithSkipFrameCount(depth)
		}

		l.logger = loggerCtx.Logger().Level(l.level)
		zerolog.SetGlobalLevel(l.level)

		return nil
	}
}

// WithLevel sets the log level
func WithLevel(level zerolog.Level) LoggerOption {
	return func(l *Logger) error {
		l.level = level
		return nil
	}
}

// WithFile adds file output
func WithFile(filename string) LoggerOption {
	return func(l *Logger) error {
		// Validate filename to prevent directory traversal and other security issues
		if err := validateLogPath(filename); err != nil {
			return fmt.Errorf("invalid log file: %w", err)
		}

		// #nosec G304 - File path is validated by validateLogPath function above
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		l.closers = append(l.closers, file)
		return nil
	}
}

// Close closes all file handles
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, closer := range l.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d resources: %v", len(errs), errs)
	}
	return nil
}

// enrichLogger adds context information to the logger
func (l *Logger) enrichLogger(ctx context.Context) zerolog.Logger {
	l.mu.RLock()
	logger := l.logger
	l.mu.RUnlock()

	// Add OpenTelemetry trace information
	if span := trace.SpanFromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			logger = logger.With().
				Str(TraceIDKey, spanCtx.TraceID().String()).
				Str(SpanIDKey, spanCtx.SpanID().String()).
				Logger()
		}
	}

	if pc, file, line, ok := runtime.Caller(4); ok {
		logger = logger.With().
			Str(CallerFileKey, file).
			Str(CallerFuncKey, runtime.FuncForPC(pc).Name()).
			Int(CallerLineKey, line).
			Logger()

		return logger
	}

	return logger
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	if l.level > zerolog.DebugLevel {
		return
	}

	event := l.enrichLogger(ctx)
	l.logWithFields(event.Debug(), msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
	if l.level > zerolog.InfoLevel {
		return
	}

	event := l.enrichLogger(ctx)
	l.logWithFields(event.Info(), msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	if l.level > zerolog.WarnLevel {
		return
	}

	event := l.enrichLogger(ctx)
	l.logWithFields(event.Warn(), msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	if l.level > zerolog.ErrorLevel {
		return
	}

	logger := l.enrichLogger(ctx)
	event := logger.Error()
	if err != nil {
		event = event.Err(err)
	}
	l.logWithFields(event, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {
	if l.level > zerolog.FatalLevel {
		return
	}

	logger := l.enrichLogger(ctx)
	event := logger.Fatal()
	if err != nil {
		event = event.Err(err)
	}
	l.logWithFields(event, msg, fields...)
}

// logWithFields handles structured logging with key-value pairs
func (l *Logger) logWithFields(event *zerolog.Event, msg string, fields ...interface{}) {
	// Handle key-value pairs
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key, ok := fields[i].(string)
			if ok {
				event = event.Interface(key, fields[i+1])
			}
		}
	}

	event.Msg(msg)
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	baseLogger := l.logger
	l.mu.RUnlock()

	ctx := baseLogger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}

	newLogger := &Logger{
		logger:  ctx.Logger(),
		level:   l.level,
		closers: l.closers, // Share closers, don't duplicate
		mu:      sync.RWMutex{},
	}

	return newLogger
}

// GetDefault returns the default logger instance
func GetDefault() *Logger {
	once.Do(func() {
		logger, err := NewLogger(WithConfig())
		if err != nil {
			// Fallback to basic logger
			logger, _ = NewLogger(
				WithLevel(zerolog.InfoLevel),
				WithConfigStruct(Config{
					AppName:      "unknown",
					EnableCaller: true,
					CallerDepth:  defaultCallerDepth,
				}),
			)
		}
		defaultLogger = logger
	})

	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// SetDefault sets the default logger instance
func SetDefault(logger *Logger) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger != nil {
		if err := defaultLogger.Close(); err != nil {
			// Log the error but don't panic since this is a cleanup operation
			// Use standard log package as fallback since we're replacing the logger
			fmt.Fprintf(os.Stderr, "Warning: failed to close previous default logger: %v\n", err)
		}
	}
	defaultLogger = logger
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(ctx context.Context, msg string, fields ...interface{}) {
	GetDefault().Debug(ctx, msg, fields...)
}

// Info logs an info message using the default logger
func Info(ctx context.Context, msg string, fields ...interface{}) {
	GetDefault().Info(ctx, msg, fields...)
}

// Warn logs a warning message using the default logger
func Warn(ctx context.Context, msg string, fields ...interface{}) {
	GetDefault().Warn(ctx, msg, fields...)
}

// Error logs an error message using the default logger
func Error(ctx context.Context, err error, msg string, fields ...interface{}) {
	GetDefault().Error(ctx, err, msg, fields...)
}

// Fatal logs a fatal message using the default logger
func Fatal(ctx context.Context, err error, msg string, fields ...interface{}) {
	GetDefault().Fatal(ctx, err, msg, fields...)
}
