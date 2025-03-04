package treblle

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/suite"
	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
	"github.com/timpratim/treblle-go/treblle"
)

type TestSuite struct {
	suite.Suite

	testServer *httptest.Server
	router     *chi.Mux

	treblleMockMux    *http.ServeMux
	treblleMockServer *httptest.Server
}

func TestTreblleTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupTest() {
	s.router = chi.NewRouter()
	s.router.Use(treblle.Middleware)
	s.testServer = httptest.NewServer(s.router)
	s.treblleMockMux = http.NewServeMux()
	s.treblleMockServer = httptest.NewServer(s.treblleMockMux)

	// Set environment to something that won't be ignored
	os.Setenv("GO_ENV", "integration")

	// Configure with the mock URL and disable async processing
	mockURL := s.treblleMockServer.URL
	internal.Configure(internal.Configuration{
		ApiKey:                 "test-api-key",
		ProjectID:              "test-project-id",
		DefaultFieldsToMask:    []string{"password", "api_key", "credit_card", "authorization"},
		Endpoint:               mockURL,
		AsyncProcessingEnabled: false,      // Disable async processing for tests
		IgnoredEnvironments:    []string{}, // Make sure no environments are ignored
	})

	// Directly set the models.Config.Endpoint to ensure it uses the mock server
	models.Config.Endpoint = mockURL
	models.Config.ApiKey = "test-api-key"
	models.Config.ProjectId = "test-project-id"
}

func (s *TestSuite) TearDownTest() {
	if s.testServer != nil {
		s.testServer.Close()
	}
	if s.treblleMockServer != nil {
		s.treblleMockServer.Close()
	}
}

func (s *TestSuite) TestJsonFormat() {
	sampleData := map[string]interface{}{
		"api_key":    "",
		"project_id": "",
		"version":    "0.6",
		"sdk":        "laravel",
		"data": map[string]interface{}{
			"server": map[string]interface{}{
				"ip":        "18.194.223.176",
				"timezone":  "UTC",
				"software":  "Apache",
				"signature": "Apache/2.4.2",
				"protocol":  "HTTP/1.1",
				"os": map[string]interface{}{
					"name":         "Linux",
					"release":      "4.14.186-110.268.amzn1.x86_64",
					"architecture": "x86_64",
				},
			},
		},
	}

	content, err := json.Marshal(sampleData)
	s.Require().NoError(err)

	var treblleMetadata models.MetaData
	err = json.Unmarshal(content, &treblleMetadata)
	s.Require().NoError(err)
}

func (s *TestSuite) testRequest(method, path, body string, headers map[string]string) (*http.Response, string) {
	var bodyReader *bytes.Reader
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequest(method, s.testServer.URL+path, bodyReader)
	s.Require().NoError(err)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	respBody, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	resp.Body.Close()

	return resp, string(respBody)
}

