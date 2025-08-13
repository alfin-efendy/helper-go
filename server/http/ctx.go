package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type Ctx struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Errors         []error
	Data           interface{}
	Page           *PageResponse
	written        bool
}

type ctxKey struct{}

func NewCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKey{}, &Ctx{
			ResponseWriter: w,
			Request:        r,
			Errors:         make([]error, 0),
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetCtx(r *http.Request) *Ctx {
	return r.Context().Value(ctxKey{}).(*Ctx)
}

func (c *Ctx) Bind(v interface{}) error {
	contentType := c.Request.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		return c.BindJSON(v)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return c.BindForm(v)
	case strings.Contains(contentType, "multipart/form-data"):
		return c.BindForm(v)
	default:
		// Default to JSON for backward compatibility
		return c.BindJSON(v)
	}
}

// BindJSON binds JSON request body to struct
func (c *Ctx) BindJSON(v interface{}) error {
	// Validate request before parsing
	if err := c.validateRequest(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		return err
	}
	return validate.Struct(v)
}

// BindForm binds form data to struct
func (c *Ctx) BindForm(v interface{}) error {
	// Validate request before parsing to mitigate potential chunked encoding issues
	if err := c.validateRequest(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	if err := c.Request.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if err := c.bindFormData(c.Request.Form, v); err != nil {
		return err
	}

	return validate.Struct(v)
}

// BindQuery binds query parameters to struct
func (c *Ctx) BindQuery(v interface{}) error {
	queryValues := c.Request.URL.Query()

	if err := c.bindFormData(queryValues, v); err != nil {
		return err
	}

	return validate.Struct(v)
}

// validateRequest performs basic request validation to mitigate parsing vulnerabilities
func (c *Ctx) validateRequest() error {
	// Check for reasonable content length to prevent extremely large payloads
	if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
		return fmt.Errorf("request too large: %d bytes", c.Request.ContentLength)
	}

	// Validate Transfer-Encoding header to mitigate chunked encoding issues
	if te := c.Request.Header.Get("Transfer-Encoding"); te != "" {
		// Only allow standard chunked encoding, reject malformed values
		if te != "chunked" && te != "identity" {
			return fmt.Errorf("unsupported transfer encoding: %s", te)
		}
	}

	// Validate Content-Length and Transfer-Encoding are not both explicitly set
	// Note: Go sets ContentLength to -1 when Transfer-Encoding is present, so we check for explicit headers
	if c.Request.Header.Get("Content-Length") != "" && c.Request.Header.Get("Transfer-Encoding") != "" {
		return fmt.Errorf("both Content-Length and Transfer-Encoding headers present")
	}

	return nil
}

// bindFormData is a helper function to bind url.Values to struct
func (c *Ctx) bindFormData(values url.Values, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to struct")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the form tag or use field name
		tag := fieldType.Tag.Get("form")
		if tag == "" {
			tag = fieldType.Tag.Get("json")
		}
		if tag == "" {
			tag = strings.ToLower(fieldType.Name)
		}

		// Handle tag options (e.g., "name,omitempty")
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}

		// Get value from form data
		value := values.Get(tag)
		if value == "" {
			continue
		}

		// Set the field value based on its type
		if err := c.setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets the field value based on its type
func (c *Ctx) setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// AddError adds an error to the context
func (c *Ctx) AddError(err error) {
	if err != nil {
		c.Errors = append(c.Errors, err)
	}
}

// SetData sets the response data
func (c *Ctx) SetData(data interface{}) {
	c.Data = data
}

// SetPage sets the pagination response
func (c *Ctx) SetPage(page PageResponse) {
	c.Page = &page
}

// JSON sends a JSON response
func (c *Ctx) JSON(status int, obj interface{}) {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(status)
	err := json.NewEncoder(c.ResponseWriter).Encode(obj)
	if err != nil {
		http.Error(c.ResponseWriter, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
	c.written = true
}

// IsWritten checks if response has been written
func (c *Ctx) IsWritten() bool {
	return c.written
}
