package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Test request struct for validation
type TestRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"required,min=1,max=150"`
}

// setupTestRouter creates a test router with middleware
func setupTestRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(NewCtx)
	r.Use(ResponseMiddleware)
	return r
}

func TestSuccessResponse(t *testing.T) {
	r := setupTestRouter()

	// Handler that returns success data
	r.Get("/success", Handler(func(c *Ctx) {
		users := []map[string]interface{}{
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
		}

		c.SetData(users)
	}))

	req := httptest.NewRequest("GET", "/success", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Success", response.Message)
	assert.NotNil(t, response.Data)
	assert.Nil(t, response.Errors)
	assert.Nil(t, response.Page)
}

func TestSuccessResponseWithPagination(t *testing.T) {
	r := setupTestRouter()

	r.Get("/success-with-page", Handler(func(c *Ctx) {
		users := []map[string]interface{}{
			{"id": 1, "name": "John Doe"},
		}

		c.SetData(users)
		c.SetPage(PageResponse{
			Page:      1,
			Limit:     10,
			Total:     100,
			TotalPage: 10,
		})
	}))

	req := httptest.NewRequest("GET", "/success-with-page", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Success", response.Message)
	assert.NotNil(t, response.Data)
	assert.NotNil(t, response.Page)
	assert.Equal(t, 1, response.Page.Page)
	assert.Equal(t, 10, response.Page.Limit)
	assert.Equal(t, int64(100), response.Page.Total)
	assert.Equal(t, 10, response.Page.TotalPage)
}

func TestValidationErrors(t *testing.T) {
	r := setupTestRouter()

	r.Post("/validation", Handler(func(c *Ctx) {
		var req TestRequest
		if err := c.Bind(&req); err != nil {
			c.AddError(err)
			return
		}

		c.SetData(map[string]interface{}{
			"message": "User created successfully",
			"user":    req,
		})
	}))

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedErrors []string
	}{
		{
			name:           "Empty request body",
			requestBody:    `{}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedErrors: []string{"name is required", "email is required", "age is required"},
		},
		{
			name:           "Invalid email",
			requestBody:    `{"name":"John","email":"invalid-email","age":25}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedErrors: []string{"email is not valid email"},
		},
		{
			name:           "Age out of range",
			requestBody:    `{"name":"John","email":"john@example.com","age":200}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedErrors: []string{"age is not valid"},
		},
		{
			name:           "Valid request",
			requestBody:    `{"name":"John","email":"john@example.com","age":25}`,
			expectedStatus: http.StatusOK,
			expectedErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/validation", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)
			} else {
				assert.Equal(t, "Validation failed", response.Message)
				assert.Len(t, response.Errors, len(tt.expectedErrors))

				// Check if all expected error messages are present
				errorMessages := make([]string, len(response.Errors))
				for i, err := range response.Errors {
					errorMessages[i] = err.Message
				}

				for _, expectedError := range tt.expectedErrors {
					assert.Contains(t, errorMessages, expectedError)
				}
			}
		})
	}
}

func TestGeneralErrors(t *testing.T) {
	r := setupTestRouter()

	tests := []struct {
		name            string
		error           error
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "Record not found error",
			error:           errors.New("record not found"),
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "Data not found",
		},
		{
			name:            "EOF error",
			error:           errors.New("EOF"),
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Bad Request",
		},
		{
			name:            "Generic error",
			error:           errors.New("some generic error"),
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/error/" + strings.ReplaceAll(tt.name, " ", "_")
			r.Get(path, Handler(func(c *Ctx) {
				c.AddError(tt.error)
			}))

			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMessage, response.Message)
			assert.Nil(t, response.Data)
		})
	}
}

