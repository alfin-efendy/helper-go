package restapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// ValidationError for error response format
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Response is a struct for standard response format
type Response struct {
	Message string        `json:"message"`
	Errors  interface{}   `json:"errors,omitempty"`
	Data    interface{}   `json:"data,omitempty"`
	Page    *PageResponse `json:"page,omitempty"`
}

func paginationRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var page PageRequest

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
				var errors []ValidationError

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

					errors = append(errors, ValidationError{
						Field:   field,
						Message: message,
					})
				}

				response := Response{
					Message: "Validation failed",
					Errors:  errors,
				}

				c.JSON(http.StatusUnprocessableEntity, response)
				c.Abort()
				return
			}

			// Handle general errors
			response := Response{
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
			response := Response{
				Message: "Success",
			}

			if data, exists := c.Get(dataStr); exists {
				response.Data = data
			}

			if page, exists := c.Get(pageStr); exists {
				if pageResponse, ok := page.(PageResponse); ok {
					response.Page = &pageResponse
				}
			}

			c.JSON(http.StatusOK, response)
		}
	}
}
