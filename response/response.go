package response

import (
	"fmt"
)

type Response struct {
	Message string `json:"message" example:""`
	ID      uint   `json:"id,omitempty" example:"1"` // the effected ID (if exists)
}

type StatusResponse struct {
	Status string `json:"status" example:"OK"`
}

type ErrorResponse struct {
	Message string `json:"message" example:"Error details"`
	Error   string `json:"error,omitempty"`
}

func NewResponse(message string, id uint) Response {
	return Response{Message: message}
}

func ErrorResponseF(format string, a ...interface{}) ErrorResponse {
	return ErrorResponse{Message: fmt.Sprintf(format, a...)}
}

func NewErrorResponseF(err error, format string, a ...interface{}) ErrorResponse {
	message := fmt.Sprintf(format, a...)
	return ErrorResponse{Message: message, Error: err.Error()}
}

func NewErrorResponse(message string, err error) ErrorResponse {
	return ErrorResponse{Message: message, Error: err.Error()}
}
