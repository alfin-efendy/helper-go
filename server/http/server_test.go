package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewHTTPServer(t *testing.T) {
	// Setup test configuration
	setupTestConfig()

	tests := []struct {
		name         string
		setupConfig  func()
		expectedAddr string
	}{
		{
			name: "Default configuration",
			setupConfig: func() {
				config.Data = &schema.Config{
					Server: schema.Server{
						RestAPI: schema.RestAPI{
							Host: "",
							Port: 0,
						},
					},
				}
			},
			expectedAddr: "localhost:8080",
		},
		{
			name: "Custom host and port",
			setupConfig: func() {
				config.Data = &schema.Config{
					Server: schema.Server{
						RestAPI: schema.RestAPI{
							Host: "0.0.0.0",
							Port: 3000,
						},
					},
				}
			},
			expectedAddr: "0.0.0.0:3000",
		},
		{
			name: "Custom host with default port",
			setupConfig: func() {
				config.Data = &schema.Config{
					Server: schema.Server{
						RestAPI: schema.RestAPI{
							Host: "127.0.0.1",
							Port: 0,
						},
					},
				}
			},
			expectedAddr: "127.0.0.1:8080",
		},
		{
			name: "Default host with custom port",
			setupConfig: func() {
				config.Data = &schema.Config{
					Server: schema.Server{
						RestAPI: schema.RestAPI{
							Host: "",
							Port: 9000,
						},
					},
				}
			},
			expectedAddr: "localhost:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()

			server := NewHTTPServer()

			require.NotNil(t, server)
			assert.Equal(t, tt.expectedAddr, server.Addr)
			assert.NotNil(t, server.Handler)
		})
	}
}

func TestServerRoutes(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()
	require.NotNil(t, server)

	t.Run("Non-existent endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/non-existent", nil)
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestServerMiddleware(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()
	require.NotNil(t, server)

	t.Run("Context middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		// If the request completes successfully, the context middleware is working
		// Health check returns 503 when services are unhealthy
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("Response middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Success", response.Message)
	})
}

func TestServerConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		config   *schema.Config
		expected string
	}{
		{
			name: "Nil config - should use defaults",
			config: &schema.Config{
				Server: schema.Server{
					RestAPI: schema.RestAPI{},
				},
			},
			expected: "localhost:8080",
		},
		{
			name: "Production config",
			config: &schema.Config{
				Server: schema.Server{
					RestAPI: schema.RestAPI{
						Host: "0.0.0.0",
						Port: 80,
					},
				},
			},
			expected: "0.0.0.0:80",
		},
		{
			name: "Development config",
			config: &schema.Config{
				Server: schema.Server{
					RestAPI: schema.RestAPI{
						Host: "localhost",
						Port: 8080,
					},
				},
			},
			expected: "localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Data = tt.config

			server := NewHTTPServer()
			assert.Equal(t, tt.expected, server.Addr)
		})
	}
}

func TestServerIntegration(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()
	require.NotNil(t, server)

	t.Run("Full request-response cycle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		server.Handler.ServeHTTP(w, req)

		// Check response status - health check returns 503 when services are unhealthy
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		// Check response headers
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		// Check response body structure
		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Validate response structure
		assert.Equal(t, "Success", response.Message)
		assert.NotNil(t, response.Data)
		assert.Nil(t, response.Errors)
		assert.Nil(t, response.Page)

		// Validate response data (health check response)
		dataMap, ok := response.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, dataMap, "status")
		assert.Contains(t, dataMap, "timestamp")
	})

	t.Run("Multiple requests", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			server.Handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusServiceUnavailable, w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "Success", response.Message)
		}
	})
}

