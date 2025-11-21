package utils

type ResponseData struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Results any    `json:"results,omitempty"`
}

// Success creates a success response
func Success(message string, results any) ResponseData {
	return ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
		Results: results,
	}
}

// Error creates an error response
func Error(status int, message string) ResponseData {
	return ResponseData{
		Status:  status,
		Code:    "ERROR",
		Message: message,
	}
}

// BadRequest creates a bad request response
func BadRequest(message string) ResponseData {
	return ResponseData{
		Status:  400,
		Code:    "BAD_REQUEST",
		Message: message,
	}
}

// NotFound creates a not found response
func NotFound(message string) ResponseData {
	return ResponseData{
		Status:  404,
		Code:    "NOT_FOUND",
		Message: message,
	}
}

// InternalServerError creates an internal server error response
func InternalServerError(message string) ResponseData {
	return ResponseData{
		Status:  500,
		Code:    "INTERNAL_SERVER_ERROR",
		Message: message,
	}
}
