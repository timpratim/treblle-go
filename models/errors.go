package models

import (
	"fmt"
	"runtime"
	"sync"
)

// ErrorType represents the type of error
type ErrorType string

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

// Predefined error types
const (
	RequestError            ErrorType = "REQUEST"
	ResponseError           ErrorType = "RESPONSE"
	ServerError             ErrorType = "SERVER"
	ClientError             ErrorType = "CLIENT"
	UnhandledExceptionError ErrorType = "UNHANDLED_EXCEPTION"
	SystemError             ErrorType = "SYSTEM"
	DatabaseError           ErrorType = "DATABASE"
	RuntimeError            ErrorType = "RUNTIME"
	FrameworkError          ErrorType = "FRAMEWORK"
	ValidationError         ErrorType = "VALIDATION"
	E_USER_WARNING          ErrorType = "USER_WARNING"
	E_USER_ERROR            ErrorType = "USER_ERROR"
)

// Predefined error severity levels
const (
	ErrorSeverityCritical ErrorSeverity = "CRITICAL"
	ErrorSeverityHigh     ErrorSeverity = "HIGH"
	ErrorSeverityMedium   ErrorSeverity = "MEDIUM"
	ErrorSeverityLow      ErrorSeverity = "LOW"
)

// ErrorProvider is responsible for collecting and managing errors
type ErrorProvider struct {
	mu     sync.Mutex
	errors []ErrorInfo
}

// AddError adds an error with full stack trace and file information
func (ep *ErrorProvider) AddError(err error, errType ErrorType, source string) {
	if err == nil {
		return
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Get file and line information
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Create error info
	errorInfo := ErrorInfo{
		Message: err.Error(),
		Type:    string(errType),
		Source:  source,
		File:    file,
		Line:    line,
	}

	ep.errors = append(ep.errors, errorInfo)
}

// GetErrors returns a copy of all collected errors
func (ep *ErrorProvider) GetErrors() []ErrorInfo {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Create a copy to avoid race conditions
	result := make([]ErrorInfo, len(ep.errors))
	copy(result, ep.errors)
	return result
}

// Error returns a string representation of all errors
func (ep *ErrorProvider) Error() string {
	errors := ep.GetErrors()
	if len(errors) == 0 {
		return "no errors"
	}

	result := fmt.Sprintf("%d errors:\n", len(errors))
	for i, err := range errors {
		result += fmt.Sprintf("[%d] %s: %s\n", i+1, err.Type, err.Message)
	}
	return result
}

// GetSeverity returns the severity level for a given error type
func (ep *ErrorProvider) GetSeverity(errType ErrorType) ErrorSeverity {
	switch errType {
	case UnhandledExceptionError, SystemError, DatabaseError:
		return ErrorSeverityCritical
	case RuntimeError, FrameworkError:
		return ErrorSeverityHigh
	case RequestError, ResponseError, E_USER_ERROR:
		return ErrorSeverityMedium
	case ValidationError, E_USER_WARNING:
		return ErrorSeverityLow
	default:
		return ErrorSeverityMedium
	}
}

// Clear removes all errors from the error provider
func (ep *ErrorProvider) Clear() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.errors = nil
}

// AddCustomError adds a custom error with the specified message, type, and source
func (ep *ErrorProvider) AddCustomError(message string, errType ErrorType, source string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Get file and line information
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Create error info
	errorInfo := ErrorInfo{
		Message: message,
		Type:    string(errType),
		Source:  source,
		File:    file,
		Line:    line,
	}

	ep.errors = append(ep.errors, errorInfo)
}