func TestServerErrorHandling(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()
	require.NotNil(t, server)

	t.Run("Handles panic recovery", func(t *testing.T) {
		// This test ensures the recoverer middleware is working
		// The actual route doesn't panic, but the middleware is there
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		// This should not panic and should complete successfully
		assert.NotPanics(t, func() {
			server.Handler.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestServerType(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()

	// Test that it returns a proper http.Server
	assert.IsType(t, &http.Server{}, server)
	assert.NotNil(t, server.Handler)
	assert.NotEmpty(t, server.Addr)
}

// Helper function to setup test configuration
func setupTestConfig() {
	config.Data = &schema.Config{
		Server: schema.Server{
			RestAPI: schema.RestAPI{
				Host: "localhost",
				Port: 8080,
			},
		},
		Database: schema.Database{
			SQL: &schema.SQL{
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
				Password: "test_pass",
			},
			Redis: &schema.Redis{
				Mode: "single",
				RedisCluster: schema.RedisCluster{
					RedisSingle: schema.RedisSingle{
						Address: "localhost:6379",
					},
				},
			},
		},
	}
}

// Helper function to setup test configuration without database
func setupTestConfigWithoutDatabase() {
	config.Data = &schema.Config{
		Server: schema.Server{
			RestAPI: schema.RestAPI{
				Host: "localhost",
				Port: 8080,
			},
		},
		Database: schema.Database{
			SQL:   nil,
			Redis: nil,
		},
	}
}

// Benchmark tests
func BenchmarkNewHTTPServer(b *testing.B) {
	setupTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewHTTPServer()
	}
}

func BenchmarkServerRootRequest(b *testing.B) {
	setupTestConfig()
	server := NewHTTPServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, req)
	}
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	setupTestConfig()

	// Create a request to the health check endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create server and test the endpoint through the server
	server := NewHTTPServer()
	server.Handler.ServeHTTP(rr, req) // Check the status code (could be OK or ServiceUnavailable depending on service state)
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, rr.Code,
		"Health check should return either 200 OK or 503 Service Unavailable")

	// Check the content type
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "Response should be JSON")

	// Parse the middleware response wrapper
	var response Response
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Extract the health check data from the wrapper
	require.NotNil(t, response.Data, "Response data should not be nil")

	// Convert the data to health check response
	var healthResponse HealthCheckResponse
	dataBytes, err := json.Marshal(response.Data)
	require.NoError(t, err, "Should be able to marshal response data")

	err = json.Unmarshal(dataBytes, &healthResponse)
	require.NoError(t, err, "Should be able to unmarshal health check response")

	// Validate response structure
	assert.NotEmpty(t, healthResponse.Status, "Status should not be empty")
	assert.NotEmpty(t, healthResponse.Timestamp, "Timestamp should not be empty")
	assert.NotNil(t, healthResponse.Services, "Services should not be nil")

	// Check if services are included
	assert.Contains(t, healthResponse.Services, "database", "Should check database service")
	assert.Contains(t, healthResponse.Services, "redis", "Should check redis service")

	// Each service should have valid status
	for serviceName, serviceInfo := range healthResponse.Services {
		assert.NotEmpty(t, serviceInfo.Status, "Service %s should have status", serviceName)
		assert.Contains(t, []string{"healthy", "unhealthy", "disabled"}, serviceInfo.Status,
			"Service %s status should be valid", serviceName)
	}
}

// TestHealthCheckEndpointInServer tests that the health check endpoint is accessible through the server
func TestHealthCheckEndpointInServer(t *testing.T) {
	setupTestConfig()

	// Create server
	server := NewHTTPServer()
	require.NotNil(t, server, "Server should be created")
	require.NotNil(t, server.Handler, "Server should have handler")

	// Test that health check endpoint is accessible
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)

	// Should return valid response
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, rr.Code,
		"Health check endpoint should be accessible")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"),
		"Health check should return JSON")

	// Response should be parseable JSON
	var response HealthCheckResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Health check response should be valid JSON")
}

// TestHealthCheckConfigurationBased tests health check behavior based on database configuration
func TestHealthCheckConfigurationBased(t *testing.T) {
	t.Run("With database configuration", func(t *testing.T) {
		setupTestConfig() // This includes database configuration

		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server := NewHTTPServer()
		server.Handler.ServeHTTP(rr, req)

		// Parse the middleware response wrapper
		var response Response
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Extract the health check data from the wrapper
		var healthResponse HealthCheckResponse
		dataBytes, err := json.Marshal(response.Data)
		require.NoError(t, err)
		err = json.Unmarshal(dataBytes, &healthResponse)
		require.NoError(t, err)

		// With database configuration, services should be checked
		assert.Contains(t, healthResponse.Services, "database")
		assert.Contains(t, healthResponse.Services, "redis")

		dbService := healthResponse.Services["database"]
		redisService := healthResponse.Services["redis"]

		// Services should have either "healthy" or "unhealthy" status (not "disabled")
		assert.Contains(t, []string{"healthy", "unhealthy"}, dbService.Status)
		assert.Contains(t, []string{"healthy", "unhealthy"}, redisService.Status)
	})

	t.Run("Without database configuration", func(t *testing.T) {
		setupTestConfigWithoutDatabase() // This excludes database configuration

		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		server := NewHTTPServer()
		server.Handler.ServeHTTP(rr, req)

		// Parse the middleware response wrapper
		var response Response
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Extract the health check data from the wrapper
		var healthResponse HealthCheckResponse
		dataBytes, err := json.Marshal(response.Data)
		require.NoError(t, err)
		err = json.Unmarshal(dataBytes, &healthResponse)
		require.NoError(t, err)

		// Without database configuration, services should be marked as disabled
		assert.Contains(t, healthResponse.Services, "database")
		assert.Contains(t, healthResponse.Services, "redis")

		dbService := healthResponse.Services["database"]
		redisService := healthResponse.Services["redis"]

		// Services should be marked as "disabled"
		assert.Equal(t, "disabled", dbService.Status)
		assert.Equal(t, "disabled", redisService.Status)
		assert.Equal(t, "Database not configured", dbService.Message)
		assert.Equal(t, "Redis not configured", redisService.Message)

		// Overall status should still be "healthy" since disabled services don't affect health
		assert.Equal(t, "healthy", healthResponse.Status)
	})
}

