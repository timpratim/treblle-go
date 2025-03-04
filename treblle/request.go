package treblle

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

var ErrNotJson = errors.New("request body is not JSON")

// Get details about the request
func GetRequestInfo(r *http.Request, startTime time.Time, errorProvider *models.ErrorProvider) (models.RequestInfo, error) {
	// Get headers
	headers := make(map[string]string)
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}

	// Get query parameters
	queryParams := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			queryParams[key] = values[0]
		} else {
			queryParams[key] = values
		}
	}

	// Get body if present
	var bodyData map[string]interface{}
	if r.Body != nil && r.Body != http.NoBody {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			errorProvider.AddError(err, models.RequestError, "body_reading")
			return models.RequestInfo{}, err
		}
		// Restore the body for downstream handlers
		defer recoverBody(r, io.NopCloser(bytes.NewBuffer(buf)))

		if err := json.Unmarshal(buf, &bodyData); err != nil {
			// If it's not JSON, return ErrNotJson
			if _, ok := err.(*json.SyntaxError); ok {
				return models.RequestInfo{}, ErrNotJson
			}
			// For other errors, add to error provider but continue
			errorProvider.AddError(err, models.RequestError, "body_parsing")
		}
	}

	// Get client IP - prefer X-Forwarded-For if available
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
		if i := strings.LastIndex(ip, ":"); i != -1 {
			ip = ip[:i]
		}
	}
	if ip == "" {
		ip = "bogon" // Match Laravel's fallback
	}

	// Get route path from router or fallback to URL path
	routePath := r.URL.Path
	if route := mux.CurrentRoute(r); route != nil {
		if pattern, err := route.GetPathTemplate(); err == nil {
			routePath = pattern
		}
	}

	// Detect device type
	device := "desktop"
	if isMobile(r.UserAgent()) {
		device = "mobile"
	}

	// Sanitize headers
	headersJson, err := json.Marshal(headers)
	if err != nil {
		return models.RequestInfo{}, err
	}

	// Sanitize query parameters
	queryJson, err := json.Marshal(queryParams)
	if err != nil {
		return models.RequestInfo{}, err
	}

	// Sanitize body
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		return models.RequestInfo{}, err
	}

	return models.RequestInfo{
		Timestamp: startTime.UTC().Format("2006-01-02 15:04:05"),
		Ip:        ip,
		Url:       r.URL.String(),
		Method:    r.Method,
		Headers:   json.RawMessage(headersJson),
		Body:      json.RawMessage(bodyJson),
		Query:     json.RawMessage(queryJson),
		UserAgent: r.UserAgent(),
		RoutePath: routePath,
		Device:    device,
	}, nil
}

func recoverBody(r *http.Request, bodyReaderCopy io.ReadCloser) {
	r.Body = bodyReaderCopy
}

// GetMaskedJSON masks sensitive data in JSON payloads
func GetMaskedJSON(payloadToMask []byte) (json.RawMessage, error) {
	var data interface{}
	if err := json.Unmarshal(payloadToMask, &data); err != nil {
		// For testing, preserve the original JSON error
		if _, ok := err.(*json.SyntaxError); ok {
			return nil, err
		}
		return nil, ErrNotJson
	}

	sanitizedData := maskData(data)
	jsonData, err := json.Marshal(sanitizedData)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(jsonData), nil
}

// maskData handles masking of any JSON data type
func maskData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return maskMap(v)
	case []interface{}:
		return maskArray(v)
	case string:
		return v
	default:
		return v
	}
}

// maskMap handles masking of JSON objects
func maskMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		// Check if this key should be masked
		if _, exists := internal.Config.FieldsMap[strings.ToLower(key)]; exists {
			switch v := value.(type) {
			case string:
				result[key] = maskValue(v, key)
			case []interface{}:
				// If it's an array of strings, mask each element
				strArray := make([]string, len(v))
				for i, elem := range v {
					if str, ok := elem.(string); ok {
						strArray[i] = maskValue(str, key)
					}
				}
				result[key] = strArray
			default:
				// For non-string values that need masking, convert to JSON string and mask
				if jsonStr, err := json.Marshal(v); err == nil {
					result[key] = strings.Repeat("*", len(string(jsonStr)))
				} else {
					result[key] = "****"
				}
			}
		} else {
			// If key doesn't need masking, recursively process its value
			result[key] = maskData(value)
		}
	}
	return result
}

// maskArray handles masking of JSON arrays
func maskArray(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, value := range data {
		result[i] = maskData(value)
	}
	return result
}

func maskValue(valueToMask string, key string) string {
	if !shouldMaskField(key) {
		return valueToMask
	}

	// Handle authorization header specially
	if strings.ToLower(key) == "authorization" {
		parts := strings.Fields(valueToMask)
		if len(parts) > 1 {
			// Keep the auth type (e.g., "Bearer") but mask the token
			return parts[0] + " " + strings.Repeat("*", 9)
		}
	}

	return strings.Repeat("*", 9)
}

// shouldMaskField checks if a field should be masked based on the configuration
func shouldMaskField(field string) bool {
	_, exists := internal.Config.FieldsMap[strings.ToLower(field)]
	return exists
}

// isMobile checks if the user agent string indicates a mobile device
func isMobile(userAgent string) bool {
	mobilePatterns := []string{
		"Mobile", "Android", "iPhone", "iPad", "Windows Phone",
		"webOS", "BlackBerry", "iPod",
	}

	userAgent = strings.ToLower(userAgent)
	for _, pattern := range mobilePatterns {
		if strings.Contains(userAgent, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
