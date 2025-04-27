package handle

import (
	"errors"
	"net/http"
	"precisiondosing-api-go/internal/utils/apierr"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, newJSendSuccess(data))
}

func SuccessWithStatus(c *gin.Context, status int, data interface{}) {
	c.JSON(status, newJSendSuccess(data))
}

func ServerError(c *gin.Context, err error) {
	Error(c, apierr.New(http.StatusInternalServerError, err.Error()))
}

func UnauthorizedError(c *gin.Context, msg string) {
	Error(c, apierr.New(http.StatusUnauthorized, msg))
}

func ForbiddenError(c *gin.Context, msg string) {
	Error(c, apierr.New(http.StatusForbidden, msg))
}

func BadRequestError(c *gin.Context, msg string) {
	Error(c, apierr.New(http.StatusBadRequest, msg))
}

func ValidationError(c *gin.Context, errors []apierr.ValidationError) {
	c.JSON(http.StatusUnprocessableEntity, newJSendValidationFailure(errors))
}

func NotFoundError(c *gin.Context, msg string) {
	Error(c, apierr.New(http.StatusNotFound, msg))
}

func Error(c *gin.Context, err error) {
	var apiErr apierr.Error
	if !errors.As(err, &apiErr) {
		apiErr = apierr.New(http.StatusInternalServerError, err.Error())
	}

	apiErr.Log(c)
	if apiErr.Status() >= http.StatusInternalServerError {
		c.JSON(apiErr.Status(), newJSendError(apiErr.Message(), apiErr.Status()))
		return
	}

	if apiErr.Status() >= http.StatusBadRequest {
		c.JSON(apiErr.Status(), newJSendFailure(apiErr.Message()))
		return
	}

	c.JSON(apiErr.Status(), errorResponse{Error: apiErr.Message()})
}

type jSendError struct {
	Status  string `json:"status" example:"error"`                  // Status
	Message string `json:"message" example:"Internal server error"` // Error message
	Code    int    `json:"code" example:"500"`                      // Error code
} //	@name	JSendError

type jsendFailure[T any] struct {
	Status string `json:"status" example:"fail"` // Status 'fail'
	Data   T      `json:"data"`                  // Data with error message(s)
} //	@name	JSendFailure

type jsendSuccess[T any] struct {
	Status string `json:"status" example:"success"` // Status 'success'
	Data   T      `json:"data"`                     // Data with success message(s)
} //	@name	JSendSuccess

func newJSendError(message string, code int) jSendError {
	return jSendError{
		Status:  "error", // Status 'error'
		Message: message, // Single error message
		Code:    code,    // Error code
	}
}

func newJSendFailure(message string) jsendFailure[errorResponse] {
	return jsendFailure[errorResponse]{
		Status: "fail",
		Data:   errorResponse{Error: message},
	}
}

func newJSendValidationFailure(validationErrors []apierr.ValidationError) jsendFailure[validationResponse] {
	return jsendFailure[validationResponse]{
		Status: "fail",
		Data:   validationResponse{Errors: validationErrors},
	}
}

func newJSendSuccess[T any](data T) jsendSuccess[T] {
	return jsendSuccess[T]{
		Status: "success",
		Data:   data,
	}
}

type errorResponse struct {
	Error string `json:"error" example:"Some error message"` // Error message
} //	@name	ErrorResponse

type validationResponse struct {
	Errors []apierr.ValidationError `json:"errors"` // Validation errors
} //	@name	ValidationResponse
