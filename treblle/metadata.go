package treblle

import (
	"context"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"strconv"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

type MetaData struct {
	ApiKey    string          `json:"api_key"`
	ProjectID string          `json:"project_id"`
	Version   string          `json:"version"`
	Sdk       string          `json:"sdk"`
	Data      models.DataInfo `json:"data"`
}

// Get information about the server environment
func GetServerInfo(r *http.Request) internal.ServerInfo {
	// Get OS information
	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "macOS"
	case "windows":
		osName = "Windows"
	case "linux":
		osName = "Linux"
	default:
		osName = runtime.GOOS
	}

	// Create OsInfo struct
	osInfo := internal.OsInfo{
		Name:         osName,
		Release:      runtime.Version(),
		Architecture: runtime.GOARCH,
	}

	// Get server software information
	var serverSoftware string
	if r != nil && r.Header.Get("Server") != "" {
		serverSoftware = r.Header.Get("Server")
	} else {
		serverSoftware = "Go/" + runtime.Version()
	}

	// Get IP address
	var ip string
	if r != nil {
		ip = SelectFirstValidIPv4(r.RemoteAddr)
	} else {
		// Try to get the local IP address
		ip = getLocalIP()
	}

	return internal.ServerInfo{
		Ip:       ip,
		Timezone: time.Local.String(),
		Software: serverSoftware,
		Os:       osInfo,
		Protocol: GetFullProtocol(r),
	}
}

// getLocalIP attempts to determine the local IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

// SelectFirstValidIPv4 extracts the first valid IPv4 address from a comma-separated list
// or from a single address with port
func SelectFirstValidIPv4(input string) string {
	if input == "" {
		return "127.0.0.1" // Return localhost IP for empty input
	}

	// Handle X-Forwarded-For style comma-separated list
	if strings.Contains(input, ",") {
		ips := strings.Split(input, ",")
		for _, ip := range ips {
			trimmedIP := strings.TrimSpace(ip)
			if isValidIPv4(trimmedIP) {
				return trimmedIP
			}
		}
	}

	// Handle IP:port format
	if strings.Contains(input, ":") {
		host, _, err := net.SplitHostPort(input)
		if err == nil && isValidIPv4(host) {
			return host
		}
	}

	// Check if the input itself is a valid IPv4
	if isValidIPv4(input) {
		return input
	}

	// If no valid IPv4 found but we have an IPv6, return the first one
	if strings.Contains(input, ":") {
		// This is likely an IPv6 address
		return strings.TrimSpace(strings.Split(input, ",")[0])
	}

	// Return the first non-empty value even if not a valid IP
	if input != "" {
		return strings.TrimSpace(strings.Split(input, ",")[0])
	}

	return "127.0.0.1" // Default to localhost if nothing else works
}

// isValidIPv4 checks if a string is a valid IPv4 address
func isValidIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() != nil
}

// DetectProtocol determines if the request is using http or https
func DetectProtocol(r *http.Request) string {
	if r == nil {
		return "http"
	}

	// Check X-Forwarded-Proto header first (common for proxies)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(proto)
	}

	// Check if the request is TLS
	if r.TLS != nil {
		return "https"
	}

	// Default to http
	return "http"
}

// GetFullProtocol determines the full HTTP protocol (HTTP/1.1, HTTP/2, etc.) from the request
func GetFullProtocol(r *http.Request) string {
	if r == nil {
		return "HTTP/1.1"
	}

	// If Proto is empty but we have valid ProtoMajor/Minor, construct it
	if r.Proto == "" && r.ProtoMajor > 0 {
		return "HTTP/" + strings.Join([]string{
			strconv.Itoa(r.ProtoMajor),
			strconv.Itoa(r.ProtoMinor),
		}, ".")
	}

	// Use the request's Proto field directly
	if r.Proto != "" {
		return r.Proto
	}

	// Default to HTTP/1.1
	return "HTTP/1.1"
}

// GetPHPVersion attempts to get the installed PHP version
func GetPHPVersion(ctx context.Context) string {
	// Try to execute php -v to get version
	cmd := exec.CommandContext(ctx, "php", "-v")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse the output to extract version
	outputStr := string(output)
	if strings.Contains(outputStr, "PHP") {
		lines := strings.Split(outputStr, "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], " ")
			if len(parts) > 1 {
				return parts[1]
			}
		}
	}

	return ""
}