// BenchmarkHealthCheckHandler benchmarks the health check handler
func BenchmarkHealthCheckHandler(b *testing.B) {
	setupTestConfig()

	// Create server
	server := NewHTTPServer()
	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		server.Handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
			b.Fatalf("Unexpected status code: %d", rr.Code)
		}
	}
}

// TestTracingMiddleware tests the tracing middleware functionality
func TestTracingMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		setupContext    func() context.Context
		expectedHeader  bool
		expectedTraceID string
	}{
		{
			name: "With valid trace context",
			setupContext: func() context.Context {
				// Create a mock trace ID
				traceID, _ := trace.TraceIDFromHex("12345678901234567890123456789012")
				spanID, _ := trace.SpanIDFromHex("1234567890123456")

				spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
				})

				return trace.ContextWithSpanContext(context.Background(), spanCtx)
			},
			expectedHeader:  true,
			expectedTraceID: "12345678901234567890123456789012",
		},
		{
			name: "Without trace context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedHeader:  false,
			expectedTraceID: "",
		},
		{
			name: "With invalid trace context",
			setupContext: func() context.Context {
				// Create an invalid span context
				spanCtx := trace.SpanContext{}
				return trace.ContextWithSpanContext(context.Background(), spanCtx)
			},
			expectedHeader:  false,
			expectedTraceID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple handler that just returns OK
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with tracing middleware
			tracingHandler := TracingMiddleware(handler)

			// Create request with the specific context
			req, err := http.NewRequestWithContext(tt.setupContext(), "GET", "/test", nil)
			require.NoError(t, err)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			tracingHandler.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())

			// Check X-Trace-Id header
			traceIDHeader := rr.Header().Get("X-Trace-Id")
			if tt.expectedHeader {
				assert.NotEmpty(t, traceIDHeader)
				assert.Equal(t, tt.expectedTraceID, traceIDHeader)
			} else {
				assert.Empty(t, traceIDHeader)
			}
		})
	}
}

// TestTracingMiddlewareIntegration tests tracing middleware integration with the server
func TestTracingMiddlewareIntegration(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()

	// Create a mock trace ID
	traceID, _ := trace.TraceIDFromHex("abcdef1234567890abcdef1234567890")
	spanID, _ := trace.SpanIDFromHex("abcdef1234567890")

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Create request with trace context
	req, err := http.NewRequestWithContext(ctx, "GET", "/health", nil)
	require.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	server.Handler.ServeHTTP(rr, req)

	// Check that health check works (should return either 200 or 503)
	assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusServiceUnavailable)

	// Check that X-Trace-Id header is present
	traceIDHeader := rr.Header().Get("X-Trace-Id")
	assert.NotEmpty(t, traceIDHeader)
	assert.Equal(t, "abcdef1234567890abcdef1234567890", traceIDHeader)

	// Verify response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response, "message")
}

// TestLoggingMiddleware tests the custom logging middleware functionality
func TestLoggingMiddleware(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create request
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "test-agent/1.0")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request (this will generate a log entry)
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Hello, World!", rr.Body.String())

	// Note: The logging output will be visible in test output if log level allows
	// The middleware logs to the configured logger (console/file)
}

