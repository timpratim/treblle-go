package treblle

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

const (
	treblleVersion = SDKVersion // Use centralized version
	sdkName        = SDKName    // Use centralized name
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		startTime := time.Now()

		// Get request info before processing
		requestInfo, errReqInfo := getRequestInfo(r, startTime, errorProvider)
		if errReqInfo != nil && !errors.Is(errReqInfo, ErrNotJson) {
			errorProvider.AddError(errReqInfo, RequestError, "request_processing")
		}

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
			errorProvider.AddError(err, ResponseError, "response_writing")
			return
		}

		// Send to Treblle if:
		// 1. The request was valid JSON (or had no body)
		// OR
		// 2. The response is JSON (regardless of status code)
		if !errors.Is(errReqInfo, ErrNotJson) || rec.Header().Get("Content-Type") == "application/json" {
			responseInfo := getResponseInfo(rec, startTime, errorProvider)

			// Add all collected errors to the response
			responseInfo.Errors = errorProvider.GetErrors()

			// Create metadata
			ti := MetaData{
				ApiKey:    Config.APIKey,
				ProjectID: Config.ProjectID,
				Version:   treblleVersion,
				Sdk:       sdkName,
				Data: DataInfo{
					Server:   Config.serverInfo,
					Language: Config.languageInfo,
					Request:  requestInfo,
					Response: responseInfo,
				},
			}

			// Don't block execution while sending data to Treblle
			go func() {
				defer func() {
					if err := recover(); err != nil {
						// Silently recover from panic
					}
				}()
				sendToTreblle(ti)
			}()
		}
	})
}
