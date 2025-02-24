package treblle

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	UnhandledExceptionError ErrorType = "UNHANDLED_EXCEPTION"
	RequestError           ErrorType = "REQUEST_ERROR"
	ResponseError         ErrorType = "RESPONSE_ERROR"
	RuntimeError          ErrorType = "RUNTIME_ERROR"
	SystemError          ErrorType = "SYSTEM_ERROR"
	FrameworkError       ErrorType = "FRAMEWORK_ERROR"
	ValidationError      ErrorType = "VALIDATION_ERROR"
	DatabaseError        ErrorType = "DATABASE_ERROR"
	E_USER_ERROR         ErrorType = "E_USER_ERROR"
	E_USER_WARNING       ErrorType = "E_USER_WARNING"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "critical"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityLow      ErrorSeverity = "low"
)

// ErrorInfo represents a detailed error with metadata
type ErrorInfo struct {
	Message    string        `json:"message"`
	File       string        `json:"file,omitempty"`
	Line       int           `json:"line,omitempty"`
	Type       ErrorType     `json:"type"`
	Source     string        `json:"source,omitempty"`
	Stack      string        `json:"stack,omitempty"`
	Timestamp  string        `json:"timestamp,omitempty"`
	Severity   ErrorSeverity `json:"severity,omitempty"`
	Context    ErrorContext  `json:"context,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ErrorProvider is responsible for collecting and managing errors
type ErrorProvider struct {
	mu     sync.Mutex
	errors []ErrorInfo
}

// NewErrorProvider creates a new ErrorProvider instance
func NewErrorProvider() *ErrorProvider {
	return &ErrorProvider{
		errors: make([]ErrorInfo, 0),
	}
}

// AddError adds an error with full stack trace and file information
func (ep *ErrorProvider) AddError(err error, errType ErrorType, source string) {
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

	errorInfo := ErrorInfo{
		Message:   err.Error(),
		File:      file,
		Line:      line,
		Type:      errType,
		Source:    source,
		Stack:     context.StackTrace,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Severity:  severity,
		Context:   context,
		Attributes: make(map[string]interface{}),
	}

	// Add additional context if error has it
	if ec, ok := err.(*ErrorWithContext); ok {
		errorInfo.Context = ec.Context
	}

	// Add to batch collector if enabled, otherwise add to regular errors slice
	if Config.batchErrorCollector != nil {
		Config.batchErrorCollector.Add(errorInfo)
	} else {
		ep.errors = append(ep.errors, errorInfo)
	}
}

// AddCustomError adds a custom error with provided details
func (ep *ErrorProvider) AddCustomError(message string, errType ErrorType, source string) {
	ep.AddError(errors.New(message), errType, source)
}

// AddErrorWithContext adds an error with additional context
func (ep *ErrorProvider) AddErrorWithContext(err error, errType ErrorType, source string, attrs map[string]interface{}) {
	if err == nil {
		return
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Get error context
	context := getErrorContext(1)

	// Get the caller's file and line number
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Determine severity
	severity := ep.getSeverity(errType)

	errorInfo := ErrorInfo{
		Message:    err.Error(),
		File:       file,
		Line:       line,
		Type:       errType,
		Source:     source,
		Stack:      context.StackTrace,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Severity:   severity,
		Context:    context,
		Attributes: attrs,
	}

	ep.errors = append(ep.errors, errorInfo)
}

// GetErrors returns all collected errors
func (ep *ErrorProvider) GetErrors() []ErrorInfo {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	// Return a copy to prevent external modifications
	result := make([]ErrorInfo, len(ep.errors))
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
func (ep *ErrorProvider) getSeverity(errType ErrorType) ErrorSeverity {
	switch errType {
	case UnhandledExceptionError, SystemError, DatabaseError:
		return ErrorSeverityCritical
	case RuntimeError, FrameworkError:
		return ErrorSeverityHigh
	case RequestError, ResponseError:
		return ErrorSeverityMedium
	case ValidationError, E_USER_WARNING:
		return ErrorSeverityLow
	default:
		return ErrorSeverityMedium
	}
}