func TestInvalidJSON(t *testing.T) {
	r := setupTestRouter()

	r.Post("/invalid-json", Handler(func(c *Ctx) {
		var req TestRequest
		if err := c.Bind(&req); err != nil {
			c.AddError(err)
			return
		}

		c.SetData(req)
	}))

	req := httptest.NewRequest("POST", "/invalid-json", strings.NewReader(`{"invalid": json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal Server Error", response.Message)
}

func TestContextMethods(t *testing.T) {
	// Test Ctx methods directly
	ctx := &Ctx{
		Errors: make([]error, 0),
	}

	// Test AddError
	assert.Empty(t, ctx.Errors)
	ctx.AddError(errors.New("test error"))
	assert.Len(t, ctx.Errors, 1)

	// Test AddError with nil (should not add)
	ctx.AddError(nil)
	assert.Len(t, ctx.Errors, 1)

	// Test SetData
	testData := map[string]string{"key": "value"}
	ctx.SetData(testData)
	assert.Equal(t, testData, ctx.Data)

	// Test SetPage
	testPage := PageResponse{Page: 1, Limit: 10, Total: 100, TotalPage: 10}
	ctx.SetPage(testPage)
	assert.Equal(t, &testPage, ctx.Page)

	// Test IsWritten (initially false)
	assert.False(t, ctx.IsWritten())
}

func TestBind(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name:        "Valid JSON and validation",
			jsonData:    `{"name":"John","email":"john@example.com","age":25}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "Valid JSON but validation fails",
			jsonData:    `{"name":"","email":"invalid","age":0}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.jsonData))
			req.Header.Set("Content-Type", "application/json")

			ctx := &Ctx{
				Request: req,
			}

			var testReq TestRequest
			err := ctx.Bind(&testReq)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "John", testReq.Name)
				assert.Equal(t, "john@example.com", testReq.Email)
				assert.Equal(t, 25, testReq.Age)
			}
		})
	}
}

func TestResponseWrittenFlag(t *testing.T) {
	r := setupTestRouter()

	// Handler that manually writes response
	r.Get("/manual-response", Handler(func(c *Ctx) {
		c.JSON(http.StatusCreated, map[string]string{"message": "Manual response"})
		// This should prevent middleware from sending another response
	}))

	req := httptest.NewRequest("GET", "/manual-response", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Manual response", response["message"])
}

// Test struct for form binding with form tags
type FormRequest struct {
	Name  string `form:"name" validate:"required"`
	Email string `form:"email" validate:"required,email"`
	Age   int    `form:"age" validate:"required,min=1,max=150"`
	Admin bool   `form:"admin"`
}

// Test struct for query binding
type QueryRequest struct {
	Page   int    `form:"page" validate:"min=1"`
	Limit  int    `form:"limit" validate:"min=1,max=100"`
	Search string `form:"search"`
	Active bool   `form:"active"`
}

func TestBindJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		contentType string
		expectError bool
	}{
		{
			name:        "Valid JSON with explicit content type",
			jsonData:    `{"name":"John","email":"john@example.com","age":25}`,
			contentType: "application/json",
			expectError: false,
		},
		{
			name:        "Valid JSON without content type (defaults to JSON)",
			jsonData:    `{"name":"Jane","email":"jane@example.com","age":30}`,
			contentType: "",
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `{"invalid": json}`,
			contentType: "application/json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.jsonData))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			ctx := &Ctx{
				Request: req,
			}

			var testReq TestRequest
			err := ctx.BindJSON(&testReq)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBindForm(t *testing.T) {
	r := setupTestRouter()

	r.Post("/form", Handler(func(c *Ctx) {
		var req FormRequest
		if err := c.BindForm(&req); err != nil {
			c.AddError(err)
			return
		}

		c.SetData(req)
	}))

	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		expectError    bool
		expectedData   *FormRequest
	}{
		{
			name: "Valid form data",
			formData: url.Values{
				"name":  []string{"John Doe"},
				"email": []string{"john@example.com"},
				"age":   []string{"25"},
				"admin": []string{"true"},
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
			expectedData: &FormRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   25,
				Admin: true,
			},
		},
		{
			name: "Missing required fields",
			formData: url.Values{
				"age": []string{"25"},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name: "Invalid email format",
			formData: url.Values{
				"name":  []string{"John"},
				"email": []string{"invalid-email"},
				"age":   []string{"25"},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name: "Invalid age (out of range)",
			formData: url.Values{
				"name":  []string{"John"},
				"email": []string{"john@example.com"},
				"age":   []string{"200"},
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/form", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if !tt.expectError {
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)

				// Convert response.Data to FormRequest for comparison
				dataJSON, err := json.Marshal(response.Data)
				assert.NoError(t, err)

				var actualData FormRequest
				err = json.Unmarshal(dataJSON, &actualData)
				assert.NoError(t, err)

				assert.Equal(t, tt.expectedData.Name, actualData.Name)
				assert.Equal(t, tt.expectedData.Email, actualData.Email)
				assert.Equal(t, tt.expectedData.Age, actualData.Age)
				assert.Equal(t, tt.expectedData.Admin, actualData.Admin)
			} else {
				assert.Equal(t, "Validation failed", response.Message)
				assert.NotEmpty(t, response.Errors)
			}
		})
	}
}

func TestBindQuery(t *testing.T) {
	r := setupTestRouter()

	r.Get("/query", Handler(func(c *Ctx) {
		var req QueryRequest
		if err := c.BindQuery(&req); err != nil {
			c.AddError(err)
			return
		}

		c.SetData(req)
	}))

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectError    bool
		expectedData   *QueryRequest
	}{
		{
			name:           "Valid query parameters",
			queryParams:    "page=1&limit=10&search=test&active=true",
			expectedStatus: http.StatusOK,
			expectError:    false,
			expectedData: &QueryRequest{
				Page:   1,
				Limit:  10,
				Search: "test",
				Active: true,
			},
		},
		{
			name:           "Partial query parameters with valid values",
			queryParams:    "page=2&limit=5&search=partial",
			expectedStatus: http.StatusOK,
			expectError:    false,
			expectedData: &QueryRequest{
				Page:   2,
				Limit:  5,
				Search: "partial",
				Active: false, // Default value
			},
		},
		{
			name:           "Partial query parameters (missing limit fails validation)",
			queryParams:    "page=2&search=partial",
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name:           "Invalid page (less than 1)",
			queryParams:    "page=0&limit=10",
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name:           "Invalid limit (greater than 100)",
			queryParams:    "page=1&limit=200",
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/query?"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if !tt.expectError {
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)

				// Convert response.Data to QueryRequest for comparison
				dataJSON, err := json.Marshal(response.Data)
				assert.NoError(t, err)

				var actualData QueryRequest
				err = json.Unmarshal(dataJSON, &actualData)
				assert.NoError(t, err)

				assert.Equal(t, tt.expectedData.Page, actualData.Page)
				assert.Equal(t, tt.expectedData.Limit, actualData.Limit)
				assert.Equal(t, tt.expectedData.Search, actualData.Search)
				assert.Equal(t, tt.expectedData.Active, actualData.Active)
			} else {
				assert.Equal(t, "Validation failed", response.Message)
				assert.NotEmpty(t, response.Errors)
			}
		})
	}
}

func TestBindContentTypeDetection(t *testing.T) {
	r := setupTestRouter()

	r.Post("/auto-bind", Handler(func(c *Ctx) {
		var req FormRequest
		if err := c.Bind(&req); err != nil {
			c.AddError(err)
			return
		}

		c.SetData(req)
	}))

	tests := []struct {
		name        string
		contentType string
		requestBody string
		expectError bool
	}{
		{
			name:        "JSON content type",
			contentType: "application/json",
			requestBody: `{"name":"John","email":"john@example.com","age":25,"admin":true}`,
			expectError: false,
		},
		{
			name:        "Form content type",
			contentType: "application/x-www-form-urlencoded",
			requestBody: "name=John&email=john@example.com&age=25&admin=true",
			expectError: false,
		},
		{
			name:        "Unknown content type (defaults to JSON)",
			contentType: "text/plain",
			requestBody: `{"name":"John","email":"john@example.com","age":25,"admin":true}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auto-bind", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectError {
				assert.NotEqual(t, http.StatusOK, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)

				var response Response
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Success", response.Message)
				assert.NotNil(t, response.Data)
			}
		})
	}
}

