package treblle

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/timpratim/treblle-go/models"
	"github.com/timpratim/treblle-go/treblle"
)

func TestResponseSizeLimit(t *testing.T) {
	// Create a new error provider
	errorProvider := models.NewErrorProvider()

	// Create a response recorder
	w := httptest.NewRecorder()
	
	// Generate a response body that exceeds 2MB
	largeBody := strings.Repeat("a", treblle.MaxResponseSize+1)
	w.WriteString(largeBody)
	
	// Get the response info
	startTime := time.Now().Add(-100 * time.Millisecond) // Simulate some processing time
	responseInfo := treblle.GetResponseInfo(w, startTime, errorProvider)
	
	// Verify the response body was replaced with an empty JSON object
	assert.Equal(t, json.RawMessage("{}"), responseInfo.Body)
	
	// Verify the size was set to 0
	assert.Equal(t, 0, responseInfo.Size)
	
	// Verify an error was added
	errors := errorProvider.GetErrors()
	assert.Equal(t, 1, len(errors))
	assert.Contains(t, errors[0].Message, "Response body exceeds maximum size")
}

func TestResponseSizeLimitNotExceeded(t *testing.T) {
	// Create a new error provider
	errorProvider := models.NewErrorProvider()

	// Create a response recorder
	w := httptest.NewRecorder()
	
	// Generate a response body that does not exceed the limit
	smallBody := `{"status":"success","data":{"id":123}}`
	w.WriteString(smallBody)
	
	// Get the response info
	startTime := time.Now().Add(-100 * time.Millisecond) // Simulate some processing time
	responseInfo := treblle.GetResponseInfo(w, startTime, errorProvider)
	
	// Verify the response body was preserved
	var body map[string]interface{}
	err := json.Unmarshal(responseInfo.Body, &body)
	assert.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	
	// Verify the size was set correctly
	assert.Equal(t, len(smallBody), responseInfo.Size)
	
	// Verify no errors were added
	errors := errorProvider.GetErrors()
	assert.Equal(t, 0, len(errors))
}