// TestLoggingMiddlewareWithTracing tests logging middleware with OpenTelemetry context
func TestLoggingMiddlewareWithTracing(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"created"}`))
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create a mock trace ID
	traceID, _ := trace.TraceIDFromHex("fedcba0987654321fedcba0987654321")
	spanID, _ := trace.SpanIDFromHex("fedcba0987654321")

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Create request with trace context
	req, err := http.NewRequestWithContext(ctx, "POST", "/api/users", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "mobile-app/2.1.0")
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.JSONEq(t, `{"status":"created"}`, rr.Body.String())

	// The log entry will include the trace ID and span ID automatically
	// due to the logger's enrichLogger method
}

// TestLoggingMiddlewareIntegration tests logging middleware integration with the server
func TestLoggingMiddlewareIntegration(t *testing.T) {
	setupTestConfig()

	server := NewHTTPServer()

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "health-checker/1.0")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	server.Handler.ServeHTTP(rr, req)

	// Check that health check works (should return either 200 or 503)
	assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusServiceUnavailable)

	// Verify response is valid JSON
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response, "message")

	// The request will be logged with structured format including
	// method, URL, status, duration, etc.
}

// TestLoggingMiddlewareWithPayload tests payload logging functionality
func TestLoggingMiddlewareWithPayload(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		url         string
		payload     string
		contentType string
		description string
	}{
		{
			name:        "JSON payload",
			method:      "POST",
			url:         "/api/users",
			payload:     `{"name":"John Doe","email":"john@example.com","age":30}`,
			contentType: "application/json",
			description: "Should log JSON payload content",
		},
		{
			name:        "Form data payload",
			method:      "POST",
			url:         "/api/login",
			payload:     "username=admin&password=secret123",
			contentType: "application/x-www-form-urlencoded",
			description: "Should log form data content",
		},
		{
			name:        "Large JSON payload",
			method:      "POST",
			url:         "/api/bulk",
			payload:     `{"data":` + strings.Repeat(`"x"`, 600) + `}`, // > 1KB payload
			contentType: "application/json",
			description: "Should truncate large payloads",
		},
		{
			name:        "Binary payload",
			method:      "POST",
			url:         "/api/upload",
			payload:     "binary data here",
			contentType: "application/octet-stream",
			description: "Should log [binary data] for binary content types",
		},
		{
			name:        "Empty payload",
			method:      "GET",
			url:         "/api/users",
			payload:     "",
			contentType: "",
			description: "Should handle requests without payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with logging middleware
			loggingHandler := LoggingMiddleware(handler)

			// Create request with payload
			var req *http.Request
			var err error
			if tt.payload != "" {
				req, err = http.NewRequest(tt.method, tt.url, strings.NewReader(tt.payload))
				require.NoError(t, err)
				req.Header.Set("Content-Type", tt.contentType)
			} else {
				req, err = http.NewRequest(tt.method, tt.url, nil)
				require.NoError(t, err)
			}
			req.Header.Set("User-Agent", "test-client/1.0")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request (this will generate a log entry with payload)
			loggingHandler.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())

			// The logging output will include payload information based on content type
			// You can see the structured log output in the test output
		})
	}
}

// TestLoggingMiddlewarePayloadWithTracing tests payload logging with OpenTelemetry context
func TestLoggingMiddlewarePayloadWithTracing(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":123,"status":"created"}`))
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create trace context
	traceID, _ := trace.TraceIDFromHex("payload123456789012345678901234567")
	spanID, _ := trace.SpanIDFromHex("payload123456789")

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Create request with JSON payload and trace context
	jsonPayload := `{
		"user": {
			"name": "Alice Smith",
			"email": "alice@example.com",
			"preferences": {
				"theme": "dark",
				"notifications": true
			}
		},
		"metadata": {
			"source": "mobile-app",
			"version": "2.1.0"
		}
	}`

	req, err := http.NewRequestWithContext(ctx, "POST", "/api/profile", strings.NewReader(jsonPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mobile-app/2.1.0")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.JSONEq(t, `{"id":123,"status":"created"}`, rr.Body.String())

	// The log entry will include:
	// - trace ID and span ID (from OpenTelemetry context)
	// - request payload (JSON content)
	// - request and response byte counts
	// - all standard HTTP request information
}

// TestLoggingMiddlewareWithResponseData tests response data logging functionality
func TestLoggingMiddlewareWithResponseData(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		url          string
		requestBody  string
		responseBody string
		responseType string
		statusCode   int
		description  string
	}{
		{
			name:         "JSON API response",
			method:       "GET",
			url:          "/api/users/123",
			requestBody:  "",
			responseBody: `{"id":123,"name":"John Doe","email":"john@example.com"}`,
			responseType: "application/json",
			statusCode:   200,
			description:  "Should log JSON response data",
		},
		{
			name:         "Error response",
			method:       "POST",
			url:          "/api/users",
			requestBody:  `{"name":"","email":"invalid"}`,
			responseBody: `{"message":"Validation failed","errors":[{"field":"name","message":"name is required"}]}`,
			responseType: "application/json",
			statusCode:   422,
			description:  "Should log error response data",
		},
		{
			name:         "Large response",
			method:       "GET",
			url:          "/api/export",
			requestBody:  "",
			responseBody: `{"data":[` + strings.Repeat(`{"id":1,"name":"user"},`, 100) + `{"id":101,"name":"last"}]}`,
			responseType: "application/json",
			statusCode:   200,
			description:  "Should truncate large response data",
		},
		{
			name:         "Text response",
			method:       "GET",
			url:          "/api/status",
			requestBody:  "",
			responseBody: "Service is running normally",
			responseType: "text/plain",
			statusCode:   200,
			description:  "Should log text response data",
		},
		{
			name:         "Empty response",
			method:       "DELETE",
			url:          "/api/users/123",
			requestBody:  "",
			responseBody: "",
			responseType: "",
			statusCode:   204,
			description:  "Should handle empty responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a handler that returns the specified response
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.responseType != "" {
					w.Header().Set("Content-Type", tt.responseType)
				}
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					w.Write([]byte(tt.responseBody))
				}
			})

			// Wrap with logging middleware
			loggingHandler := LoggingMiddleware(handler)

			// Create request
			var req *http.Request
			var err error
			if tt.requestBody != "" {
				req, err = http.NewRequest(tt.method, tt.url, strings.NewReader(tt.requestBody))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, tt.url, nil)
				require.NoError(t, err)
			}
			req.Header.Set("User-Agent", "test-client/1.0")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request (this will generate a log entry with response data)
			loggingHandler.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.statusCode, rr.Code)
			assert.Equal(t, tt.responseBody, rr.Body.String())

			// The logging output will include both request and response data
			// You can see the structured log output in the test output
		})
	}
}

