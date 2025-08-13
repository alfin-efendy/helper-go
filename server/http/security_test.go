package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequestValidation tests the new request validation security measures
func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name                   string
		contentLength          int64
		transferEncoding       string
		contentLengthHeader    string
		body                   string
		contentType            string
		expectError            bool
		expectedErrorSubstring string
	}{
		{
			name:        "Valid JSON request",
			contentType: "application/json",
			body:        `{"name":"test","email":"test@example.com"}`,
			expectError: false,
		},
		{
			name:        "Valid form request",
			contentType: "application/x-www-form-urlencoded",
			body:        "name=test&email=test@example.com",
			expectError: false,
		},
		{
			name:                   "Request too large",
			contentType:            "application/json",
			body:                   strings.Repeat("x", 11*1024*1024), // 11MB
			expectError:            true,
			expectedErrorSubstring: "request too large",
		},
		{
			name:                   "Invalid transfer encoding",
			contentType:            "application/json",
			body:                   `{"test":"data"}`,
			transferEncoding:       "malformed-chunked",
			expectError:            true,
			expectedErrorSubstring: "unsupported transfer encoding",
		},
		{
			name:                   "Both Content-Length and Transfer-Encoding",
			contentType:            "application/json",
			body:                   `{"test":"data"}`,
			transferEncoding:       "chunked",
			contentLengthHeader:    "15",
			expectError:            true,
			expectedErrorSubstring: "both Content-Length and Transfer-Encoding headers present",
		},
		{
			name:             "Valid chunked encoding",
			contentType:      "application/json",
			body:             `{"test":"data"}`,
			transferEncoding: "chunked",
			expectError:      false,
		},
		{
			name:             "Valid identity encoding",
			contentType:      "application/json",
			body:             `{"test":"data"}`,
			transferEncoding: "identity",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			if tt.transferEncoding != "" {
				req.Header.Set("Transfer-Encoding", tt.transferEncoding)
			}

			if tt.contentLengthHeader != "" {
				req.Header.Set("Content-Length", tt.contentLengthHeader)
			}

			// Create context
			ctx := &Ctx{
				Request: req,
			}

			// Test structure for binding
			type TestStruct struct {
				Name  string `json:"name" form:"name" validate:"required"`
				Email string `json:"email" form:"email" validate:"required,email"`
			}
			var data TestStruct

			// Test the binding methods that include validation
			var err error
			if strings.Contains(tt.contentType, "application/json") {
				err = ctx.BindJSON(&data)
			} else {
				err = ctx.BindForm(&data)
			}

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorSubstring)
			} else {
				// Note: The request may still fail due to validation or JSON parsing
				// but it should NOT fail due to our security validation
				if err != nil {
					// Make sure it's not our security validation error
					assert.NotContains(t, err.Error(), "invalid request")
					assert.NotContains(t, err.Error(), "request too large")
					assert.NotContains(t, err.Error(), "unsupported transfer encoding")
					assert.NotContains(t, err.Error(), "both Content-Length and Transfer-Encoding")
				}
			}
		})
	}
}

// TestValidateRequestMethod tests the validateRequest method directly
func TestValidateRequestMethod(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectError    bool
		errorSubstring string
	}{
		{
			name: "Valid request",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
				return req
			},
			expectError: false,
		},
		{
			name: "Request too large",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
				req.ContentLength = 11 * 1024 * 1024 // 11MB
				return req
			},
			expectError:    true,
			errorSubstring: "request too large",
		},
		{
			name: "Invalid transfer encoding",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
				req.Header.Set("Transfer-Encoding", "invalid-encoding")
				return req
			},
			expectError:    true,
			errorSubstring: "unsupported transfer encoding",
		},
		{
			name: "Both headers present",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
				req.Header.Set("Content-Length", "4")
				req.Header.Set("Transfer-Encoding", "chunked")
				return req
			},
			expectError:    true,
			errorSubstring: "both Content-Length and Transfer-Encoding headers present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			ctx := &Ctx{Request: req}

			err := ctx.validateRequest()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorSubstring)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
