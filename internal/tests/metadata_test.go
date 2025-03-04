package treblle

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timpratim/treblle-go/treblle"
)

func TestSelectFirstValidIPv4(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "127.0.0.1",
		},
		{
			name:     "single valid IPv4",
			input:    "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "multiple IPv4 addresses",
			input:    "192.168.1.1, 10.0.0.1, 172.16.0.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 and IPv4 mixed",
			input:    "2001:db8::1, 192.168.1.1, 10.0.0.1",
			expected: "192.168.1.1",
		},
		{
			name:     "only IPv6 addresses",
			input:    "2001:db8::1, 2001:db8::2",
			expected: "2001:db8::1", // Returns first one even if not IPv4
		},
		{
			name:     "with spaces and invalid entries",
			input:    " 192.168.1.1 , invalid, 10.0.0.1",
			expected: "192.168.1.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := treblle.SelectFirstValidIPv4(tc.input)
			assert.Equal(t, tc.expected, result, "SelectFirstValidIPv4 returned unexpected result")
		})
	}
}

func TestDetectProtocol(t *testing.T) {
	testCases := []struct {
		name     string
		request  *http.Request
		expected string
	}{
		{
			name:     "nil request",
			request:  nil,
			expected: "http",
		},
		{
			name: "X-Forwarded-Proto header set to https",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Proto", "https")
				return req
			}(),
			expected: "https",
		},
		{
			name: "X-Forwarded-Proto header set to http",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			}(),
			expected: "http",
		},
		{
			name: "TLS request",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com", nil)
				req.TLS = &tls.ConnectionState{}
				return req
			}(),
			expected: "https",
		},
		{
			name: "plain HTTP request",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				return req
			}(),
			expected: "http",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := treblle.DetectProtocol(tc.request)
			assert.Equal(t, tc.expected, result, "DetectProtocol returned unexpected result")
		})
	}
}
