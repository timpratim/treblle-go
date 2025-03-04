package treblle

import (
	"context"
	"net/http"
)

// RoutePathKey is the context key for storing route paths
type routePathKeyType struct{}
var routePathKey = routePathKeyType{}

// SetRoutePath sets a custom route path for a request context
// This can be called before the middleware to set the route pattern for frameworks
// that don't automatically expose their route templates
func SetRoutePath(r *http.Request, pattern string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, routePathKey, pattern)
	return r.WithContext(ctx)
}

// WithRoutePath is a helper function to set the route pattern for a specific handler
// This is useful for non-gorilla/mux routers or custom router implementations
func WithRoutePath(pattern string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the route pattern to the request context
		r = SetRoutePath(r, pattern)
		handler.ServeHTTP(w, r)
	})
}

// HandleFunc is a helper that wraps an http.HandlerFunc with a route pattern
// This simplifies route pattern setting for standard http handlers
func HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) http.Handler {
	return WithRoutePath(pattern, http.HandlerFunc(handler))
}

// GetRoutePathFromContext retrieves a route path from a request context if available
func GetRoutePathFromContext(r *http.Request) (string, bool) {
	pattern, ok := r.Context().Value(routePathKey).(string)
	return pattern, ok && pattern != ""
}