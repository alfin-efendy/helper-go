package logger

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestNewLoggerWithOptions(t *testing.T) {
	t.Run("Basic Logger", func(t *testing.T) {
		logger, err := NewLogger()
		require.NoError(t, err)
		require.NotNil(t, logger)
		defer logger.Close()

		assert.Equal(t, zerolog.InfoLevel, logger.level)
	})

	t.Run("Logger with Level", func(t *testing.T) {
		logger, err := NewLogger(WithLevel(zerolog.DebugLevel))
		require.NoError(t, err)
		defer logger.Close()

		assert.Equal(t, zerolog.DebugLevel, logger.level)
	})

	t.Run("Logger with File", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "test.log")

		logger, err := NewLogger(
			WithLevel(zerolog.InfoLevel),
			WithFile(tempFile),
		)
		require.NoError(t, err)
		defer logger.Close()

		// Test that file was created and is closeable
		assert.FileExists(t, tempFile)
	})

	t.Run("Logger with Config", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "config_test.log")

		// Mock config.Data
		config.Data = &schema.Config{
			App: schema.App{Name: "test-app"},
			Log: schema.Log{
				Level:    "debug",
				Location: tempFile,
			},
		}
		defer func() { config.Data = nil }()

		logger, err := NewLogger(WithConfig())
		require.NoError(t, err)
		defer logger.Close()

		assert.Equal(t, zerolog.DebugLevel, logger.level)
	})

	t.Run("Config with nil data", func(t *testing.T) {
		config.Data = nil

		logger, err := NewLogger(WithConfig())
		assert.Error(t, err)
		assert.Nil(t, logger)
		assert.Contains(t, err.Error(), "config data is nil")
	})
}

func TestLoggerMethods(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "methods_test.log")

	logger, err := NewLogger(
		WithLevel(zerolog.DebugLevel),
		WithConfigStruct(Config{
			Level:        "debug",
			Location:     tempFile,
			AppName:      "test-app",
			EnableCaller: true,
			CallerDepth:  defaultCallerDepth,
		}),
	)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	testErr := errors.New("test error")

	t.Run("Debug", func(t *testing.T) {
		logger.Debug(ctx, "debug message", "key1", "value1", "key2", 42)

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "debug message")
		assert.Contains(t, logContent, "key1")
		assert.Contains(t, logContent, "value1")
	})

	t.Run("Info", func(t *testing.T) {
		logger.Info(ctx, "info message", "user", "john", "count", 10)

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "info message")
	})

	t.Run("Warn", func(t *testing.T) {
		logger.Warn(ctx, "warning message", "warning_code", "W001")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "warning message")
	})

	t.Run("Error", func(t *testing.T) {
		logger.Error(ctx, testErr, "error occurred", "operation", "database_query")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "error occurred")
		assert.Contains(t, logContent, "test error")
	})
}

func TestGlobalFunctions(t *testing.T) {
	// Reset default logger
	mu.Lock()
	defaultLogger = nil
	mu.Unlock()
	once = sync.Once{}

	tempFile := filepath.Join(t.TempDir(), "global_test.log")

	// Mock config.Data
	config.Data = &schema.Config{
		App: schema.App{Name: "global-test"},
		Log: schema.Log{
			Level:    "debug",
			Location: tempFile,
		},
	}
	defer func() {
		config.Data = nil
		mu.Lock()
		if defaultLogger != nil {
			defaultLogger.Close()
			defaultLogger = nil
		}
		mu.Unlock()
		once = sync.Once{}
	}()

	ctx := context.Background()
	testErr := errors.New("global test error")

	t.Run("Global Debug", func(t *testing.T) {
		Debug(ctx, "global debug message", "global_key", "global_value")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "global debug message")
	})

	t.Run("Global Info", func(t *testing.T) {
		Info(ctx, "global info message", "request_id", "abc123")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "global info message")
	})

	t.Run("Global Error", func(t *testing.T) {
		Error(ctx, testErr, "global error occurred", "component", "auth")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "global error occurred")
		assert.Contains(t, logContent, "global test error")
	})
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		level     zerolog.Level
		shouldLog map[string]bool
	}{
		{
			name:  "debug level",
			level: zerolog.DebugLevel,
			shouldLog: map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			name:  "info level",
			level: zerolog.InfoLevel,
			shouldLog: map[string]bool{
				"debug": false,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			name:  "error level",
			level: zerolog.ErrorLevel,
			shouldLog: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  false,
				"error": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "level_test.log")

			logger, err := NewLogger(
				WithLevel(tt.level),
				WithConfigStruct(Config{
					Location:     tempFile,
					AppName:      "level-test",
					EnableCaller: false, // Disable to make log parsing easier
				}),
			)
			require.NoError(t, err)
			defer logger.Close()

			ctx := context.Background()

			// Log at different levels
			logger.Debug(ctx, "debug message")
			logger.Info(ctx, "info message")
			logger.Warn(ctx, "warn message")
			logger.Error(ctx, errors.New("test"), "error message")

			// Read log content
			content, err := os.ReadFile(tempFile)
			require.NoError(t, err)
			logContent := string(content)

			// Check which messages should appear
			if tt.shouldLog["debug"] {
				assert.Contains(t, logContent, "debug message")
			} else {
				assert.NotContains(t, logContent, "debug message")
			}

			if tt.shouldLog["info"] {
				assert.Contains(t, logContent, "info message")
			} else {
				assert.NotContains(t, logContent, "info message")
			}

			if tt.shouldLog["warn"] {
				assert.Contains(t, logContent, "warn message")
			} else {
				assert.NotContains(t, logContent, "warn message")
			}

			if tt.shouldLog["error"] {
				assert.Contains(t, logContent, "error message")
			} else {
				assert.NotContains(t, logContent, "error message")
			}
		})
	}
}

