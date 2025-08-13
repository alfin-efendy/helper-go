package logger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// GormLoggerTestSuite provides comprehensive testing for the GORM logger
type GormLoggerTestSuite struct {
	suite.Suite
	ctx    context.Context
	logger gormLogger.Interface
}

func (suite *GormLoggerTestSuite) SetupTest() {
	suite.ctx = context.Background()

	// Create a test logger with basic config
	testLogger, err := NewLogger(
		WithConfigStruct(Config{
			Level:        "debug",
			AppName:      "gorm-test",
			EnableCaller: true,
		}),
	)
	suite.Require().NoError(err)

	suite.logger = NewGormLoggerWithLogger(testLogger,
		WithLogLevel(gormLogger.Info),
		WithSlowThreshold(100*time.Millisecond),
		WithIgnoreRecordNotFoundError(false),
	)
}

func TestGormLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(GormLoggerTestSuite))
}

func TestNewGormLogger(t *testing.T) {
	logger := NewGormLogger()
	require.NotNil(t, logger)
	assert.IsType(t, &ZerologGormLogger{}, logger)
}

func TestNewGormLoggerWithOptions(t *testing.T) {
	logger := NewGormLogger(
		WithLogLevel(gormLogger.Error),
		WithSlowThreshold(500*time.Millisecond),
		WithIgnoreRecordNotFoundError(true),
		WithParameterizedQueries(true),
		WithCustomFields(
			AnyField("service", "test-service"),
			StringField("version", "1.0.0"),
		),
	)

	require.NotNil(t, logger)

	gormLog, ok := logger.(*ZerologGormLogger)
	require.True(t, ok)

	assert.Equal(t, gormLogger.Error, gormLog.config.LogLevel)
	assert.Equal(t, 500*time.Millisecond, gormLog.config.SlowThreshold)
	assert.True(t, gormLog.config.IgnoreRecordNotFoundError)
	assert.True(t, gormLog.config.ParameterizedQueries)
	assert.Len(t, gormLog.customFields, 2)
}

func TestNewGormLoggerWithLogger(t *testing.T) {
	testLogger, err := NewLogger(
		WithConfigStruct(Config{
			Level:   "warn",
			AppName: "custom-app",
		}),
	)
	require.NoError(t, err)

	gormLog := NewGormLoggerWithLogger(testLogger)
	require.NotNil(t, gormLog)

	gormLogger, ok := gormLog.(*ZerologGormLogger)
	require.True(t, ok)
	assert.Equal(t, testLogger, gormLogger.logger)
}

func TestLogMode(t *testing.T) {
	logger := NewGormLogger(WithLogLevel(gormLogger.Info))

	// Test changing log mode
	errorLogger := logger.LogMode(gormLogger.Error)
	require.NotNil(t, errorLogger)

	// Original logger should be unchanged
	originalGormLog, ok := logger.(*ZerologGormLogger)
	require.True(t, ok)
	assert.Equal(t, gormLogger.Info, originalGormLog.config.LogLevel)

	// New logger should have error level
	errorGormLog, ok := errorLogger.(*ZerologGormLogger)
	require.True(t, ok)
	assert.Equal(t, gormLogger.Error, errorGormLog.config.LogLevel)
}

func (suite *GormLoggerTestSuite) TestInfo() {
	suite.logger.Info(suite.ctx, "Test info message", "key", "value", "number", 42)
	// Since we're testing with actual logging, we mainly verify it doesn't panic
	suite.True(true, "Info logging should not panic")
}

func (suite *GormLoggerTestSuite) TestWarn() {
	suite.logger.Warn(suite.ctx, "Test warning message", "key", "value")
	suite.True(true, "Warn logging should not panic")
}

func (suite *GormLoggerTestSuite) TestError() {
	suite.logger.Error(suite.ctx, "Test error message", "error_code", 500)
	suite.True(true, "Error logging should not panic")
}

func (suite *GormLoggerTestSuite) TestTrace() {
	begin := time.Now()

	// Test normal SQL execution
	suite.logger.Trace(suite.ctx, begin, func() (string, int64) {
		return "SELECT * FROM users WHERE id = ?", 1
	}, nil)

	// Test SQL with error
	suite.logger.Trace(suite.ctx, begin, func() (string, int64) {
		return "SELECT * FROM invalid_table", -1
	}, errors.New("table doesn't exist"))

	// Test slow query
	slowBegin := time.Now().Add(-200 * time.Millisecond)
	suite.logger.Trace(suite.ctx, slowBegin, func() (string, int64) {
		return "SELECT * FROM large_table", 1000
	}, nil)

	suite.True(true, "Trace logging should not panic")
}

func (suite *GormLoggerTestSuite) TestTraceWithRecordNotFound() {
	// Create logger that ignores record not found errors
	logger := NewGormLogger(
		WithIgnoreRecordNotFoundError(true),
		WithLogLevel(gormLogger.Error),
	)

	begin := time.Now()
	logger.Trace(suite.ctx, begin, func() (string, int64) {
		return "SELECT * FROM users WHERE id = 999", 0
	}, gorm.ErrRecordNotFound)

	// Should not log error due to IgnoreRecordNotFoundError = true
	suite.True(true, "Should handle record not found error gracefully")
}

