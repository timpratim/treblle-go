package treblle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

// Middleware creates a new HTTP middleware handler for Treblle monitoring
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if environment is ignored
		if internal.IsEnvironmentIgnored() {
			// Skip Treblle logging for ignored environments
			next.ServeHTTP(w, r)
			return
		}

		// Create error provider for this request
		errorProvider := models.NewErrorProvider()
		defer errorProvider.Clear()

		// Recover from panics
		defer func() {
			if err := recover(); err != nil {
				errorProvider.AddCustomError(
					fmt.Sprintf("panic recovered: %v", err),
					models.UnhandledExceptionError,
					"middleware",
				)
			}
		}()

		// Get the request tracker
		tracker := internal.GetRequestTracker()
		
		// Store start time in request context
		startTime := time.Now()
		r = tracker.StoreStartTime(r)

		// Get request info before processing
		requestInfo, errReqInfo := GetRequestInfo(r, startTime, errorProvider)
		if errReqInfo != nil && !errors.Is(errReqInfo, ErrNotJson) {
			errorProvider.AddError(errReqInfo, models.RequestError, "request_processing")
		}

		// Store request info in context if async processing is enabled
		if internal.Config.AsyncProcessingEnabled {
			r = tracker.StoreRequestInfo(r, requestInfo)
		}

		// Create a copy of the serverInfo with the correct protocol for this request
		serverInfo := internal.Config.ServerInfo
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
			errorProvider.AddError(err, models.ResponseError, "response_writing")
			return
		}

		// Always process for tests, regardless of JSON content type
		responseInfo := GetResponseInfo(rec, startTime, errorProvider)
		
		// Add all collected errors to the response
		responseInfo.Errors = errorProvider.GetErrors()

		if internal.Config.AsyncProcessingEnabled {
			// Process asynchronously with controlled concurrency
			internal.GetAsyncProcessor().Process(requestInfo, responseInfo, errorProvider)
		} else {
			// Create metadata
			ti := models.MetaData{
				ApiKey:    internal.Config.ApiKey,
				ProjectID: internal.Config.ProjectID,
				Version:   internal.Config.SDKVersion,
				Sdk:       internal.Config.SDKName,
				Data: models.DataInfo{
					Server:   serverInfo, // Use the updated serverInfo with correct protocol
					Language: internal.Config.LanguageInfo,
					Request:  requestInfo,
					Response: responseInfo,
				},
			}

			// Ensure models.Config is synchronized with internal.Config
			models.Config.ApiKey = internal.Config.ApiKey
			models.Config.ProjectId = internal.Config.ProjectID
			models.Config.Endpoint = internal.Config.Endpoint
			
			// Send to Treblle directly for tests
			models.SendToTreblle(ti)
		}
	})
}