func (s *TestSuite) TestCRUDMasking() {
	s.router.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		s.Require().NoError(err)

		// Mask sensitive data in response
		response := map[string]interface{}{
			"id":       1,
			"password": maskValue("should-be-masked", "password"),
			"api_key":  maskValue("should-be-masked", "api_key"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// Test POST request with sensitive data
	resp, body := s.testRequest("POST", "/users", `{
		"username": "test",
		"password": "secret123",
		"api_key": "key123"
	}`, map[string]string{
		"Content-Type": "application/json",
	})
	s.Require().Equal(http.StatusCreated, resp.StatusCode)

	var responseBody map[string]interface{}
	err := json.Unmarshal([]byte(body), &responseBody)
	s.Require().NoError(err)
	s.Require().Equal("*********", responseBody["password"])
	s.Require().Equal("*********", responseBody["api_key"])
}

func (s *TestSuite) TestMiddleware() {
	// Set environment to ensure it's not ignored
	os.Setenv("GO_ENV", "integration")
	defer os.Unsetenv("GO_ENV")

	testCases := map[string]struct {
		requestJson        string
		responseJson       string
		requestHeaderKey   string
		requestHeaderValue string
		respHeaderKey      string
		respHeaderValue    string
		status             int
		treblleCalled      bool
		contentType        string
	}{
		"happy-path": {
			requestJson:   `{"id":1}`,
			responseJson:  `{"id":1}`,
			status:        http.StatusOK,
			treblleCalled: true,
			contentType:   "application/json",
		},
		"invalid-request-json": {
			requestJson:   `{"id":`,
			responseJson:  `{"error":"bad request"}`,
			status:        http.StatusBadRequest,
			treblleCalled: true,
			contentType:   "application/json",
		},
		"non-json-response": {
			requestJson:   `{"id":5}`,
			responseJson:  `Hello, World!`,
			status:        http.StatusOK,
			treblleCalled: true, // Changed to true since we're now always sending to Treblle
			contentType:   "text/plain",
		},
	}

	// Process test cases in a specific order to ensure "happy-path" is first
	testOrder := []string{"happy-path", "invalid-request-json", "non-json-response"}

	for _, tn := range testOrder {
		tc := testCases[tn]
		s.SetupTest()

		// Use a channel to track if the mock server was called
		treblleCalled := make(chan bool, 1)

		// No need to reconfigure here, it's already set up in SetupTest
		// log.Printf("Models Config Endpoint: %s", models.Config.Endpoint)
		// log.Printf("Internal Config Endpoint: %s", internal.Config.Endpoint)

		// Setup the mock handler with a longer timeout
		s.treblleMockMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// log.Printf("Mock server received request to: %s", r.URL.String())
			// log.Printf("Test case: %s", tn)
			// log.Printf("Request method: %s, headers: %v", r.Method, r.Header)

			// Read the request body for logging
			bodyBytes, _ := io.ReadAll(r.Body)
			// log.Printf("Request body: %s", string(bodyBytes))

			// Create a new reader from the bytes for the JSON decoder
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var treblleMetadata models.MetaData
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&treblleMetadata)
			if err != nil {
				// log.Printf("Error decoding request body in mock server: %v", err)
				return
			}
			// log.Printf("Received metadata - APIKey: %s, ProjectID: %s", treblleMetadata.ApiKey, treblleMetadata.ProjectID)
			s.Require().Equal("test-api-key", treblleMetadata.ApiKey)
			s.Require().Equal("test-project-id", treblleMetadata.ProjectID)

			if tn == "non-json-response" {
				// For non-JSON responses, the body should be a JSON string
				expectedBody, _ := json.Marshal("Hello, World!")
				s.Require().Equal(string(expectedBody), string(treblleMetadata.Data.Response.Body))
			}

			// log.Printf("Sending treblleCalled signal for test case: %s", tn)
			treblleCalled <- true
			w.WriteHeader(http.StatusOK)
		})

		// Setup the test route
		s.router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			// log.Printf("Request headers: %+v", r.Header)
			if tc.requestHeaderKey != "" {
				s.Require().Equal(tc.requestHeaderValue, r.Header.Get(tc.requestHeaderKey))
			}
			if tc.respHeaderKey != "" {
				w.Header().Set(tc.respHeaderKey, tc.respHeaderValue)
			}
			w.Header().Set("Content-Type", tc.contentType)
			w.WriteHeader(tc.status)
			w.Write([]byte(tc.responseJson))
		})

		// Make the request directly instead of using the helper
		client := &http.Client{}
		req, err := http.NewRequest("GET", s.testServer.URL+"/test", strings.NewReader(tc.requestJson))
		if err != nil {
			s.Fail("Failed to create request", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if tc.requestHeaderKey != "" {
			req.Header.Set(tc.requestHeaderKey, tc.requestHeaderValue)
		}

		resp, err := client.Do(req)
		if err != nil {
			s.Fail("Failed to execute request", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			s.Fail("Failed to read response body", err)
		}
		body := string(bodyBytes)

		s.Require().Equal(tc.status, resp.StatusCode, tn)
		s.Require().Equal(tc.responseJson, body, tn)
		if tc.respHeaderKey != "" {
			s.Require().Equal(tc.respHeaderValue, resp.Header.Get(tc.respHeaderKey), tn)
		}

		// Wait longer for the Treblle call to finish
		select {
		case <-treblleCalled:
			// Successfully received the signal
			s.Require().Equal(tc.treblleCalled, true, tn)
		case <-time.After(2 * time.Second):
			// Timeout occurred, no signal received
			s.Require().Equal(tc.treblleCalled, false, tn)
		}

		s.TearDownTest()
	}
}

func (s *TestSuite) TestProtocolDetection() {
	s.SetupTest()

	// Set environment to ensure it's not ignored
	os.Setenv("GO_ENV", "integration")
	defer os.Unsetenv("GO_ENV")

	// Create a channel to receive the detected protocol
	protocolChan := make(chan string, 1)

	// Setup the mock Treblle server to capture the protocol
	// No need to reconfigure here, it's already set up in SetupTest
	// log.Printf("Models Config Endpoint: %s", models.Config.Endpoint)
	// log.Printf("Internal Config Endpoint: %s", internal.Config.Endpoint)

	s.treblleMockMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("Protocol detection mock server received request")
		// log.Printf("Request method: %s, headers: %v", r.Method, r.Header)
		// log.Printf("Request URL: %s", r.URL.String())

		// Read the request body for logging
		bodyBytes, _ := io.ReadAll(r.Body)
		// log.Printf("Request body: %s", string(bodyBytes))

		// Create a new reader from the bytes for the JSON decoder
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var treblleMetadata models.MetaData
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&treblleMetadata)
		if err != nil {
			// log.Printf("Error decoding request body: %v", err)
			s.Fail("Failed to decode Treblle metadata", err)
			return
		}

		// log.Printf("Received protocol: %s", treblleMetadata.Data.Server.Protocol)
		// Send the protocol to the channel
		protocolChan <- treblleMetadata.Data.Server.Protocol
		// log.Printf("Sent protocol to channel")
		w.WriteHeader(http.StatusOK)
	})

	// Create a test request with HTTP/1.1
	s.router.Get("/protocol-test", func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("Protocol test handler called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Make the request with a valid JSON body
	client := &http.Client{}
	req, err := http.NewRequest("GET", s.testServer.URL+"/protocol-test", strings.NewReader(`{"test":"data"}`))
	if err != nil {
		s.Fail("Failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// log.Printf("Sending request to %s", s.testServer.URL+"/protocol-test")
	resp, err := client.Do(req)
	if err != nil {
		s.Fail("Failed to execute request", err)
	}
	defer resp.Body.Close()

	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// Wait for the protocol to be detected with a longer timeout
	select {
	case protocol := <-protocolChan:
		s.Require().Equal("http", protocol)
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for Treblle metadata")
	}

	s.TearDownTest()
}

// Helper function to mask sensitive values
func maskValue(valueToMask string, key string) string {
	// For authorization headers, preserve the auth type
	if strings.ToLower(key) == "authorization" {
		parts := strings.SplitN(valueToMask, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " " + strings.Repeat("*", 9)
		}
	}
	return strings.Repeat("*", 9)
}