func TestDirectBindMethods(t *testing.T) {
	t.Run("BindJSON direct test", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"Test","email":"test@example.com","age":25}`))
		ctx := &Ctx{Request: req}

		var testReq TestRequest
		err := ctx.BindJSON(&testReq)
		assert.NoError(t, err)
		assert.Equal(t, "Test", testReq.Name)
		assert.Equal(t, "test@example.com", testReq.Email)
		assert.Equal(t, 25, testReq.Age)
	})

	t.Run("BindForm direct test", func(t *testing.T) {
		formData := url.Values{
			"name":  []string{"Test User"},
			"email": []string{"test@example.com"},
			"age":   []string{"30"},
			"admin": []string{"false"},
		}
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ctx := &Ctx{Request: req}

		var formReq FormRequest
		err := ctx.BindForm(&formReq)
		assert.NoError(t, err)
		assert.Equal(t, "Test User", formReq.Name)
		assert.Equal(t, "test@example.com", formReq.Email)
		assert.Equal(t, 30, formReq.Age)
		assert.False(t, formReq.Admin)
	})

	t.Run("BindQuery direct test", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?page=5&limit=20&search=query&active=true", nil)
		ctx := &Ctx{Request: req}

		var queryReq QueryRequest
		err := ctx.BindQuery(&queryReq)
		assert.NoError(t, err)
		assert.Equal(t, 5, queryReq.Page)
		assert.Equal(t, 20, queryReq.Limit)
		assert.Equal(t, "query", queryReq.Search)
		assert.True(t, queryReq.Active)
	})
}
