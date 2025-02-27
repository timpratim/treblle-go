package treblle

import (
	"errors"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

// ErrorType represents different categories of errors
type ErrorType models.ErrorType

const (
	UnhandledExceptionError ErrorType = ErrorType(models.UnhandledExceptionError)
	RequestError            ErrorType = ErrorType(models.RequestError)
	ResponseError           ErrorType = ErrorType(models.ResponseError)
	RuntimeError            ErrorType = ErrorType(models.RuntimeError)
	SystemError             ErrorType = ErrorType(models.SystemError)
	FrameworkError          ErrorType = ErrorType(models.FrameworkError)
	ValidationError         ErrorType = ErrorType(models.ValidationError)
	DatabaseError           ErrorType = ErrorType(models.DatabaseError)
	E_USER_ERROR            ErrorType = ErrorType(models.E_USER_ERROR)
	E_USER_WARNING          ErrorType = ErrorType(models.E_USER_WARNING)
)

// ErrorSeverity represents the severity level of errors
type ErrorSeverity models.ErrorSeverity

const (
	ErrorSeverityCritical ErrorSeverity = ErrorSeverity(models.ErrorSeverityCritical)
	ErrorSeverityHigh     ErrorSeverity = ErrorSeverity(models.ErrorSeverityHigh)
	ErrorSeverityMedium   ErrorSeverity = ErrorSeverity(models.ErrorSeverityMedium)
	ErrorSeverityLow      ErrorSeverity = ErrorSeverity(models.ErrorSeverityLow)
)

// ErrorProvider collects and manages errors
type ErrorProvider struct {
	mu     sync.Mutex
	errors []models.ErrorInfo
}

// NewErrorProvider creates a new error provider
func NewErrorProvider() *ErrorProvider {
	return &ErrorProvider{
		errors: make([]models.ErrorInfo, 0),
	}
}

func getErrorContext(skip int) string {
	// Get error context with stack trace
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	// Get the caller's file and line number
	function := runtime.FuncForPC(pc).Name()
	return strings.Join([]string{file, function, strconv.Itoa(line)}, ":")
}

// AddError adds an error with full stack trace and file information
func (ep *ErrorProvider) AddError(err error, errType models.ErrorType, source string) {
	if err == nil {
		return
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Get error context with stack trace
	context := getErrorContext(1)

	// Get the caller's file and line number
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Determine severity based on error type
	severity := ep.getSeverity(errType)

	// Create error info
	errorInfo := models.ErrorInfo{
		Message:  err.Error(),
		File:     file,
		Line:     line,
		Type:     string(errType),
		Source:   source,
		Severity: string(severity),
		Context:  context,
	}

	// Add additional context if error has it
	if _, ok := err.(*internal.ErrorWithContext); ok {
		// Add context to errorInfo
	}

	// Add to errors slice
	ep.errors = append(ep.errors, errorInfo)
}

// AddCustomError adds a custom error with provided details
func (ep *ErrorProvider) AddCustomError(message string, errType models.ErrorType, source string) {
	ep.AddError(errors.New(message), errType, source)
}

// AddErrorWithContext adds an error with additional context
func (ep *ErrorProvider) AddErrorWithContext(err error, errType models.ErrorType, source string) {
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

	// Get error context
	context := getErrorContext(2)

	// Determine severity
	severity := ep.getSeverity(errType)

	// Create error info with context
	errorInfo := models.ErrorInfo{
		Message:  err.Error(),
		File:     file,
		Line:     line,
		Type:     string(errType),
		Source:   source,
		Severity: string(severity),
		Context:  context,
	}

	ep.errors = append(ep.errors, errorInfo)
}

// GetErrors returns all collected errors
func (ep *ErrorProvider) GetErrors() []models.ErrorInfo {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Return a copy to prevent external modifications
	result := make([]models.ErrorInfo, len(ep.errors))
	copy(result, ep.errors)
	return result
}

// Clear removes all collected errors
func (ep *ErrorProvider) Clear() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.errors = ep.errors[:0]
}

// getSeverity determines error severity based on type
func (ep *ErrorProvider) getSeverity(errType models.ErrorType) models.ErrorSeverity {
	switch errType {
	case models.UnhandledExceptionError, models.SystemError, models.DatabaseError:
		return models.ErrorSeverityCritical
	case models.RuntimeError, models.FrameworkError:
		return models.ErrorSeverityHigh
	case models.RequestError, models.ResponseError:
		return models.ErrorSeverityMedium
	case models.ValidationError, models.E_USER_WARNING:
		return models.ErrorSeverityLow
	default:
		return models.ErrorSeverityMedium
	}
}