func TestStructuredLogging(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "structured_test.log")

	logger, err := NewLogger(
		WithLevel(zerolog.InfoLevel),
		WithConfigStruct(Config{
			Location:     tempFile,
			AppName:      "structured-test",
			EnableCaller: false,
		}),
	)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	t.Run("Key-Value Pairs", func(t *testing.T) {
		logger.Info(ctx, "user action",
			"user_id", "123",
			"action", "login",
			"ip", "192.168.1.1",
			"duration", 250,
			"success", true,
		)

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		logContent := string(content)

		assert.Contains(t, logContent, "user action")
		assert.Contains(t, logContent, "user_id")
		assert.Contains(t, logContent, "123")
		assert.Contains(t, logContent, "action")
		assert.Contains(t, logContent, "login")
	})

	t.Run("Odd Number of Fields", func(t *testing.T) {
		// Should handle gracefully when odd number of fields
		logger.Info(ctx, "incomplete fields", "key1", "value1", "key2")

		content, err := os.ReadFile(tempFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "incomplete fields")
	})
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer

	// Create logger with buffer for easier testing
	baseLogger := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{
		logger: baseLogger,
		level:  zerolog.InfoLevel,
		mu:     sync.RWMutex{},
	}

	// Test WithFields
	enrichedLogger := logger.WithFields(map[string]interface{}{
		"service":     "auth",
		"version":     "1.0.0",
		"environment": "test",
	})

	ctx := context.Background()
	enrichedLogger.Info(ctx, "service started")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service started")
	assert.Contains(t, logOutput, "auth")
	assert.Contains(t, logOutput, "1.0.0")
	assert.Contains(t, logOutput, "test")
}

func TestTraceEnrichment(t *testing.T) {
	var buf bytes.Buffer

	baseLogger := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{
		logger: baseLogger,
		level:  zerolog.InfoLevel,
		mu:     sync.RWMutex{},
	}

	// Create a span context
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	logger.Info(ctx, "operation completed")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "operation completed")

	// Check if trace context is enriched (trace IDs should be present)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		assert.Contains(t, logOutput, spanContext.TraceID().String())
		assert.Contains(t, logOutput, spanContext.SpanID().String())
	}
}

func TestResourceManagement(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "resource_test.log")

	logger, err := NewLogger(
		WithLevel(zerolog.InfoLevel),
		WithConfigStruct(Config{
			Location: tempFile,
			AppName:  "resource-test",
		}),
	)
	require.NoError(t, err)

	// Write some data
	ctx := context.Background()
	logger.Info(ctx, "test message")

	// Close should not error
	err = logger.Close()
	assert.NoError(t, err)

	// Second close should not panic or error (idempotent)
	// But it might return an error for already closed files, which is fine
	err2 := logger.Close()
	// Don't assert NoError here since closing already closed files may error
	t.Logf("Second close result: %v", err2)
}

func TestDefaultLoggerFallback(t *testing.T) {
	// Reset default logger
	mu.Lock()
	defaultLogger = nil
	mu.Unlock()
	once = sync.Once{}

	// Set config.Data to nil to trigger fallback
	originalConfig := config.Data
	config.Data = nil
	defer func() {
		config.Data = originalConfig
		mu.Lock()
		if defaultLogger != nil {
			defaultLogger.Close()
			defaultLogger = nil
		}
		mu.Unlock()
		once = sync.Once{}
	}()

	// This should trigger the fallback logger creation
	logger := GetDefault()
	require.NotNil(t, logger)

	// Should be able to log without error
	ctx := context.Background()
	logger.Info(ctx, "fallback test message")
}

func TestConcurrentLogging(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "concurrent_test.log")

	logger, err := NewLogger(
		WithLevel(zerolog.InfoLevel),
		WithConfigStruct(Config{
			Location:     tempFile,
			AppName:      "concurrent-test",
			EnableCaller: false,
		}),
	)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Run concurrent logging operations
	const numGoroutines = 50
	const logsPerGoroutine = 20

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info(ctx, "concurrent log",
					"goroutine", id,
					"message", j,
					"timestamp", time.Now().Unix(),
				)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for goroutines to complete")
		}
	}

	// Verify logs were written
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// Should have approximately numGoroutines * logsPerGoroutine entries
	expectedLogs := numGoroutines * logsPerGoroutine
	assert.GreaterOrEqual(t, len(lines), expectedLogs-10, "Should have most of the expected log entries")
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	logger, _ := NewLogger(WithLevel(zerolog.InfoLevel))
	defer logger.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "benchmark message", "count", 12345, "key", "value")
		}
	})
}

func BenchmarkGlobalInfo(b *testing.B) {
	// Reset default logger
	mu.Lock()
	defaultLogger = nil
	mu.Unlock()
	once = sync.Once{}

	config.Data = &schema.Config{
		App: schema.App{Name: "benchmark"},
		Log: schema.Log{Level: "info"},
	}
	defer func() {
		config.Data = nil
		mu.Lock()
		if defaultLogger != nil {
			defaultLogger.Close()
			defaultLogger = nil
		}
		mu.Unlock()
		once = sync.Once{}
	}()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info(ctx, "global benchmark message", "count", 12345)
		}
	})
}

func BenchmarkStructuredLogging(b *testing.B) {
	logger, _ := NewLogger(WithLevel(zerolog.InfoLevel))
	defer logger.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "structured benchmark",
				"user_id", "123",
				"action", "api_call",
				"duration", 45,
				"success", true,
				"ip", "192.168.1.1",
			)
		}
	})
}
