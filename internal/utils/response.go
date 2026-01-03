package utils

// Response represents a standardized response structure.
// It includes a status code, a message, and data.
type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` // Ensure data is always present, even if nil (will be null in JSON)
}

// NewResponse creates a new Response instance.
func NewResponse(status int, message string, data interface{}) Response {
	return Response{
		Status:  status,
		Message: message,
		Data:    data,
	}
}

// NewSuccessResponse creates a new success Response instance.
// Defaults status to 200 (OK).
func NewSuccessResponse(message string, data interface{}) Response {
	return Response{
		Status:  200,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates a new error Response instance.
// Data is explicitly set to nil.
func NewErrorResponse(status int, message string) Response {
	return Response{
		Status:  status,
		Message: message,
		Data:    nil,
	}
}
