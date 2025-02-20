package treblle

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type MetaData struct {
	ApiKey    string   `json:"api_key"`
	ProjectID string   `json:"project_id"`
	Version   string   `json:"version"`
	Sdk       string   `json:"sdk"`
	Data      DataInfo `json:"data"`
}

type DataInfo struct {
	Server   ServerInfo   `json:"server"`
	Language LanguageInfo `json:"language"`
	Request  RequestInfo  `json:"request"`
	Response ResponseInfo `json:"response"`
}

type ServerInfo struct {
	Ip        string `json:"ip"`
	Timezone  string `json:"timezone"`
	Software  string `json:"software"`
	Signature string `json:"signature"`
	Protocol  string `json:"protocol"`
	Os        OsInfo `json:"os"`
}

type OsInfo struct {
	Name         string `json:"name"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
}

type LanguageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Get information about the server environment
func GetServerInfo() ServerInfo {
	// Get local timezone
	_, offset := time.Now().Zone()
	tzInfo := fmt.Sprintf("UTC%+d", offset/3600) // Simplified timezone format like the old SDK

	// Get OS version with timeout
	osVersion := GetOSVersion()

	return ServerInfo{
		Ip:        "127.0.0.1", // Default to localhost, will be updated with actual IP in middleware
		Timezone:  tzInfo,
		Software:  runtime.Version(),
		Signature: "Treblle Go SDK",
		Protocol:  "HTTP/1.1",
		Os:        GetOSInfo(osVersion),
	}
}

// GetOSVersion returns the OS version with a timeout
func GetOSVersion() string {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Prepare command based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "sw_vers", "-productVersion")
	case "linux":
		cmd = exec.CommandContext(ctx, "uname", "-r")
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "ver")
	default:
		return "unknown"
	}

	// Run command with timeout
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Clean and return output
	return strings.TrimSpace(string(out))
}

// GetOSInfo returns information about the operating system that is running on the server
func GetOSInfo(version string) OsInfo {
	return OsInfo{
		Name:         runtime.GOOS,
		Release:      version,
		Architecture: runtime.GOARCH,
	}
}

func GetLanguageInfo() LanguageInfo {
	return LanguageInfo{
		Name:    "go",
		Version: runtime.Version(),
	}
}
