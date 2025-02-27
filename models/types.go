package models

import (
	"encoding/json"
)

// RequestInfo represents information about an HTTP request
type RequestInfo struct {
	Timestamp string          `json:"timestamp"`
	Ip        string          `json:"ip"`
	Url       string          `json:"url"`
	UserAgent string          `json:"user_agent"`
	Method    string          `json:"method"`
	Protocol  string          `json:"protocol"`
	Headers   json.RawMessage `json:"headers"`
	Body      json.RawMessage `json:"body"`
	Query     json.RawMessage `json:"query"`
}

// ResponseInfo represents information about an HTTP response
type ResponseInfo struct {
	Code     int             `json:"code"`
	Size     int             `json:"size"`
	LoadTime float64         `json:"load_time"`
	Body     json.RawMessage `json:"body"`
	Headers  json.RawMessage `json:"headers"`
	Errors   []ErrorInfo     `json:"errors"`
}

// ErrorInfo represents detailed information about an error
type ErrorInfo struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Context   string `json:"context,omitempty"`
}

// Constants for error types
const (
	// ErrorTypeServer represents server-side errors
	ErrorTypeServer = "ServerError"
	// ErrorTypeClient represents client-side errors
	ErrorTypeClient = "ClientError"
)
