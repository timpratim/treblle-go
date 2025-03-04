package treblle

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

var ErrNotJson = errors.New("request body is not JSON")

// Get details about the request
func GetRequestInfo(r *http.Request, startTime time.Time, errorProvider *models.ErrorProvider) (models.RequestInfo, error) {

	headers := make(map[string]string)
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}

	// Get and mask query parameters
	queryParams := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			queryParams[key] = values[0]
		} else {
			queryParams[key] = values
		}
	}

	// Detect protocol
	protocol := DetectProtocol(r)

	rawPath := r.URL.EscapedPath()
	if rawPath == "" {
		rawPath = r.URL.Path
	}
	// Create URL without query parameters to avoid duplicating them
	baseURL := protocol + "://" + r.Host + rawPath

	// Get client IP - prefer X-Forwarded-For if available
	var ip string
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		// Use the SelectFirstValidIPv4 function directly on the header value
		ip = SelectFirstValidIPv4(forwardedFor)
	} else {
		// Fall back to RemoteAddr
		ip = extractIP(r.RemoteAddr)
	}

	ri := models.RequestInfo{
		Timestamp: startTime.UTC().Format("2006-01-02 15:04:05"),
		Ip:        ip,
		Url:       baseURL,
		RoutePath: r.URL.Path, // Initially set to the actual path. Will be overridden by route pattern if available.
		UserAgent: r.UserAgent(),
		Method:    r.Method,
		Protocol:  protocol,
	}

	// Mask query parameters
	if len(queryParams) > 0 {
		sanitizedQuery, err := json.Marshal(maskData(queryParams))
		if err != nil {
			errorProvider.AddError(err, models.RequestError, "query_masking")
			return ri, err
		}
		ri.Query = json.RawMessage(sanitizedQuery)

		// Add masked query string back to URL
		if queryStr := getMaskedQueryString(r.URL.Query()); queryStr != "" {
			ri.Url = baseURL + "?" + queryStr
		}
	}

	if r.Body != nil && r.Body != http.NoBody {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			errorProvider.AddError(err, models.RequestError, "body_reading")
			return ri, err
		}
		bodyReaderOriginal := io.NopCloser(bytes.NewBuffer(buf))
		defer recoverBody(r, io.NopCloser(bytes.NewBuffer(buf)))

		body, err := io.ReadAll(bodyReaderOriginal)
		if err != nil {
			errorProvider.AddError(err, models.RequestError, "body_reading")
			return ri, err
		}

		sanitizedBody, err := GetMaskedJSON(body)
		if err != nil {
			// If it's not JSON, return ErrNotJson
			if errors.Is(err, ErrNotJson) {
				return ri, ErrNotJson
			}
			// For other errors, add to error provider but continue
			errorProvider.AddError(err, models.RequestError, "body_masking")
			return ri, nil
		}

		ri.Body = sanitizedBody
	}

	headersJson, err := json.Marshal(headers)
	if err != nil {
		return ri, err
	}

	sanitizedHeaders, err := GetMaskedJSON(headersJson)
	if err != nil {
		return ri, err
	}
	ri.Headers = sanitizedHeaders

	return ri, nil
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
		parts := strings.SplitN(valueToMask, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " " + strings.Repeat("*", 9)
		}
		return strings.Repeat("*", 9)
	}

	// For all other fields
	return strings.Repeat("*", 9)
}

// shouldMaskField checks if a field should be masked based on the configuration
func shouldMaskField(field string) bool {
	_, exists := internal.Config.FieldsMap[strings.ToLower(field)]
	return exists
}

func extractIP(remoteAddr string) string {
	var ipAddress string

	// If RemoteAddr contains both IP and port, split and return the IP
	if strings.Contains(remoteAddr, ":") {
		ip, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			ipAddress = ip
		} else {
			ipAddress = remoteAddr
		}
	} else {
		ipAddress = remoteAddr
	}

	// Return the first valid IPv4 address
	return SelectFirstValidIPv4(ipAddress)
}

// getMaskedQueryString returns a masked query string
func getMaskedQueryString(query url.Values) string {
	maskedQuery := make(url.Values)
	for key, values := range query {
		maskedValues := make([]string, len(values))
		for i, value := range values {
			if shouldMaskField(key) {
				maskedValues[i] = strings.Repeat("*", 9)
			} else {
				maskedValues[i] = value
			}
		}
		maskedQuery[key] = maskedValues
	}
	return maskedQuery.Encode()
}
