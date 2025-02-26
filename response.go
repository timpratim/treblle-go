package treblle

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"time"
)

// Define the maximum response size (2MB in bytes)
const maxResponseSize = 2 * 1024 * 1024

type ResponseInfo struct {
	Headers  json.RawMessage `json:"headers"`
	Code     int             `json:"code"`
	Size     int             `json:"size"`
	LoadTime float64         `json:"load_time"`
	Body     json.RawMessage `json:"body"`
	Errors   []ErrorInfo     `json:"errors"`
}

// Extract information from the response recorder
func getResponseInfo(response *httptest.ResponseRecorder, startTime time.Time, errorProvider *ErrorProvider) ResponseInfo {

	re := ResponseInfo{
		Code:     response.Code,
		Size:     len(response.Body.Bytes()),
		LoadTime: float64(time.Since(startTime).Microseconds()),
		Errors:   []ErrorInfo{},
	}

	// Handle response body
	if bodyBytes := response.Body.Bytes(); len(bodyBytes) > 0 {
		// Check if response body size exceeds 2MB
		if len(bodyBytes) > maxResponseSize {
			// Replace with empty JSON object
			re.Body = json.RawMessage("{}")
			// Set size to 0 as we're not sending the actual body
			re.Size = 0
			// Add an error log for exceeding response size
			errorProvider.AddCustomError(
				"JSON response size is over 2MB",
				ResponseError,
				"response_size_limit",
			)
		} else {
			// Try to mask if it's JSON
			sanitizedBody, err := getMaskedJSON(bodyBytes)
			if err != nil {
				// For non-JSON responses, just store the raw body as a JSON string
				if errors.Is(err, ErrNotJson) {
					// Create a JSON-encoded string without extra quotes
					jsonBytes, err := json.Marshal(string(bodyBytes))
					if err != nil {
						errorProvider.AddError(err, ResponseError, "body_encoding")
					} else {
						re.Body = json.RawMessage(jsonBytes)
					}
				} else {
					errorProvider.AddCustomError(err.Error(), ResponseError, "body_masking")
					jsonBytes, err := json.Marshal(string(bodyBytes))
					if err != nil {
						errorProvider.AddError(err, ResponseError, "body_encoding")
					} else {
						re.Body = json.RawMessage(jsonBytes)
					}
				}
			} else {
				re.Body = sanitizedBody
			}
		}
	}

	// Handle response headers
	headers := make(map[string]interface{})
	for k, v := range response.Header() {
		if len(v) == 1 {
			if shouldMaskHeader(k) {
				if strings.ToLower(k) == "authorization" {
					parts := strings.SplitN(v[0], " ", 2)
					if len(parts) == 2 {
						headers[k] = parts[0] + " " + strings.Repeat("*", 9)
					} else {
						headers[k] = strings.Repeat("*", 9)
					}
				} else {
					headers[k] = strings.Repeat("*", 9)
				}
			} else {
				headers[k] = v[0]
			}
		} else {
			if shouldMaskHeader(k) {
				masked := make([]string, len(v))
				for i := range v {
					masked[i] = strings.Repeat("*", 9)
				}
				headers[k] = masked
			} else {
				headers[k] = v
			}
		}
	}

	headersJson, err := json.Marshal(headers)
	if err != nil {
		errorProvider.AddError(err, ResponseError, "header_encoding")
		return re
	}
	re.Headers = json.RawMessage(headersJson)

	return re
}

// Helper function to check if a header should be masked
func shouldMaskHeader(headerName string) bool {
	// Convert header name to lowercase for consistent matching
	headerName = strings.ToLower(headerName)

	// Check direct match
	if _, exists := Config.FieldsMap[headerName]; exists {
		return true
	}

	// Check with common prefixes
	prefixes := []string{"x-", "x_"}
	for _, prefix := range prefixes {
		if _, exists := Config.FieldsMap[prefix+headerName]; exists {
			return true
		}
	}

	return false
}
