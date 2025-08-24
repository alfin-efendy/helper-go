package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/logger"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
)

// LoggingMiddleware provides HTTP request logging integrated with the custom logger
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Capture request payload
		var payload string
		var payloadSize int64
		if r.Body != nil && r.ContentLength > 0 {
			// Read the body
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				payloadSize = int64(len(bodyBytes))
				// Convert to string, limit size for logging
				if len(bodyBytes) <= 1024 { // Log full payload if <= 1KB
					payload = string(bodyBytes)
				} else { // Truncate if larger
					payload = string(bodyBytes[:1024]) + "... (truncated)"
				}
				// Restore the body for the handler
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Create a response writer wrapper to capture status and size
		ww := &responseWriter{
			ResponseWriter: w,
			status:         200, // default status
			size:           0,
			body:           &bytes.Buffer{},
		}

		// Call the next handler
		next.ServeHTTP(ww, r)

		// Calculate duration
		duration := time.Since(start)

		// Capture response data
		var responseData string
		if ww.body != nil && ww.body.Len() > 0 {
			responseBytes := ww.body.Bytes()
			if len(responseBytes) <= 1024 { // Log full response if <= 1KB
				responseData = string(responseBytes)
			} else { // Truncate if larger
				responseData = string(responseBytes[:1024]) + "... (truncated)"
			}
		}

		// Build log fields
		logFields := []interface{}{
			"method", r.Method,
			"url", r.URL.String(),
			"proto", r.Proto,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"status", ww.status,
			"response_bytes", ww.size,
			"duration", duration.String(),
			"duration_ms", float64(duration.Nanoseconds()) / 1e6,
		}

		// Add request headers
		if len(r.Header) > 0 {
			requestHeaders := make(map[string]interface{})
			for key, values := range r.Header {
				// Mask sensitive headers for security
				if strings.ToLower(key) == "authorization" ||
					strings.ToLower(key) == "cookie" ||
					strings.ToLower(key) == "x-api-key" ||
					strings.Contains(strings.ToLower(key), "token") ||
					strings.Contains(strings.ToLower(key), "password") {
					requestHeaders[key] = "[MASKED]"
				} else if len(values) == 1 {
					requestHeaders[key] = values[0]
				} else {
					requestHeaders[key] = values
				}
			}
			logFields = append(logFields, "request_headers", requestHeaders)
		}

		// Add response headers
		responseHeaders := make(map[string]interface{})
		for key, values := range w.Header() {
			// Mask sensitive response headers
			if strings.ToLower(key) == "set-cookie" ||
				strings.Contains(strings.ToLower(key), "token") {
				responseHeaders[key] = "[MASKED]"
			} else if len(values) == 1 {
				responseHeaders[key] = values[0]
			} else {
				responseHeaders[key] = values
			}
		}
		if len(responseHeaders) > 0 {
			logFields = append(logFields, "response_headers", responseHeaders)
		}

		// Add payload information if present
		if payloadSize > 0 {
			logFields = append(logFields, "request_bytes", payloadSize)
			// Only log payload content for non-binary content types
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "application/xml") ||
				strings.Contains(contentType, "text/") ||
				strings.Contains(contentType, "application/x-www-form-urlencoded") {
				logFields = append(logFields, "payload", payload)
			} else {
				logFields = append(logFields, "payload", "[binary data]")
			}
		}

		// Add response data if present
		if responseData != "" {
			// Check response content type for logging decision
			responseContentType := w.Header().Get("Content-Type")
			if strings.Contains(responseContentType, "application/json") ||
				strings.Contains(responseContentType, "application/xml") ||
				strings.Contains(responseContentType, "text/") ||
				responseContentType == "" { // Default to logging if no content type specified
				logFields = append(logFields, "response_data", responseData)
			} else {
				logFields = append(logFields, "response_data", "[binary data]")
			}
		}

		// Log the request using our custom logger
		logger.Info(r.Context(), "HTTP Request", logFields...)
	})
}

