package restapi

import (
	"net/http"
	"strings"

	"github.com/alfin-efendy/helper-go/otel"
	"github.com/alfin-efendy/helper-go/server"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var (
	dataStr string = "data"
	pageStr string = "page"
)

func traceRequest() gin.HandlerFunc {
	tracer := otel.GetTracer()

	return func(c *gin.Context) {
		ctx, span := tracer.Start(c.Request.Context(), c.Request.URL.Path)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
		c.Next()
	}
}

func paginationRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var page server.PageRequest

		if err := c.ShouldBindQuery(&page); err != nil {
			c.Error(err)
			return
		}

		if page.Page < 1 {
			page.Page = 1
		}

		if page.PageSize < 1 {
			page.PageSize = 10
		}

		if page.PageSize > 100 {
			page.PageSize = 100
		}

		c.Set(pageStr, page)
		c.Next()
	}
}

// ErrorResponse is a middleware to handle error responses
func errorResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Continue request to handler

		// Check if there are errors from the handler
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Handle validation errors
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				var errors []server.ValidationError

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

					errors = append(errors, server.ValidationError{
						Field:   field,
						Message: message,
					})
				}

				response := server.Response{
					Message: "Validation failed",
					Errors:  errors,
				}

				c.JSON(http.StatusUnprocessableEntity, response)
				c.Abort()
				return
			}

			// Handle general errors
			response := server.Response{
				Message: "Internal Server Error",
			}

			if err == gorm.ErrRecordNotFound {
				response.Message = "Data not found"
				c.JSON(http.StatusNotFound, response)
				c.Abort()
				return
			}

			if err.Error() == "EOF" {
				response.Message = "Bad Request"
				c.JSON(http.StatusBadRequest, response)
				c.Abort()
				return
			}

			c.JSON(http.StatusInternalServerError, response)
			c.Abort()
			return
		}
	}
}

// SuccessResponse is a middleware to handle success responses
func successResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// If there are no errors and response has not been sent
		if len(c.Errors) == 0 && !c.Writer.Written() {
			response := server.Response{
				Message: "Success",
			}

			if data, exists := c.Get(dataStr); exists {
				response.Data = data
			}

			if page, exists := c.Get(pageStr); exists {
				if pageResponse, ok := page.(server.PageResponse); ok {
					response.Page = &pageResponse
				}
			}

			c.JSON(http.StatusOK, response)
		}
	}
}
