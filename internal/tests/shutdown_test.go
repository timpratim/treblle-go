package treblle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

func TestShutdown(t *testing.T) {
	// Configure Treblle for testing
	internal.Configure(internal.Configuration{
		ApiKey:    "test-api-key",
		ProjectID: "test-project-id",
		Endpoint:  "https://test-endpoint.treblle.com", // Use a test endpoint
	})

	// Create a test request
	req, err := http.NewRequest("GET", "https://example.com/api/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create a test response writer
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	responseBody := []byte(`{"status":"success"}`)
	_, err = w.Write(responseBody)
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	// Test the Shutdown function
	internal.Shutdown(req, w, responseBody, nil)

	// Test with custom options
	errorProvider := models.NewErrorProvider()
	errorProvider.AddCustomError("Test error", models.RuntimeError, "test")
	
	options := &internal.ShutdownOptions{
		AdditionalFieldsToMask: []string{"password", "token"},
		ErrorProvider:          errorProvider,
	}
	
	internal.Shutdown(req, w, responseBody, options)

	// Test ShutdownWithCustomData
	// Create headers for request
	reqHeaders := make(map[string]interface{})
	for k, v := range req.Header {
		if len(v) == 1 {
			reqHeaders[k] = v[0]
		} else {
			reqHeaders[k] = v
		}
	}
	reqHeadersJson, _ := json.Marshal(reqHeaders)
	
	// Create headers for response
	respHeaders := make(map[string]interface{})
	for k, v := range w.Header() {
		if len(v) == 1 {
			respHeaders[k] = v[0]
		} else {
			respHeaders[k] = v
		}
	}
	respHeadersJson, _ := json.Marshal(respHeaders)
	
	requestInfo := models.RequestInfo{
		Timestamp: time.Now().Format(time.RFC3339),
		Ip:        "127.0.0.1",
		Url:       "https://example.com/api/test",
		UserAgent: "Test User Agent",
		Method:    "GET",
		Headers:   json.RawMessage(reqHeadersJson),
	}
	
	responseInfo := models.ResponseInfo{
		Code:     200,
		Size:     len(responseBody),
		LoadTime: 10.5,
		Headers:  json.RawMessage(respHeadersJson),
		Errors:   []models.ErrorInfo{},
	}
	
	internal.ShutdownWithCustomData(requestInfo, responseInfo, errorProvider)
}

func TestGracefulShutdown(t *testing.T) {
	// Configure Treblle with batch error collector
	internal.Configure(internal.Configuration{
		ApiKey:            "test-api-key",
		ProjectID:         "test-project-id",
		Endpoint:          "https://test-endpoint.treblle.com",
		BatchErrorEnabled: true,
		BatchErrorSize:    10,
		BatchFlushInterval: 5 * time.Second,
	})
	

	internal.GracefulShutdown()
	

}
