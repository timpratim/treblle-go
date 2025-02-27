package treblle

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/timpratim/treblle-go/models"
)

func TestErrorSeverityMapping(t *testing.T) {
	tests := []struct {
		name     string
		errType  models.ErrorType
		expected models.ErrorSeverity
	}{
		{"UnhandledException", models.UnhandledExceptionError, models.ErrorSeverityCritical},
		{"SystemError", models.SystemError, models.ErrorSeverityCritical},
		{"DatabaseError", models.DatabaseError, models.ErrorSeverityCritical},
		{"RuntimeError", models.RuntimeError, models.ErrorSeverityHigh},
		{"FrameworkError", models.FrameworkError, models.ErrorSeverityHigh},
		{"RequestError", models.RequestError, models.ErrorSeverityMedium},
		{"ResponseError", models.ResponseError, models.ErrorSeverityMedium},
		{"ValidationError", models.ValidationError, models.ErrorSeverityLow},
		{"UserWarning", models.E_USER_WARNING, models.ErrorSeverityLow},
		{"UserError", models.E_USER_ERROR, models.ErrorSeverityMedium}, // Default case
	}

	ep := models.NewErrorProvider()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := ep.GetSeverity(tt.errType)
			assert.Equal(t, tt.expected, severity, "Error type %s should have severity %s", tt.errType, tt.expected)
		})
	}
}
