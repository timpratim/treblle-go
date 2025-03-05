package treblle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RequestInfo struct {
	Timestamp string          `json:"timestamp"`
	Ip        string          `json:"ip"`
	Url       string          `json:"url"`        // This will now contain the normalized route path
	FullUrl   string          `json:"full_url"`   // Add a field to store the actual full URL
	RoutePath string          `json:"route_path"` // Keep the route path for compatibility
	UserAgent string          `json:"user_agent"`
	Method    string          `json:"method"`
	Headers   json.RawMessage `json:"headers"`
	Body      json.RawMessage `json:"body"`
	Query     json.RawMessage `json:"query"`
}

var ErrNotJson = errors.New("request body is not JSON")

// SetRoutePath sets a custom route path for a request context
// This can be called before the middleware to set the route pattern for frameworks
// that don't automatically expose their route templates
func SetRoutePath(r *http.Request, pattern string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, routePathKey, pattern)
	return r.WithContext(ctx)
}

// RoutePathKey is the context key for storing route paths
type routePathKeyType struct{}

var routePathKey = routePathKeyType{}

// GetRoutePath returns the route path from the request context
func GetRoutePath(r *http.Request) string {
	if ctxValue := r.Context().Value(routePathKey); ctxValue != nil {
		return ctxValue.(string)
	}
	return ""
}

// Get details about the request
func getRequestInfo(r *http.Request, startTime time.Time, errorProvider *ErrorProvider) (RequestInfo, error) {
	// Format timestamp to match Laravel (Y-m-d H:i:s)
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")

	// Get client IP with same fallback logic as Laravel
	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		ip = strings.TrimSpace(ips[0])
	}
	if ip == "" {
		ip = "bogon" // Laravel fallback
	}

	// Build full URL including query parameters (matching Laravel's Request::fullUrl())
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.String())

	// Get route path with better fallback
	routePath := GetRoutePath(r)
	if routePath == "" {
		routePath = r.URL.Path
	}
	
	// Normalize the route path to ensure it works with Treblle's endpoint grouping
	routePath = normalizeRoutePath(routePath)

	// Process headers (similar to Laravel's collect()->first())
	headers := make(map[string]interface{})
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	headerJSON, err := json.Marshal(headers)
	if err != nil {
		return RequestInfo{}, fmt.Errorf("failed to marshal headers: %w", err)
	}

	// Process query parameters
	var queryJSON []byte
	queryParams := r.URL.Query()
	if len(queryParams) > 0 {
		maskedQueryStr := getMaskedQueryString(queryParams)
		queryJSON = []byte(fmt.Sprintf("{%q: %q}", "query", maskedQueryStr))
	} else {
		queryJSON = []byte("{}")
	}

	// Process body
	var bodyJSON json.RawMessage
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return RequestInfo{}, fmt.Errorf("failed to read body: %w", err)
		}
		// Restore body for downstream handlers
		r.Body = io.NopCloser(strings.NewReader(string(body)))

		if len(body) > 0 {
			maskedBody, err := getMaskedJSON(body)
			if err != nil {
				if err == ErrNotJson {
					errorProvider.AddCustomError(
						"Request body is not valid JSON",
						ValidationError,
						"getRequestInfo",
					)
					bodyJSON = json.RawMessage("{}")
				} else {
					return RequestInfo{}, fmt.Errorf("failed to mask body: %w", err)
				}
			} else {
				bodyJSON = maskedBody
			}
		}
	}

	// Format the URL like Laravel does but with our route path
	// This creates a URL that looks like a real URL but with route parameters
	// Reuse scheme variable already declared above
	endpointUrl := fmt.Sprintf("%s://%s%s", scheme, r.Host, routePath)
	
	return RequestInfo{
		Timestamp: timestamp,
		Ip:        ip,
		Url:       endpointUrl, // Use endpoint URL with normalized path
		FullUrl:   fullURL,     // Keep the full URL for reference
		RoutePath: routePath,   // Keep route path for compatibility
		UserAgent: r.UserAgent(),
		Method:    r.Method,
		Headers:   headerJSON,
		Body:      bodyJSON,
		Query:     queryJSON,
	}, nil
}

func recoverBody(r *http.Request, bodyReaderCopy io.ReadCloser) {
	r.Body = bodyReaderCopy
}

// normalizeRoutePath converts dynamic route segments to a consistent format
// This helps Treblle to properly group requests under the same endpoint
func normalizeRoutePath(path string) string {
	// Already has route parameters in the expected format
	if strings.Contains(path, ":") || strings.Contains(path, "{") {
		return path
	}
	
	// Convert simple numeric segments to parameter placeholders
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		// Check if segment is a numeric ID
		if _, err := fmt.Sscanf(segment, "%d", new(int)); err == nil {
			segments[i] = ":id"
		} else if len(segment) >= 20 && isUUID(segment) {
			// Looks like a UUID
			segments[i] = ":uuid"
		}
	}
	
	return strings.Join(segments, "/")
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
	// Simple UUID check - could be improved for specific formats
	return strings.Count(s, "-") >= 3 && len(s) >= 32
}
