package treblle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/timpratim/treblle-go/internal"
)

func TestSDKVersioning(t *testing.T) {
	// Initialize with default configuration
	internal.Configure(internal.Configuration{
		ApiKey:    "test-api-key",
		ProjectID: "test-project-id",
	})

	// Ensure default version is correct
	assert.Equal(t, "go", internal.Config.SDKName)
	assert.Equal(t, "1.0.0", internal.Config.SDKVersion)

	// Test GetSDKInfo function
	info := internal.GetSDKInfo()
	assert.Equal(t, "go", info["SDK Name"])
	assert.Equal(t, "1.0.0", info["SDK Version"])

	// Set environment variables and reconfigure
	os.Setenv("TREBLLE_SDK_VERSION", "2.0.0")
	internal.Configure(internal.Configuration{})

	// Check if version updates
	assert.Equal(t, "2.0.0", internal.Config.SDKVersion)

	// Clean up
	os.Unsetenv("TREBLLE_SDK_VERSION")
}
