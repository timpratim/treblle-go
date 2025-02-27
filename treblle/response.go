package treblle

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/timpratim/treblle-go/models"
)

// Define the maximum response size (2MB in bytes)
const MaxResponseSize = 2 * 1024 * 1024

// GetResponseInfo extracts information from the HTTP response
func GetResponseInfo(response *httptest.ResponseRecorder, startTime time.Time, errorProvider *models.ErrorProvider) models.ResponseInfo {

	re := models.ResponseInfo{
		Code:     response.Code,
		Size:     len(response.Body.Bytes()),
		LoadTime: float64(time.Since(startTime).Microseconds()),
		Errors:   []models.ErrorInfo{},
	}

	// Process headers
	headers := make(map[string]interface{})
	for k, v := range response.Header() {
		if len(v) > 1 {
			// For multiple values, create an array
			values := make([]interface{}, len(v))
			for i, val := range v {
				values[i] = val
			}
			headers[k] = values
		} else if len(v) == 1 {
			headers[k] = v[0]
		}
	}

	// Mask sensitive headers
	for k, v := range headers {
		lowerKey := strings.ToLower(k)
		if lowerKey == "authorization" || lowerKey == "cookie" || lowerKey == "set-cookie" {
			// For authorization headers, preserve the auth type if present
			if lowerKey == "authorization" {
				if strValue, ok := v.(string); ok {
					parts := strings.SplitN(strValue, " ", 2)
					if len(parts) == 2 {
						headers[k] = parts[0] + " " + strings.Repeat("*", 9)
						continue
					}
				}
			}
			
			// For array values (like multiple Set-Cookie headers)
			if arrValue, ok := v.([]interface{}); ok {
				maskedValues := make([]interface{}, len(arrValue))
				for i := range arrValue {
					maskedValues[i] = strings.Repeat("*", 9)
				}
				headers[k] = maskedValues
			} else {
				// For single values
				headers[k] = strings.Repeat("*", 9)
			}
		}
	}

	// Convert headers to JSON
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		errorProvider.AddError(err, models.ResponseError, "headers_processing")
	} else {
		re.Headers = json.RawMessage(headersJSON)
	}

	// Process body
	if response.Body.Len() > 0 {
		// Check if response is JSON
		var bodyObj interface{}
		body := response.Body.Bytes()
		err := json.Unmarshal(body, &bodyObj)

		if err != nil {
			// If not JSON, just store as is if under size limit
			if len(body) <= MaxResponseSize {
				// For non-JSON responses, we need to encode the string as JSON
				jsonString, err := json.Marshal(string(body))
				if err != nil {
					re.Body = json.RawMessage("{}")
				} else {
					re.Body = json.RawMessage(jsonString)
				}
			} else {
				// If over size limit, replace with empty object
				re.Body = json.RawMessage("{}")
				re.Size = 0
				errorProvider.AddCustomError(
					"Response body exceeds maximum size",
					models.ResponseError,
					"response_size_limit",
				)
			}
		} else {
			// If JSON, mask sensitive fields
			if len(body) <= MaxResponseSize {
				// Mask sensitive data in JSON
				maskedBody, err := json.Marshal(maskData(bodyObj))
				if err != nil {
					errorProvider.AddError(err, models.ResponseError, "body_masking")
					re.Body = json.RawMessage("{}")
				} else {
					re.Body = json.RawMessage(maskedBody)
				}
			} else {
				// If over size limit, replace with empty object
				re.Body = json.RawMessage("{}")
				re.Size = 0
				errorProvider.AddCustomError(
					"Response body exceeds maximum size",
					models.ResponseError,
					"response_size_limit",
				)
			}
		}
	} else {
		// Empty body
		re.Body = json.RawMessage("{}")
	}

	return re
}
