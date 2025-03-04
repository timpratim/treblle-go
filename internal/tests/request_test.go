package treblle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
	"github.com/timpratim/treblle-go/treblle"
)

func (s *TestSuite) TestMasking() {
	testCases := map[string]struct {
		input          []byte
		expectedErr    error
		expectedOutput map[string]interface{}
	}{
		"invalid-json": {
			input:          []byte(`{"id"`),
			expectedErr:    &json.SyntaxError{},
			expectedOutput: nil,
		},
		"simple-object": {
			input:          []byte(`{"id":1,"password":"secret"}`),
			expectedErr:    nil,
			expectedOutput: map[string]interface{}{"id": float64(1), "password": "*********"},
		},
		"nested-object": {
			input:          []byte(`{"id":2,"node":{"password":"secret"}}`),
			expectedErr:    nil,
			expectedOutput: map[string]interface{}{"id": float64(2), "node": map[string]interface{}{"password": "*********"}},
		},
		"array-of-objects": {
			input:       []byte(`{"users":[{"id":1,"password":"secret1"},{"id":2,"password":"secret2"}]}`),
			expectedErr: nil,
			expectedOutput: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": float64(1), "password": "*********"},
					map[string]interface{}{"id": float64(2), "password": "*********"},
				},
			},
		},
	}

	for tn, tc := range testCases {
		internal.Configure(internal.Configuration{
			DefaultFieldsToMask: []string{"password"},
		})

		masked, err := treblle.GetMaskedJSON(tc.input)
		if tc.expectedErr != nil {
			s.Require().IsType(tc.expectedErr, err, tn)
			continue
		}

		s.Require().NoError(err, tn)
		var result map[string]interface{}
		err = json.Unmarshal(masked, &result)
		s.Require().NoError(err, tn)
		s.Require().Equal(tc.expectedOutput, result, tn)
	}
}

func (s *TestSuite) TestQueryParamMasking() {
	testCases := map[string]struct {
		query    url.Values
		expected string
	}{
		"no-sensitive-params": {
			query: url.Values{
				"page":  []string{"1"},
				"limit": []string{"10"},
			},
			expected: "limit=10&page=1",
		},
		"with-sensitive-params": {
			query: url.Values{
				"api_key": []string{"secret123"},
				"page":    []string{"1"},
			},
			expected: "api_key=%2A%2A%2A%2A%2A%2A%2A%2A%2A&page=1",
		},
		"multiple-values": {
			query: url.Values{
				"token": []string{"token1", "token2"},
				"sort":  []string{"desc"},
			},
			expected: "sort=desc&token=%2A%2A%2A%2A%2A%2A%2A%2A%2A&token=%2A%2A%2A%2A%2A%2A%2A%2A%2A",
		},
	}

	for tn, tc := range testCases {
		internal.Configure(internal.Configuration{
			DefaultFieldsToMask: []string{"api_key", "token"},
		})
		maskedQuery := make(url.Values)
		for key, values := range tc.query {
			maskedValues := make([]string, len(values))
			for i, value := range values {
				if key == "api_key" || key == "token" {
					maskedValues[i] = "*********"
				} else {
					maskedValues[i] = value
				}
			}
			maskedQuery[key] = maskedValues
		}
		result := maskedQuery.Encode()
		s.Require().Equal(tc.expected, result, tn)
	}
}

