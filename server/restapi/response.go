package restapi

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Responses struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type CommonError struct {
	Errors map[string]interface{}
}

// APIResponse is the function to handle response
func APIResponse(ctx *gin.Context, Message string, StatusCode int, Data interface{}) {
	var jsonResponse Responses

	if StatusCode == 422 {
		jsonResponse = Responses{
			Message: Message,
			Error:   Data.(CommonError).Errors,
		}
	} else {

		jsonResponse = Responses{
			Message: Message,
			Data:    Data,
		}
	}

	if StatusCode >= 400 {
		ctx.JSON(StatusCode, jsonResponse)
		defer ctx.AbortWithStatus(StatusCode)
	} else {
		ctx.JSON(StatusCode, jsonResponse)
	}
}

// ValidatorError is the function to handle validator error
func ValidatorError(err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		errs := err.(validator.ValidationErrors)
		for _, v := range errs {
			// can translate each error one at a time.
			//fmt.Println("gg",v.NameNamespace)
			if v.Param() != "" {
				res.Errors[v.Field()] = fmt.Sprintf("{%v: %v}", v.Tag(), v.Param())
			} else {
				res.Errors[v.Field()] = fmt.Sprintf("%v", v.Tag())
			}

		}
		return res
	}

	res.Errors = map[string]interface{}{
		"Format": err.Error(),
	}
	return res
}
