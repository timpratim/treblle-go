package treblle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
	
	"github.com/gorilla/mux"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the current environment is in the ignored list
		if IsEnvironmentIgnored() {
			// Skip Treblle logging for ignored environments
			next.ServeHTTP(w, r)
			return
		}

		// Create error provider for this request
		errorProvider := NewErrorProvider()
		defer errorProvider.Clear()

		// Recover from panics
		defer func() {
			if err := recover(); err != nil {
				errorProvider.AddCustomError(
					fmt.Sprintf("panic recovered: %v", err),
					UnhandledExceptionError,
					"middleware",
				)
			}
		}()

		// Get the request tracker
		tracker := GetRequestTracker()
		
		// Store start time in request context
		startTime := time.Now()
		r = tracker.StoreStartTime(r)

		// Get request info before processing
		requestInfo, errReqInfo := getRequestInfo(r, startTime, errorProvider)
		if errReqInfo != nil && !errors.Is(errReqInfo, ErrNotJson) {
			errorProvider.AddError(errReqInfo, ValidationError, "request_processing")
		}
		
		// First check if a route path was manually set
		if pattern, ok := r.Context().Value(routePathKey).(string); ok && pattern != "" {
			requestInfo.RoutePath = pattern
		} else {
			// Try to extract route pattern from gorilla/mux if available
			if route := mux.CurrentRoute(r); route != nil {
				if pattern, err := route.GetPathTemplate(); err == nil && pattern != "" {
					requestInfo.RoutePath = pattern
				}
			}
		}
		
		// Normalize the route path to ensure consistent endpoint grouping in Treblle
		requestInfo.RoutePath = normalizeRoutePath(requestInfo.RoutePath)

		// Store request info in context if async processing is enabled
		if Config.AsyncProcessingEnabled {
			r = tracker.StoreRequestInfo(r, requestInfo)
		}

		// Create a copy of the serverInfo with the correct protocol for this request
		serverInfo := Config.serverInfo
		serverInfo.Protocol = DetectProtocol(r)

		// Intercept the response so it can be copied
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		// Copy everything from response recorder to response writer
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)

		// Write response body
		_, err := w.Write(rec.Body.Bytes())
		if err != nil {
			errorProvider.AddError(err, ServerError, "response_writing")
			return
		}

		// Send to Treblle if:
		// 1. The request was valid JSON (or had no body)
		// OR
		// 2. The response is JSON (regardless of status code)
		// OR
		// 3. The response is not JSON (we'll still track it)
		responseInfo := getResponseInfo(rec, startTime, errorProvider)
		
		// Add all collected errors to the response
		responseInfo.Errors = errorProvider.GetErrors()

		if Config.AsyncProcessingEnabled {
			// Process asynchronously with controlled concurrency
			GetAsyncProcessor().Process(requestInfo, responseInfo, errorProvider)
		} else {
			// Create metadata
			ti := MetaData{
				ApiKey:    Config.APIKey,
				ProjectID: Config.ProjectID,
				Version:   Config.SDKVersion,
				Sdk:       Config.SDKName,
				Url:       requestInfo.Url, // Use the normalized URL from requestInfo (critical for endpoint grouping)
				Data: DataInfo{
					Server:   serverInfo, // Use the updated serverInfo with correct protocol
					Language: Config.languageInfo,
					Request:  requestInfo,
					Response: responseInfo,
				},
			}

			// Don't block execution while sending data to Treblle
			go func(ti MetaData) {
				defer func() {
					if err := recover(); err != nil {
						fmt.Printf("Panic recovered in goroutine: %v\n", err)
					// Silently recover from panic
					}
				}()
				sendToTreblle(ti)
			}(ti)
		}
	})
}