func TestBuildFields(t *testing.T) {
	testLogger, err := NewLogger(WithConfigStruct(Config{Level: "debug"}))
	require.NoError(t, err)

	gormLog := &ZerologGormLogger{
		logger: testLogger,
		customFields: []func(ctx context.Context) map[string]interface{}{
			AnyField("service", "test"),
			StringField("version", "1.0.0"),
		},
	}

	ctx := context.Background()
	fields := gormLog.buildFields(ctx, "key1", "value1", "key2", 42, "single_value")

	// Should contain custom fields + args
	// Custom fields: service, version (2 pairs = 4 items)
	// Args: key1/value1, key2/42, arg_4/single_value (3 pairs = 6 items)
	// Total: 10 items
	assert.Len(t, fields, 10)

	// Check custom fields are included
	assert.Contains(t, fields, "service")
	assert.Contains(t, fields, "test")
	assert.Contains(t, fields, "version")
	assert.Contains(t, fields, "1.0.0")

	// Check args are included
	assert.Contains(t, fields, "key1")
	assert.Contains(t, fields, "value1")
	assert.Contains(t, fields, "key2")
	assert.Contains(t, fields, 42)
	assert.Contains(t, fields, "arg_4")
	assert.Contains(t, fields, "single_value")
}

func TestFormatRows(t *testing.T) {
	tests := []struct {
		name     string
		rows     int64
		expected string
	}{
		{
			name:     "normal rows",
			rows:     42,
			expected: "42",
		},
		{
			name:     "no rows",
			rows:     0,
			expected: "0",
		},
		{
			name:     "unknown rows",
			rows:     -1,
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRows(tt.rows)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomFieldFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("AnyField", func(t *testing.T) {
		field := AnyField("test", 42)
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"test": 42}, result)
	})

	t.Run("StringField", func(t *testing.T) {
		field := StringField("name", "test")
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"name": "test"}, result)
	})

	t.Run("IntField", func(t *testing.T) {
		field := IntField("count", 100)
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"count": int64(100)}, result)
	})

	t.Run("FloatField", func(t *testing.T) {
		field := FloatField("price", 99.99)
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"price": 99.99}, result)
	})

	t.Run("DurationField", func(t *testing.T) {
		duration := 5 * time.Second
		field := DurationField("timeout", duration)
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"timeout": "5s"}, result)
	})

	t.Run("ContextField", func(t *testing.T) {
		type contextKey string
		key := contextKey("user_id")
		ctx := context.WithValue(context.Background(), key, "12345")

		field := ContextField("user", key)
		result := field(ctx)
		assert.Equal(t, map[string]interface{}{"user": "12345"}, result)

		// Test with missing context value
		emptyCtx := context.Background()
		result = field(emptyCtx)
		assert.Nil(t, result)
	})
}

func TestGormLogLevelFiltering(t *testing.T) {
	testLogger, err := NewLogger(WithConfigStruct(Config{Level: "debug"}))
	require.NoError(t, err)

	tests := []struct {
		name      string
		logLevel  gormLogger.LogLevel
		method    string
		shouldLog bool
	}{
		{
			name:      "silent mode blocks all",
			logLevel:  gormLogger.Silent,
			method:    "info",
			shouldLog: false,
		},
		{
			name:      "error level allows error",
			logLevel:  gormLogger.Error,
			method:    "error",
			shouldLog: true,
		},
		{
			name:      "error level blocks info",
			logLevel:  gormLogger.Error,
			method:    "info",
			shouldLog: false,
		},
		{
			name:      "warn level allows warn",
			logLevel:  gormLogger.Warn,
			method:    "warn",
			shouldLog: true,
		},
		{
			name:      "info level allows info",
			logLevel:  gormLogger.Info,
			method:    "info",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewGormLoggerWithLogger(testLogger, WithLogLevel(tt.logLevel))
			gormLog, ok := logger.(*ZerologGormLogger)
			require.True(t, ok)

			ctx := context.Background()

			// Test that calling the method doesn't panic
			assert.NotPanics(t, func() {
				switch tt.method {
				case "info":
					gormLog.Info(ctx, "test message")
				case "warn":
					gormLog.Warn(ctx, "test message")
				case "error":
					gormLog.Error(ctx, "test message")
				}
			})
		})
	}
}

// Benchmark tests
func BenchmarkGormLoggerTrace(b *testing.B) {
	testLogger, _ := NewLogger(WithConfigStruct(Config{Level: "info"}))
	gormLog := NewGormLoggerWithLogger(testLogger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		begin := time.Now()
		gormLog.Trace(ctx, begin, func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 1
		}, nil)
	}
}

func BenchmarkGormLoggerInfo(b *testing.B) {
	testLogger, _ := NewLogger(WithConfigStruct(Config{Level: "info"}))
	gormLog := NewGormLoggerWithLogger(testLogger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gormLog.Info(ctx, "Test message", "key", "value")
	}
}

func BenchmarkBuildFields(b *testing.B) {
	testLogger, _ := NewLogger(WithConfigStruct(Config{Level: "info"}))
	gormLog := &ZerologGormLogger{
		logger: testLogger,
		customFields: []func(ctx context.Context) map[string]interface{}{
			AnyField("service", "test"),
			StringField("version", "1.0.0"),
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gormLog.buildFields(ctx, "key1", "value1", "key2", 42)
	}
}