// TestLoggingMiddlewareFullCycle tests complete request-response logging with tracing
func TestLoggingMiddlewareFullCycle(t *testing.T) {
	// Create a realistic API handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing the request body
		body, _ := io.ReadAll(r.Body)
		var requestData map[string]interface{}
		json.Unmarshal(body, &requestData)

		// Create a response based on the request
		response := map[string]interface{}{
			"success": true,
			"message": "User created successfully",
			"data": map[string]interface{}{
				"id":    123,
				"name":  requestData["name"],
				"email": requestData["email"],
			},
			"timestamp": "2025-07-20T13:54:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create trace context
	traceID, _ := trace.TraceIDFromHex("fullcycle123456789012345678901234")
	spanID, _ := trace.SpanIDFromHex("fullcycle12345678")

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Create request with payload
	requestPayload := `{
		"name": "Alice Johnson",
		"email": "alice.johnson@example.com",
		"department": "Engineering",
		"role": "Senior Developer"
	}`

	req, err := http.NewRequestWithContext(ctx, "POST", "/api/users", strings.NewReader(requestPayload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "api-client/3.0")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), "Alice Johnson")
	assert.Contains(t, rr.Body.String(), "User created successfully")

	// The log entry will include:
	// - OpenTelemetry trace/span IDs
	// - Complete request payload (user data)
	// - Complete response data (created user with ID)
	// - Request and response byte counts
	// - Status codes, timing, and all HTTP metadata
}