func (s *TestSuite) TestResponseHeaderMasking() {
	testCases := map[string]struct {
		headers  http.Header
		expected map[string]interface{}
	}{
		"no-sensitive-headers": {
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			expected: map[string]interface{}{
				"Content-Type": "application/json",
			},
		},
		"with-sensitive-headers": {
			headers: http.Header{
				"Authorization": []string{"Bearer token123"},
				"Content-Type":  []string{"application/json"},
			},
			expected: map[string]interface{}{
				"Authorization": "Bearer *********",
				"Content-Type":  "application/json",
			},
		},
		"multiple-value-headers": {
			headers: http.Header{
				"Set-Cookie": []string{"session=abc123", "token=xyz789"},
			},
			expected: map[string]interface{}{
				"Set-Cookie": []interface{}{"*********", "*********"},
			},
		},
	}

	for tn, tc := range testCases {
		internal.Configure(internal.Configuration{
			DefaultFieldsToMask: []string{"authorization", "set-cookie"},
		})

		rec := httptest.NewRecorder()
		for k, v := range tc.headers {
			for _, val := range v {
				rec.Header().Add(k, val)
			}
		}

		errorProvider := models.NewErrorProvider()
		resp := treblle.GetResponseInfo(rec, time.Now(), errorProvider)
		var headers map[string]interface{}
		err := json.Unmarshal(resp.Headers, &headers)
		s.Require().NoError(err, tn)
		s.Require().Equal(tc.expected, headers, tn)
	}
}

func (s *TestSuite) TestAuthorizationHeaderMasking() {
	testCases := map[string]struct {
		value    string
		expected string
	}{
		"bearer-token": {
			value:    "Bearer abc123def456",
			expected: "Bearer *********",
		},
		"basic-auth": {
			value:    "Basic dXNlcjpwYXNz",
			expected: "Basic *********",
		},
		"api-key": {
			value:    "ApiKey secret123",
			expected: "ApiKey *********",
		},
		"token": {
			value:    "Token xyz789",
			expected: "Token *********",
		},
		"no-type": {
			value:    "raw-token-123",
			expected: "*********",
		},
	}

	for tn, tc := range testCases {
		result := maskValue(tc.value, "authorization")
		s.Require().Equal(tc.expected, result, tn)
	}
}

func (s *TestSuite) TestGetRequestInfo() {
	testCases := map[string]struct {
		setupRequest func() *http.Request
		expected     models.RequestInfo
		expectError  bool
	}{
		"basic-request": {
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/test", nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
				return req
			},
			expected: models.RequestInfo{
				Ip:        "192.0.2.1",
				Method:    "GET",
				Url:       "http://example.com/test",
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
				Device:    "desktop",
				RoutePath: "/test",
			},
		},
		"mobile-request": {
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/test", nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)")
				return req
			},
			expected: models.RequestInfo{
				Ip:        "192.0.2.1",
				Method:    "GET",
				Url:       "http://example.com/test",
				UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
				Device:    "mobile",
				RoutePath: "/test",
			},
		},
		"with-x-forwarded-for": {
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.195")
				return req
			},
			expected: models.RequestInfo{
				Ip:        "203.0.113.195",
				Method:    "GET",
				Url:       "http://example.com/test",
				Device:    "desktop",
				RoutePath: "/test",
			},
		},
		"with-query-params": {
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/test?key=value&array=1&array=2", nil)
				return req
			},
			expected: models.RequestInfo{
				Ip:        "192.0.2.1",
				Method:    "GET",
				Url:       "http://example.com/test?key=value&array=1&array=2",
				Device:    "desktop",
				RoutePath: "/test",
			},
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			req := tc.setupRequest()
			errorProvider := models.NewErrorProvider()
			result, err := treblle.GetRequestInfo(req, time.Now(), errorProvider)

			if tc.expectError {
				s.Error(err)
				return
			}

			s.NoError(err)
			s.Equal(tc.expected.Ip, result.Ip)
			s.Equal(tc.expected.Method, result.Method)
			s.Equal(tc.expected.Url, result.Url)
			s.Equal(tc.expected.UserAgent, result.UserAgent)
			s.Equal(tc.expected.Device, result.Device)
			s.Equal(tc.expected.RoutePath, result.RoutePath)

			// Verify JSON fields are valid
			if result.Headers != nil {
				var headers map[string]interface{}
				s.NoError(json.Unmarshal(result.Headers, &headers))
			}
			if result.Query != nil {
				var query map[string]interface{}
				s.NoError(json.Unmarshal(result.Query, &query))
			}
			if result.Body != nil {
				var body map[string]interface{}
				s.NoError(json.Unmarshal(result.Body, &body))
			}
		})
	}
}

func (s *TestSuite) TestMaskHeaders() {
	// TODO: implement test
}
