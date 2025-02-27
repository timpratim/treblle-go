package treblle

import (
	"github.com/timpratim/treblle-go/internal"
)

// SDK Versioning (Matching Laravel)
const (
	SDKName    = "go"
	SDKVersion = "1.0.0" // Change this when updating versions
)

// GetSDKInfo returns SDK name and version information
func GetSDKInfo() map[string]string {
	return internal.GetSDKInfo()
}

// GetSDKVersion returns the current SDK version
func GetSDKVersion() string {
	return internal.SDKVersion
}

// GetSDKName returns the SDK name
func GetSDKName() string {
	return internal.SDKName
}