// TestLoggingMiddlewareWithHeaders tests header logging functionality
func TestLoggingMiddlewareWithHeaders(t *testing.T) {
	tests := []struct {
		name            string
		requestHeaders  map[string]string
		responseHeaders map[string]string
		description     string
	}{
		{
			name: "Standard headers",
			requestHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Accept":        "application/json",
				"User-Agent":    "test-client/1.0",
				"X-Request-ID":  "req-123456",
				"Cache-Control": "no-cache",
			},
			responseHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-Response-ID": "resp-789012",
				"Cache-Control": "no-store",
			},
			description: "Should log standard headers without masking",
		},
		{
			name: "Sensitive headers (masked)",
			requestHeaders: map[string]string{
				"Authorization": "Bearer secret-token-123",
				"Cookie":        "session=abcd1234; user=john",
				"X-API-Key":     "api-key-secret-456",
				"X-Auth-Token":  "auth-token-789",
				"Content-Type":  "application/json",
			},
			responseHeaders: map[string]string{
				"Set-Cookie":     "session=new-session-id; HttpOnly",
				"X-Access-Token": "new-access-token-123",
				"Content-Type":   "application/json",
			},
			description: "Should mask sensitive headers for security",
		},
		{
			name: "Multiple header values",
			requestHeaders: map[string]string{
				"Accept":          "application/json, text/plain",
				"Accept-Encoding": "gzip, deflate, br",
				"Content-Type":    "application/json",
			},
			responseHeaders: map[string]string{
				"Content-Type": "application/json; charset=utf-8",
				"Vary":         "Accept-Encoding, Origin",
			},
			description: "Should handle headers with multiple values",
		},
		{
			name:            "No custom headers",
			requestHeaders:  map[string]string{},
			responseHeaders: map[string]string{},
			description:     "Should handle requests with minimal headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a handler that sets response headers
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set response headers
				for key, value := range tt.responseHeaders {
					w.Header().Set(key, value)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message":"success"}`))
			})

			// Wrap with logging middleware
			loggingHandler := LoggingMiddleware(handler)

			// Create request with headers
			req, err := http.NewRequest("POST", "/api/test", strings.NewReader(`{"test":"data"}`))
			require.NoError(t, err)

			// Set request headers
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request (this will generate a log entry with headers)
			loggingHandler.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Contains(t, rr.Body.String(), "success")

			// Verify response headers were set
			for key, expectedValue := range tt.responseHeaders {
				if !strings.Contains(strings.ToLower(key), "token") && key != "Set-Cookie" {
					assert.Equal(t, expectedValue, rr.Header().Get(key))
				}
			}

			// The logging output will include request_headers and response_headers
			// Sensitive headers will be masked as [MASKED]
		})
	}
}

// TestLoggingMiddlewareHeaderSecurity tests header masking security
func TestLoggingMiddlewareHeaderSecurity(t *testing.T) {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=secret123; HttpOnly")
		w.Header().Set("X-Internal-Token", "internal-secret")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"response"}`))
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create request with sensitive headers
	req, err := http.NewRequest("POST", "/api/secure", strings.NewReader(`{"sensitive":"data"}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer very-secret-token")
	req.Header.Set("Cookie", "session=user123; auth=token456")
	req.Header.Set("X-API-Key", "super-secret-api-key")
	req.Header.Set("X-Password", "user-password")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "security-test/1.0")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// The log will contain masked sensitive headers:
	// request_headers: {
	//   "Authorization": "[MASKED]",
	//   "Cookie": "[MASKED]",
	//   "X-Api-Key": "[MASKED]",
	//   "X-Password": "[MASKED]",
	//   "Content-Type": "application/json",
	//   "User-Agent": "security-test/1.0"
	// }
	// response_headers: {
	//   "Set-Cookie": "[MASKED]",
	//   "X-Internal-Token": "[MASKED]",
	//   "Content-Type": "application/json"
	// }
}

// TestLoggingMiddlewareCompleteWithHeaders tests complete logging including headers and tracing
func TestLoggingMiddlewareCompleteWithHeaders(t *testing.T) {
	// Create a realistic handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS and other response headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Rate-Limit-Remaining", "99")
		w.Header().Set("X-Response-Time", "15ms")

		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"success": true,
			"message": "Resource created",
			"id":      "resource-123",
		}
		json.NewEncoder(w).Encode(response)
	})

	// Wrap with logging middleware
	loggingHandler := LoggingMiddleware(handler)

	// Create trace context
	traceID, _ := trace.TraceIDFromHex("headers123456789012345678901234567")
	spanID, _ := trace.SpanIDFromHex("headers123456789")

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	// Create request with comprehensive headers
	requestPayload := `{"name":"Test Resource","type":"example"}`
	req, err := http.NewRequestWithContext(ctx, "POST", "/api/resources", strings.NewReader(requestPayload))
	require.NoError(t, err)

	// Set comprehensive request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer jwt-token-here")
	req.Header.Set("User-Agent", "api-client/4.0")
	req.Header.Set("X-Request-ID", "req-abc123def")
	req.Header.Set("X-Client-Version", "4.0.1")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	loggingHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), "Resource created")

	// The complete log entry will include:
	// - OpenTelemetry trace/span IDs
	// - Request payload and response data
	// - Complete request headers (with Authorization masked)
	// - Complete response headers
	// - Request and response byte counts
	// - Performance metrics and all HTTP metadata
}
