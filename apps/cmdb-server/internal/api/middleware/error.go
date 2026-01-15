package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess = 0

	CodeInvalidRequest   = 1001
	CodeValidationFailed = 1002

	CodeUnauthorized = 2001
	CodeInvalidToken = 2002
	CodeTokenExpired = 2003

	CodeForbidden = 3001

	CodeNotFound = 4001

	CodeInternalError    = 5001
	CodeDatabaseError    = 5002
	CodeExternalAPIError = 5003
)

var codeMessages = map[int]string{
	CodeSuccess:          "success",
	CodeInvalidRequest:   "invalid request",
	CodeValidationFailed: "validation failed",
	CodeUnauthorized:     "unauthorized",
	CodeInvalidToken:     "invalid token",
	CodeTokenExpired:     "token expired",
	CodeForbidden:        "forbidden",
	CodeNotFound:         "not found",
	CodeInternalError:    "internal server error",
	CodeDatabaseError:    "database error",
	CodeExternalAPIError: "external API error",
}

type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func NewErrorResponse(code int, message string, details interface{}) ErrorResponse {
	if message == "" {
		if msg, ok := codeMessages[code]; ok {
			message = msg
		} else {
			message = "unknown error"
		}
	}
	return ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewSuccessResponse(data interface{}) gin.H {
	return gin.H{
		"code":    CodeSuccess,
		"message": "success",
		"data":    data,
	}
}

type AppError struct {
	Code       int
	Message    string
	Details    interface{}
	HTTPStatus int
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, message string, httpStatus int) *AppError {
	if message == "" {
		if msg, ok := codeMessages[code]; ok {
			message = msg
		}
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

var (
	ErrInvalidRequest   = NewAppError(CodeInvalidRequest, "", http.StatusBadRequest)
	ErrValidationFailed = NewAppError(CodeValidationFailed, "", http.StatusBadRequest)
	ErrUnauthorized     = NewAppError(CodeUnauthorized, "", http.StatusUnauthorized)
	ErrInvalidToken     = NewAppError(CodeInvalidToken, "", http.StatusUnauthorized)
	ErrTokenExpired     = NewAppError(CodeTokenExpired, "", http.StatusUnauthorized)
	ErrForbidden        = NewAppError(CodeForbidden, "", http.StatusForbidden)
	ErrNotFound         = NewAppError(CodeNotFound, "", http.StatusNotFound)
	ErrInternalError    = NewAppError(CodeInternalError, "", http.StatusInternalServerError)
	ErrDatabaseError    = NewAppError(CodeDatabaseError, "", http.StatusInternalServerError)
	ErrExternalAPIError = NewAppError(CodeExternalAPIError, "", http.StatusBadGateway)
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		if appErr, ok := err.(*AppError); ok {
			c.JSON(appErr.HTTPStatus, ErrorResponse{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: err.Error(),
		})
	}
}

func AbortWithError(c *gin.Context, httpStatus int, code int, message string) {
	c.AbortWithStatusJSON(httpStatus, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func AbortWithAppError(c *gin.Context, err *AppError) {
	c.AbortWithStatusJSON(err.HTTPStatus, ErrorResponse{
		Code:    err.Code,
		Message: err.Message,
		Details: err.Details,
	})
}
