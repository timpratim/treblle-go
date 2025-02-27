package models

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// MetaData represents the complete data structure sent to Treblle
type MetaData struct {
	ApiKey    string   `json:"api_key"`
	ProjectID string   `json:"project_id"`
	Version   string   `json:"version"`
	Sdk       string   `json:"sdk"`
	Data      DataInfo `json:"data"`
}

// DataInfo contains all the information about a request/response cycle
type DataInfo struct {
	Server   ServerInfo   `json:"server"`
	Language LanguageInfo `json:"language"`
	Request  RequestInfo  `json:"request"`
	Response ResponseInfo `json:"response"`
	Errors   []ErrorInfo  `json:"errors"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Ip        string `json:"ip"`
	Timezone  string `json:"timezone"`
	Software  string `json:"software"`
	Signature string `json:"signature"`
	Protocol  string `json:"protocol"`
	Os        struct {
		Name    string `json:"name"`
		Release string `json:"release"`
		Arch    string `json:"architecture"`
	} `json:"os"`
}

// LanguageInfo contains information about the programming language
type LanguageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Configuration represents Treblle SDK configuration
type Configuration struct {
	ApiKey       string
	ProjectId    string
	Debug        bool
	Endpoint     string
	ServerInfo   ServerInfo
	LanguageInfo LanguageInfo
	SDKVersion   string
	SDKName      string
}

// Config holds the global configuration for the Treblle SDK
var Config Configuration

// SDK version information
const (
	SDKVersion = "0.9.0"
	SDKName    = "go"
)

// GetMaskedJSON masks sensitive data in JSON
func GetMaskedJSON(jsonData []byte) (json.RawMessage, error) {
	// This is a placeholder - in a real implementation, this would mask sensitive fields
	return json.RawMessage(jsonData), nil
}

// NewErrorProvider creates a new ErrorProvider instance
func NewErrorProvider() *ErrorProvider {
	return &ErrorProvider{
		errors: make([]ErrorInfo, 0),
	}
}

// SendToTreblle sends data to Treblle
func SendToTreblle(treblleInfo MetaData) {
	// Use the context-aware version with a default timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	SendToTreblleWithContext(ctx, treblleInfo)
}

// SendToTreblleWithContext sends data to Treblle with context support
func SendToTreblleWithContext(ctx context.Context, treblleInfo MetaData) {
	baseUrl := getTreblleBaseUrl()
	
	bytesRepresentation, err := json.Marshal(treblleInfo)
	if err != nil {
		return
	}

	// Create a request with the provided context
	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl, bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", Config.ApiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// getTreblleBaseUrl returns the base URL for Treblle API
func getTreblleBaseUrl() string {
	if Config.Endpoint != "" {
		return Config.Endpoint
	}
	// Default Treblle endpoints
	treblleBaseUrls := []string{
		"https://rocknrolla.treblle.com",
		"https://punisher.treblle.com",
		"https://sicario.treblle.com",
	}
	return treblleBaseUrls[0]
}

// GetRequestInfo extracts information from an HTTP request
func GetRequestInfo(r *http.Request, startTime time.Time, errorProvider *ErrorProvider) (RequestInfo, error) {
	// This is a placeholder - in a real implementation, this would extract request info
	return RequestInfo{
		Timestamp: time.Now().Format(time.RFC3339),
		Method:    r.Method,
		Url:       r.URL.String(),
	}, nil
}

// Initialize the default configuration
func init() {
	Config = Configuration{
		Debug:      false,
		SDKVersion: SDKVersion,
		SDKName:    SDKName,
		ServerInfo: ServerInfo{
			Os: struct {
				Name    string `json:"name"`
				Release string `json:"release"`
				Arch    string `json:"architecture"`
			}{
				Name:    runtime.GOOS,
				Release: "",
				Arch:    runtime.GOARCH,
			},
		},
		LanguageInfo: LanguageInfo{
			Name:    "go",
			Version: runtime.Version(),
		},
	}
}