// responseWriter wraps http.ResponseWriter to capture status and response size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int64
	body   *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Capture response body
	if rw.body != nil {
		rw.body.Write(b)
	}

	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// TracingMiddleware extracts OpenTelemetry trace ID and adds X-Trace-Id header to responses
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the span context from the request context
		spanCtx := trace.SpanContextFromContext(r.Context())

		// Extract trace ID if available
		var traceID string
		if spanCtx.IsValid() {
			traceID = spanCtx.TraceID().String()
		}

		// Add trace ID to response header if present
		if traceID != "" {
			w.Header().Set("X-Trace-Id", traceID)
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// ErrorResponseMiddleware handles error responses after the handler executes
func ErrorResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a custom response writer to capture the response
		ctx := GetCtx(r)

		// Call the next handler
		next.ServeHTTP(w, r)

		// Check if there are errors and response hasn't been written
		if len(ctx.Errors) > 0 && !ctx.IsWritten() {
			err := ctx.Errors[len(ctx.Errors)-1] // Get the last error

			// Handle validation errors
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				var errorList []ValidationError

				for _, e := range validationErrors {
					var message string
					field := strings.ToLower(string(e.Field()[0])) + e.Field()[1:]

					switch e.Tag() {
					case "required":
						message = field + " is required"
					case "email":
						message = field + " is not valid email"
					case "enum":
						validValues := strings.Split(e.Param(), "/")
						message = field + " must be one of: " + strings.Join(validValues, ", ")
					default:
						message = field + " is not valid"
					}

					errorList = append(errorList, ValidationError{
						Field:   field,
						Message: message,
					})
				}

				response := Response{
					Message: "Validation failed",
					Errors:  errorList,
				}

				ctx.JSON(http.StatusUnprocessableEntity, response)
				return
			}

			// Handle general errors
			response := Response{
				Message: "Internal Server Error",
			}

			// Check for specific error types
			switch err.Error() {
			case "record not found":
				response.Message = "Data not found"
				ctx.JSON(http.StatusNotFound, response)
				return
			case "EOF":
				response.Message = "Bad Request"
				ctx.JSON(http.StatusBadRequest, response)
				return
			}

			ctx.JSON(http.StatusInternalServerError, response)
		}
	})
}

// SuccessResponseMiddleware handles success responses after the handler executes
func SuccessResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)

		// Call the next handler
		next.ServeHTTP(w, r)

		// If there are no errors and response has not been sent
		if len(ctx.Errors) == 0 && !ctx.IsWritten() {
			response := Response{
				Message: "Success",
			}

			if ctx.Data != nil {
				response.Data = ctx.Data
			}

			if ctx.Page != nil {
				response.Page = ctx.Page
			}

			ctx.JSON(http.StatusOK, response)
		}
	})
}

// ResponseMiddleware combines both error and success response handling
func ResponseMiddleware(next http.Handler) http.Handler {
	return ErrorResponseMiddleware(SuccessResponseMiddleware(next))
}

// QueryParamsMiddleware parses query parameters and sets them in the context
func QueryParamsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := GetCtx(r)

		// Parse query parameters
		queryParams := &QueryParams{
			Filters: make(map[string]string),
		}

		values := r.URL.Query()

		// Parse pagination parameters
		if pageStr := values.Get("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil {
				queryParams.Page = page
			}
		}

		if limitStr := values.Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil {
				queryParams.Limit = limit
			}
		}

		// Parse search parameter
		queryParams.Search = values.Get("search")

		// Parse order parameters
		queryParams.Order = values.Get("order")
		queryParams.OrderBy = values.Get("order_by")

		// Parse filter parameters (any parameter that starts with "filter_")
		for key, vals := range values {
			if strings.HasPrefix(key, "filter_") && len(vals) > 0 {
				filterKey := strings.TrimPrefix(key, "filter_")
				queryParams.Filters[filterKey] = vals[0]
			}
		}

		// Set query parameters in context
		ctx.SetQuery(queryParams)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
