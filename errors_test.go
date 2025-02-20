package treblle

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestErrorSeverityMapping(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected ErrorSeverity
	}{
		{"UnhandledException", UnhandledExceptionError, ErrorSeverityCritical},
		{"SystemError", SystemError, ErrorSeverityCritical},
		{"DatabaseError", DatabaseError, ErrorSeverityCritical},
		{"RuntimeError", RuntimeError, ErrorSeverityHigh},
		{"FrameworkError", FrameworkError, ErrorSeverityHigh},
		{"RequestError", RequestError, ErrorSeverityMedium},
		{"ResponseError", ResponseError, ErrorSeverityMedium},
		{"ValidationError", ValidationError, ErrorSeverityLow},
		{"UserWarning", E_USER_WARNING, ErrorSeverityLow},
		{"UserError", E_USER_ERROR, ErrorSeverityMedium}, // Default case
	}

	ep := NewErrorProvider()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := ep.getSeverity(tt.errType)
			assert.Equal(t, tt.expected, severity, "Error type %s should have severity %s", tt.errType, tt.expected)
		})
	}
}
